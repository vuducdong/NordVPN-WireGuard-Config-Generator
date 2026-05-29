package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"nordgen/internal/client"
	"nordgen/internal/models"
	"nordgen/internal/ui"
)

const fileNameMaxLength = 15

type FileWrite struct {
	Name    string
	Content []byte
}

type DirJob struct {
	Path  string
	Files []FileWrite
}

type Generator struct {
	client          *client.NordClient
	consoleManager  *ui.ConsoleManager
	Stats           models.GenerationStats
	outputDirectory string
}

func NewGenerator(c *client.NordClient, u *ui.ConsoleManager) *Generator {
	return &Generator{
		client:         c,
		consoleManager: u,
	}
}

func sanitizePathSegment(segment string) string {
	segment = strings.TrimSpace(segment)
	if segment == "" {
		return "unknown"
	}
	segment = strings.ToLower(segment)
	b := make([]byte, 0, len(segment))

	for i := 0; i < len(segment); i++ {
		c := segment[i]
		switch c {
		case '<', '>', ':', '"', '/', '\\', '|', '?', '*', '\x00', ' ':
			b = append(b, '_')
		default:
			b = append(b, c)
		}
	}
	res := string(b)
	for strings.HasSuffix(res, ".") {
		res = res[:len(res)-1]
	}
	if res == "" {
		return "unknown"
	}
	return res
}

func (g *Generator) Process(privateKey string, prefs models.UserPreferences) (string, error) {
	g.outputDirectory = fmt.Sprintf("nordvpn_configs_%s", time.Now().Format("20060102_150405"))

	g.consoleManager.StartStatus("Fetching data...")

	var wgData sync.WaitGroup
	var obsLat, obsLon float64
	var rawServers []models.RawServer
	var errServers error

	wgData.Add(2)
	go func() {
		defer wgData.Done()
		obsLat, obsLon, _ = g.client.GetGeo()
	}()
	go func() {
		defer wgData.Done()
		rawServers, errServers = g.client.GetServers()
	}()
	wgData.Wait()

	g.consoleManager.StopStatus()

	if errServers != nil || len(rawServers) == 0 {
		g.consoleManager.Fail("Failed to fetch server data")
		return "", fmt.Errorf("server data fetch failed")
	}
	g.consoleManager.Success("Fetched server data")

	g.consoleManager.StartStatus("Processing dataset...")

	allParsed := parseServers(rawServers, obsLat, obsLon, prefs.Groups, prefs.ExcludeDedicated)

	uniqueMap := make(map[string]models.Server, len(allParsed))
	for _, s := range allParsed {
		uniqueMap[s.Name] = s
	}

	uniqueServers := make([]models.Server, 0, len(uniqueMap))
	for _, s := range uniqueMap {
		uniqueServers = append(uniqueServers, s)
	}

	if len(uniqueServers) == 0 {
		g.consoleManager.StopStatus()
		g.consoleManager.Fail("No servers found matching the specified filters")
		return "", fmt.Errorf("no servers matched filters")
	}

	sort.Slice(uniqueServers, func(i, j int) bool {
		if uniqueServers[i].Load == uniqueServers[j].Load {
			return uniqueServers[i].Distance < uniqueServers[j].Distance
		}
		return uniqueServers[i].Load < uniqueServers[j].Load
	})

	g.Stats.Total = len(uniqueServers)

	type bestKey struct {
		combo   string
		country string
		city    string
	}
	bestMap := make(map[bestKey]models.Server, len(uniqueServers))
	for _, s := range uniqueServers {
		k := bestKey{s.Combo, s.Country, s.City}
		if _, exists := bestMap[k]; !exists {
			bestMap[k] = s
		}
	}
	g.Stats.Best = len(bestMap)

	bestServers := make([]models.Server, 0, len(bestMap))
	for _, s := range bestMap {
		bestServers = append(bestServers, s)
	}

	dirMap := make(map[string]*DirJob)
	g.buildJobs(uniqueServers, "configs", privateKey, prefs, dirMap)
	g.buildJobs(bestServers, "best_configs", privateKey, prefs, dirMap)

	g.consoleManager.StopStatus()
	g.consoleManager.Success("Dataset processed")

	var dirJobs []*DirJob
	totalFiles := 0
	for _, dj := range dirMap {
		dirJobs = append(dirJobs, dj)
		totalFiles += len(dj.Files)
	}

	errWrite := g.writeJobsParallel(dirJobs, totalFiles)

	if errWrite != nil {
		g.consoleManager.Fail(fmt.Sprintf("Failed to write configuration files: %v", errWrite))
		return "", fmt.Errorf("file write failed: %w", errWrite)
	}

	g.consoleManager.Success("Configuration files written")

	return g.outputDirectory, nil
}

func (g *Generator) buildJobs(servers []models.Server, subDir, privKey string, prefs models.UserPreferences, dirMap map[string]*DirJob) {
	interfaceBlock := []byte("[Interface]\nPrivateKey = " + privKey + "\nAddress = 10.5.0.2/16\nDNS = " + prefs.DNS + "\n\n[Peer]\n")
	keepaliveBlock := []byte("\nPersistentKeepalive = " + strconv.Itoa(prefs.Keepalive))
	pubKeyPrefix := []byte("PublicKey = ")
	allowedIPPrefix := []byte("\nAllowedIPs = 0.0.0.0/0, ::/0\nEndpoint = ")
	endpointSuffix := []byte(":51820")

	counts := make(map[string]int, len(servers))

	for _, s := range servers {
		countrySeg := sanitizePathSegment(s.Country)
		citySeg := sanitizePathSegment(s.City)

		fnameRoot := sanitizePathSegment(s.Name)
		if len(fnameRoot) > fileNameMaxLength {
			fnameRoot = fnameRoot[:fileNameMaxLength]
		}

		dir := filepath.Join(g.outputDirectory, subDir, s.Combo, countrySeg, citySeg)

		dj, exists := dirMap[dir]
		if !exists {
			dj = &DirJob{Path: dir}
			dirMap[dir] = dj
		}

		baseKey := filepath.Join(dir, fnameRoot)
		count := counts[baseKey]
		counts[baseKey] = count + 1

		fname := fnameRoot + ".conf"
		if count > 0 {
			fname = fnameRoot + "_" + strconv.Itoa(count) + ".conf"
		}

		endpoint := s.Hostname
		if prefs.UseIP {
			endpoint = s.Station
		}

		size := len(interfaceBlock) + len(pubKeyPrefix) + len(s.PublicKey) + len(allowedIPPrefix) + len(endpoint) + len(endpointSuffix) + len(keepaliveBlock)
		content := make([]byte, 0, size)
		content = append(content, interfaceBlock...)
		content = append(content, pubKeyPrefix...)
		content = append(content, s.PublicKey...)
		content = append(content, allowedIPPrefix...)
		content = append(content, endpoint...)
		content = append(content, endpointSuffix...)
		content = append(content, keepaliveBlock...)

		dj.Files = append(dj.Files, FileWrite{
			Name:    fname,
			Content: content,
		})
	}
}

func (g *Generator) writeJobsParallel(dirJobs []*DirJob, totalFiles int) error {
	if totalFiles == 0 {
		return nil
	}

	g.consoleManager.StartStatus("Preparing file system...")
	for _, dj := range dirJobs {
		if err := os.MkdirAll(dj.Path, 0755); err != nil {
			return err
		}
	}
	g.consoleManager.StopStatus()
	g.consoleManager.Success("File system prepared")

	g.consoleManager.StartProgress(totalFiles, "Writing configs")

	var completed int32
	var firstErr error
	var errMutex sync.Mutex

	uiDone := make(chan struct{})
	uiExited := make(chan struct{})

	go func() {
		defer close(uiExited)
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				c := atomic.LoadInt32(&completed)
				g.consoleManager.UpdateProgress(int(c), totalFiles, "Writing configs")
			case <-uiDone:
				c := atomic.LoadInt32(&completed)
				g.consoleManager.UpdateProgress(int(c), totalFiles, "Writing configs")
				return
			}
		}
	}()

	workerCount := runtime.NumCPU() * 4
	if workerCount > 64 {
		workerCount = 64
	}
	if workerCount < 4 {
		workerCount = 4
	}
	if workerCount > len(dirJobs) {
		workerCount = len(dirJobs)
	}

	jobChan := make(chan *DirJob, len(dirJobs))
	for _, dj := range dirJobs {
		jobChan <- dj
	}
	close(jobChan)

	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for dj := range jobChan {
				for _, file := range dj.Files {
					fullPath := filepath.Join(dj.Path, file.Name)
					if err := os.WriteFile(fullPath, file.Content, 0644); err != nil {
						errMutex.Lock()
						if firstErr == nil {
							firstErr = err
						}
						errMutex.Unlock()
					}
					atomic.AddInt32(&completed, 1)
				}
			}
		}()
	}

	wg.Wait()
	close(uiDone)
	<-uiExited
	g.consoleManager.StopProgress()

	return firstErr
}

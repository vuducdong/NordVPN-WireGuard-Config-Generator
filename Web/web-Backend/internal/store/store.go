package store

import (
	"bytes"
	"net/http"
	"sort"
	"strconv"
	"sync/atomic"
	"time"
	"unsafe"

	"nordgen/internal/types"
	"nordgen/internal/wg"

	"github.com/andybalholm/brotli"
	"github.com/bytedance/sonic"
)

const (
	API_URL = "https://api.nordvpn.com/v1/servers?limit=16384&filters[servers_technologies][identifier]=wireguard_udp"
	REFRESH = 5 * time.Minute
)

var (
	refreshClient = &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			ForceAttemptHTTP2:   true,
			MaxIdleConns:        10,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}
)

type RegionBoundary struct {
	Hash  uint64
	Start uint32
	End   uint32
}

type NameIndexEntry struct {
	Name [16]byte
	Idx  uint32
}

type State struct {
	AllServers   []types.ProcessedServer
	NameIndex    []NameIndexEntry
	Keys         map[int]string
	Boundaries   []RegionBoundary
	ServerJson   []byte
	ServerJsonBr []byte
	ServerEtag   string
	PeerHostname [][]byte
	PeerStation  [][]byte
}

type Store struct {
	state atomic.Pointer[State]
}

var Core = &Store{}

func (s *Store) Init() {
	s.updateServers()
	go func() {
		ticker := time.NewTicker(REFRESH)
		for range ticker.C {
			s.updateServers()
		}
	}()
}

func (s *Store) LoadState() *State {
	return s.state.Load()
}

func (s *Store) updateServers() {
	resp, err := refreshClient.Get(API_URL)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return
	}

	var raw []types.RawServer
	if err := sonic.ConfigDefault.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return
	}

	state := &State{
		AllServers: make([]types.ProcessedServer, 0, len(raw)),
		Keys:       make(map[int]string, len(raw)),
	}

	newKeys := make(map[string]int, len(raw))
	payload := types.ServerPayload{
		Headers: []string{"name", "load", "station"},
		List:    make(map[string]map[string][][]interface{}),
	}

	strPool := make(map[string]string)
	intern := func(in string) string {
		if val, ok := strPool[in]; ok {
			return val
		}
		strPool[in] = in
		return in
	}

	kID := 1

	for _, srv := range raw {
		if len(srv.Locations) == 0 {
			continue
		}

		ver := "0.0.0"
		for _, sp := range srv.Specifications {
			if sp.ID == "version" && len(sp.Values) > 0 {
				ver = sp.Values[0].Value
				break
			}
		}

		if !validateVersion(ver) {
			continue
		}

		name := normalize(srv.Name)
		loc := srv.Locations[0]
		var pk string
		for _, tech := range srv.Technologies {
			for _, meta := range tech.Metadata {
				if meta.Name == "public_key" {
					pk = meta.Value
					break
				}
			}
		}

		if loc.Country.Code == "" || pk == "" {
			continue
		}

		id, exists := newKeys[pk]
		if !exists {
			id = kID
			kID++
			newKeys[pk] = id
			state.Keys[id] = pk
		}

		country := intern(normalize(loc.Country.Name))
		city := intern(normalize(loc.Country.City.Name))
		code := intern(loc.Country.Code)
		lowCode := intern(toLower(code))

		num := extractNumber(name)
		if num == "" {
			num = "wg"
		}
		fileName := buildFileName(lowCode, num)

		processed := types.ProcessedServer{}
		copyToBytes(processed.Name[:], name)
		copyToBytes(processed.Station[:], srv.Station)
		copyToBytes(processed.Hostname[:], srv.Hostname)
		copyToBytes(processed.Country[:], country)
		copyToBytes(processed.City[:], city)
		copyToBytes(processed.Code[:], code)
		copyToBytes(processed.LowCode[:], lowCode)
		copyToBytes(processed.Number[:], num)
		copyToBytes(processed.FileName[:], fileName)
		processed.KeyID = uint16(id)

		state.AllServers = append(state.AllServers, processed)

		if payload.List[country] == nil {
			payload.List[country] = make(map[string][][]interface{})
		}

		payload.List[country][city] = append(payload.List[country][city], []interface{}{name, srv.Load, srv.Station})
	}

	sort.Slice(state.AllServers, func(i, j int) bool {
		ci := state.AllServers[i].GetCountry()
		cj := state.AllServers[j].GetCountry()
		if ci != cj {
			return ci < cj
		}
		ti := state.AllServers[i].GetCity()
		tj := state.AllServers[j].GetCity()
		if ti != tj {
			return ti < tj
		}
		return state.AllServers[i].GetName() < state.AllServers[j].GetName()
	})

	nameIndex := make([]NameIndexEntry, len(state.AllServers))
	for i := 0; i < len(state.AllServers); i++ {
		nameIndex[i] = NameIndexEntry{
			Name: state.AllServers[i].Name,
			Idx:  uint32(i),
		}
	}
	sort.Slice(nameIndex, func(i, j int) bool {
		ni := getString(nameIndex[i].Name[:])
		nj := getString(nameIndex[j].Name[:])
		return ni < nj
	})
	state.NameIndex = nameIndex

	boundaries := []RegionBoundary{}
	n := len(state.AllServers)
	if n > 0 {
		start := 0
		currentCountry := state.AllServers[0].GetCountry()
		currentCity := state.AllServers[0].GetCity()

		for i := 1; i <= n; i++ {
			var country, city string
			if i < n {
				country = state.AllServers[i].GetCountry()
				city = state.AllServers[i].GetCity()
			}
			if i == n || country != currentCountry || city != currentCity {
				hash := computeRegionHash(currentCountry, currentCity)
				boundaries = append(boundaries, RegionBoundary{
					Hash:  hash,
					Start: uint32(start),
					End:   uint32(i),
				})
				if i < n {
					start = i
					currentCountry = country
					currentCity = city
				}
			}
		}

		start = 0
		currentCountry = state.AllServers[0].GetCountry()
		for i := 1; i <= n; i++ {
			var country string
			if i < n {
				country = state.AllServers[i].GetCountry()
			}
			if i == n || country != currentCountry {
				hash := computeRegionHash(currentCountry, "")
				boundaries = append(boundaries, RegionBoundary{
					Hash:  hash,
					Start: uint32(start),
					End:   uint32(i),
				})
				if i < n {
					start = i
					currentCountry = country
				}
			}
		}
	}

	sort.Slice(boundaries, func(i, j int) bool {
		return boundaries[i].Hash < boundaries[j].Hash
	})
	state.Boundaries = boundaries

	peerHostname := make([][]byte, len(state.AllServers))
	peerStation := make([][]byte, len(state.AllServers))
	for i := range state.AllServers {
		srv := state.AllServers[i]
		pk := state.Keys[int(srv.KeyID)]
		peerHostname[i] = wg.BuildPeerPrefix(pk, getBytes(srv.Hostname[:]))
		peerStation[i] = wg.BuildPeerPrefix(pk, getBytes(srv.Station[:]))
	}
	state.PeerHostname = peerHostname
	state.PeerStation = peerStation

	cityStart := 0
	for cityStart < n {
		cityEnd := cityStart + 1
		for cityEnd < n &&
			state.AllServers[cityEnd].GetCity() == state.AllServers[cityStart].GetCity() &&
			state.AllServers[cityEnd].GetCountry() == state.AllServers[cityStart].GetCountry() {
			cityEnd++
		}

		seenNames := make(map[string]int)
		for i := cityStart; i < cityEnd; i++ {
			fileName := state.AllServers[i].GetFileName()
			base := fileName[:len(fileName)-5]
			count := seenNames[base]
			seenNames[base] = count + 1
			if count > 0 {
				suffix := "_" + strconv.Itoa(count)
				copyToBytes(state.AllServers[i].CityDedupSuffix[:], suffix)
			}
		}

		cityStart = cityEnd
	}

	jsonData, err := sonic.Marshal(payload)
	if err != nil {
		return
	}

	var brBuf bytes.Buffer
	brw := brotli.NewWriterLevel(&brBuf, 1)
	brw.Write(jsonData)
	brw.Close()

	state.ServerJson = jsonData
	state.ServerJsonBr = brBuf.Bytes()
	state.ServerEtag = buildServerEtag(time.Now().UnixNano())

	s.state.Store(state)
}

func (s *Store) GetServerList() ([]byte, []byte, string) {
	state := s.state.Load()
	if state == nil {
		return nil, nil, ""
	}
	return state.ServerJson, state.ServerJsonBr, state.ServerEtag
}

func (s *Store) GetServer(name string) (types.ProcessedServer, bool) {
	state := s.state.Load()
	if state == nil {
		return types.ProcessedServer{}, false
	}
	idx, found := state.SearchByName(name)
	if !found {
		return types.ProcessedServer{}, false
	}
	return state.AllServers[idx], true
}

func (s *Store) GetKey(id int) (string, bool) {
	state := s.state.Load()
	if state == nil {
		return "", false
	}
	v, ok := state.Keys[id]
	return v, ok
}

func (s *Store) GetBatch(country, city string) []types.ProcessedServer {
	state := s.state.Load()
	if state == nil {
		return nil
	}
	servers, _ := s.GetBatchFromState(state, country, city)
	return servers
}

func (s *Store) GetBatchFromState(state *State, country, city string) ([]types.ProcessedServer, uint32) {
	cKey := normalize(country)
	tKey := normalize(city)

	if cKey == "" {
		return state.AllServers, 0
	}

	hash := computeRegionHash(cKey, tKey)
	idx := state.SearchBoundary(hash)
	if idx == -1 {
		return nil, 0
	}
	bound := state.Boundaries[idx]
	return state.AllServers[bound.Start:bound.End], bound.Start
}

func (s *State) SearchByName(name string) (int, bool) {
	low, high := 0, len(s.NameIndex)-1
	for low <= high {
		mid := (low + high) >> 1
		midName := getString(s.NameIndex[mid].Name[:])
		if midName < name {
			low = mid + 1
		} else if midName > name {
			high = mid - 1
		} else {
			return int(s.NameIndex[mid].Idx), true
		}
	}
	return -1, false
}

func (s *State) SearchBoundary(hash uint64) int {
	low, high := 0, len(s.Boundaries)-1
	for low <= high {
		mid := (low + high) >> 1
		val := s.Boundaries[mid].Hash
		if val < hash {
			low = mid + 1
		} else if val > hash {
			high = mid - 1
		} else {
			return mid
		}
	}
	return -1
}

func buildServerEtag(ts int64) string {
	buf := make([]byte, 0, 22)
	buf = append(buf, '"')
	buf = strconv.AppendInt(buf, ts, 16)
	buf = append(buf, '"')
	return string(buf)
}

func toLower(s string) string {
	for i := 0; i < len(s); i++ {
		if s[i] >= 'A' && s[i] <= 'Z' {
			b := make([]byte, len(s))
			copy(b, s[:i])
			for ; i < len(s); i++ {
				c := s[i]
				if c >= 'A' && c <= 'Z' {
					c += 32
				}
				b[i] = c
			}
			return string(b)
		}
	}
	return s
}

func extractNumber(s string) string {
	start := -1
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			if start == -1 {
				start = i
			}
		} else {
			if start != -1 {
				return s[start:i]
			}
		}
	}
	if start != -1 {
		return s[start:]
	}
	return ""
}

func buildFileName(lowCode, number string) string {
	buf := make([]byte, 0, len(lowCode)+len(number)+5)
	buf = append(buf, lowCode...)
	buf = append(buf, number...)
	buf = append(buf, '.', 'c', 'o', 'n', 'f')
	return string(buf)
}

func normalize(s string) string {
	b := make([]byte, 0, len(s))
	lastUnderscore := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			b = append(b, c)
			lastUnderscore = false
		} else {
			if !lastUnderscore {
				b = append(b, '_')
				lastUnderscore = true
			}
		}
	}
	return string(b)
}

func validateVersion(v string) bool {
	if len(v) < 3 {
		return false
	}
	dot := -1
	for i := 0; i < len(v); i++ {
		if v[i] == '.' {
			dot = i
			break
		}
	}
	if dot == -1 {
		return false
	}

	maj := 0
	for i := 0; i < dot; i++ {
		if v[i] < '0' || v[i] > '9' {
			return false
		}
		maj = maj*10 + int(v[i]-'0')
		if maj > 999 {
			return false
		}
	}

	if maj > 2 {
		return true
	}
	if maj < 2 {
		return false
	}

	min := 0
	start := dot + 1
	if start >= len(v) {
		return false
	}

	for i := start; i < len(v); i++ {
		if v[i] == '.' {
			break
		}
		if v[i] < '0' || v[i] > '9' {
			return false
		}
		min = min*10 + int(v[i]-'0')
		if min > 999 {
			return false
		}
	}

	return min >= 1
}

func getString(b []byte) string {
	for i := 0; i < len(b); i++ {
		if b[i] == 0 {
			if i == 0 {
				return ""
			}
			return unsafe.String(&b[0], i)
		}
	}
	if len(b) == 0 {
		return ""
	}
	return unsafe.String(&b[0], len(b))
}

func getBytes(b []byte) []byte {
	for i := 0; i < len(b); i++ {
		if b[i] == 0 {
			return b[:i]
		}
	}
	return b
}

func copyToBytes(dst []byte, src string) {
	limit := len(dst)
	if len(src) < limit {
		limit = len(src)
	}
	copy(dst[:limit], src[:limit])
	if limit < len(dst) {
		dst[limit] = 0
	}
}

func computeRegionHash(country, city string) uint64 {
	var hash uint64 = 14695981039346656037
	for i := 0; i < len(country); i++ {
		hash ^= uint64(country[i])
		hash *= 1099511628211
	}
	hash ^= uint64('/')
	hash *= 1099511628211
	for i := 0; i < len(city); i++ {
		hash ^= uint64(city[i])
		hash *= 1099511628211
	}
	return hash
}

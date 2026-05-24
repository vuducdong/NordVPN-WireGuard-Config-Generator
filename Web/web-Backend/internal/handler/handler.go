package handler

import (
	"bufio"
	"io"
	"net/http"
	"strings"
	"time"

	"nordgen/internal/qr"
	"nordgen/internal/store"
	"nordgen/internal/types"
	"nordgen/internal/validator"
	"nordgen/internal/wg"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"
	"github.com/klauspost/compress/zip"
)

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
	Transport: &http.Transport{
		ForceAttemptHTTP2:   true,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
	},
}

func GetServers(c fiber.Ctx) error {
	data, brData, etag := store.Core.GetServerList()
	if data == nil {
		return c.Status(503).JSON(fiber.Map{"error": "Initializing"})
	}
	c.Set("ETag", etag)
	c.Set("Cache-Control", "public, no-transform, max-age=300")
	c.Set("Vary", "Accept-Encoding")
	if c.Get("if-none-match") == etag {
		return c.SendStatus(304)
	}
	c.Set("Content-Type", "application/json; charset=utf-8")
	if brData != nil && strings.Contains(c.Get("Accept-Encoding"), "br") {
		c.Set("Content-Encoding", "br")
		return c.Send(brData)
	}
	return c.Send(data)
}

func ExchangeToken(c fiber.Ctx) error {
	var body struct {
		Token string `json:"token"`
	}
	if err := sonic.Unmarshal(c.Body(), &body); err != nil {
		return c.SendStatus(400)
	}

	if !validator.IsHex(body.Token) {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid token"})
	}

	req, _ := http.NewRequest("GET", "https://api.nordvpn.com/v1/users/services/credentials", nil)
	req.Header.Set("Authorization", "Bearer token:"+body.Token)

	resp, err := httpClient.Do(req)
	if err != nil {
		return c.Status(503).JSON(fiber.Map{"error": "Upstream error"})
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return c.Status(401).JSON(fiber.Map{"error": "Expired token"})
	}
	if resp.StatusCode != 200 {
		return c.Status(503).JSON(fiber.Map{"error": "Upstream error"})
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return c.SendStatus(500)
	}

	var data struct {
		Key string `json:"nordlynx_private_key"`
	}
	if sonic.Unmarshal(respBody, &data) != nil {
		return c.SendStatus(500)
	}

	return c.JSON(fiber.Map{"key": data.Key})
}

func GenerateConfig(outputType string) fiber.Handler {
	return func(c fiber.Ctx) error {
		var body types.ConfigRequest
		if err := sonic.Unmarshal(c.Body(), &body); err != nil {
			return c.SendStatus(400)
		}

		cfg, errMsg := validator.ValidateConfig(body)
		if errMsg != "" {
			return c.Status(400).JSON(fiber.Map{"error": errMsg})
		}

		state := store.Core.LoadState()
		if state == nil {
			return c.Status(503).JSON(fiber.Map{"error": "Initializing"})
		}

		idx, found := state.SearchByName(cfg.Name)
		if !found {
			return c.Status(404).JSON(fiber.Map{"error": "Server not found"})
		}

		srv := state.AllServers[idx]

		var peerPrefix []byte
		if cfg.UseStation {
			peerPrefix = state.PeerStation[idx]
		} else {
			peerPrefix = state.PeerHostname[idx]
		}

		c.Set("Cache-Control", "no-store")

		configBytes := wg.Build(cfg.PrivateKey, cfg.DNS, peerPrefix, cfg.KeepAlive)

		if outputType == "text" {
			c.Set("Content-Type", "text/plain")
			return c.Send(configBytes)
		}

		if outputType == "file" {
			c.Set("Content-Disposition", buildConfDisposition(srv.GetLowCode(), srv.GetNumber()))
			c.Set("Content-Type", "application/x-wireguard-config")
			return c.Send(configBytes)
		}

		svg, err := qr.GenerateSVG(configBytes)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "QR generation error"})
		}
		c.Set("Content-Type", "image/svg+xml; charset=utf-8")
		return c.Send(svg)
	}
}

func GenerateBatch(c fiber.Ctx) error {
	var body types.BatchConfigReq
	if err := sonic.Unmarshal(c.Body(), &body); err != nil {
		return c.SendStatus(400)
	}

	cfg, errMsg := validator.ValidateBatch(body)
	if errMsg != "" {
		return c.Status(400).JSON(fiber.Map{"error": errMsg})
	}

	state := store.Core.LoadState()
	if state == nil {
		return c.Status(503).JSON(fiber.Map{"error": "Initializing"})
	}

	servers, startIdx := store.Core.GetBatchFromState(state, body.Country, body.City)
	if len(servers) == 0 {
		return c.Status(404).JSON(fiber.Map{"error": "No servers found"})
	}

	baseName := buildBaseName(body.Country, body.City)

	c.Set("Content-Type", "application/octet-stream")
	c.Set("Content-Disposition", buildDisposition(baseName))
	c.Set("Cache-Control", "no-store")

	return c.SendStreamWriter(func(w *bufio.Writer) {
		zw := zip.NewWriter(w)
		defer zw.Close()

		for i, srv := range servers {
			idx := int(startIdx) + i
			var peerPrefix []byte
			if cfg.UseStation {
				peerPrefix = state.PeerStation[idx]
			} else {
				peerPrefix = state.PeerHostname[idx]
			}

			path := buildBatchPath(body.Country, body.City, srv)

			f, err := zw.CreateHeader(&zip.FileHeader{
				Name:   path,
				Method: zip.Store,
			})
			if err != nil {
				continue
			}

			if err := wg.WriteConfig(f, cfg.PrivateKey, cfg.DNS, peerPrefix, cfg.KeepAlive); err != nil {
				continue
			}
		}
	})
}

func buildBatchPath(batchCountry, batchCity string, srv types.ProcessedServer) string {
	srvCountry := srv.GetCountry()
	srvCity := srv.GetCity()
	srvFileName := srv.GetFileName()

	if batchCity != "" {
		suffix := srv.GetCityDedupSuffix()
		if suffix != "" {
			base := srvFileName[:len(srvFileName)-5]
			srvFileName = base + suffix + ".conf"
		}
		return srvFileName
	}
	if batchCountry == "" {
		size := len(srvCountry) + len(srvCity) + len(srvFileName) + 2
		buf := make([]byte, 0, size)
		buf = append(buf, srvCountry...)
		buf = append(buf, '/')
		buf = append(buf, srvCity...)
		buf = append(buf, '/')
		buf = append(buf, srvFileName...)
		return string(buf)
	}
	size := len(srvCity) + len(srvFileName) + 1
	buf := make([]byte, 0, size)
	buf = append(buf, srvCity...)
	buf = append(buf, '/')
	buf = append(buf, srvFileName...)
	return string(buf)
}

func sanitizeFilename(s string) string {
	b := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' {
			b = append(b, c)
		} else if c == ' ' {
			b = append(b, '_')
		}
	}
	return string(b)
}

func buildBaseName(country, city string) string {
	if country == "" {
		return "NordVPN_All"
	}
	sc := sanitizeFilename(country)
	if city == "" {
		buf := make([]byte, 0, 8+len(sc))
		buf = append(buf, "NordVPN_"...)
		buf = append(buf, sc...)
		return string(buf)
	}
	scity := sanitizeFilename(city)
	buf := make([]byte, 0, 9+len(sc)+len(scity))
	buf = append(buf, "NordVPN_"...)
	buf = append(buf, sc...)
	buf = append(buf, '_')
	buf = append(buf, scity...)
	return string(buf)
}

func buildDisposition(name string) string {
	buf := make([]byte, 0, 24+len(name))
	buf = append(buf, `attachment; filename="`...)
	buf = append(buf, name...)
	buf = append(buf, `.nord"`...)
	return string(buf)
}

func buildConfDisposition(code, num string) string {
	buf := make([]byte, 0, 24+len(code)+len(num)+5)
	buf = append(buf, `attachment; filename="`...)
	buf = append(buf, code...)
	buf = append(buf, num...)
	buf = append(buf, `.conf"`...)
	return string(buf)
}

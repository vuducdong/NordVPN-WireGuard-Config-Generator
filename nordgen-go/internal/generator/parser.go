package generator

import (
	"math"
	"sort"
	"strings"

	"nordgen/internal/constants"
	"nordgen/internal/models"
)

const earthRadiusKM = 6371.0

func parseServers(rawServers []models.RawServer, obsLat, obsLon float64, reqGroups []string, excDedicated bool) []models.Server {
	reqMap := make(map[string]struct{}, len(reqGroups))
	for _, rg := range reqGroups {
		reqMap[rg] = struct{}{}
	}

	obsLatRad := obsLat * math.Pi / 180.0
	obsLonRad := obsLon * math.Pi / 180.0
	obsLatCos := math.Cos(obsLatRad)

	parsed := make([]models.Server, 0, len(rawServers))

	for _, raw := range rawServers {
		var typeGroupIDs []string
		hasDedicated := false

		for _, g := range raw.Groups {
			gid := g.Identifier
			if _, exists := constants.TypeGroups[gid]; !exists {
				continue
			}
			typeGroupIDs = append(typeGroupIDs, gid)
			if gid == "legacy_dedicated_ip" {
				hasDedicated = true
			}
		}

		if len(typeGroupIDs) == 0 {
			continue
		}
		if excDedicated && hasDedicated {
			continue
		}

		if len(reqMap) > 0 {
			hasAll := true
			for rg := range reqMap {
				found := false
				for _, tg := range typeGroupIDs {
					if tg == rg {
						found = true
						break
					}
				}
				if !found {
					hasAll = false
					break
				}
			}
			if !hasAll {
				continue
			}
		}

		sort.Strings(typeGroupIDs)
		comboParts := make([]string, len(typeGroupIDs))
		for i, g := range typeGroupIDs {
			comboParts[i] = constants.GroupIDToAlias[g]
		}
		combo := strings.Join(comboParts, "_")

		var pubKey string
		for _, tech := range raw.Technologies {
			for _, meta := range tech.Metadata {
				if meta.Name == "public_key" && meta.Value != "" {
					pubKey = meta.Value
					break
				}
			}
			if pubKey != "" {
				break
			}
		}

		if pubKey == "" || len(raw.Locations) == 0 {
			continue
		}

		loc := raw.Locations[0]
		latRad := loc.Latitude * math.Pi / 180.0
		dlat := latRad - obsLatRad
		dlon := (loc.Longitude * math.Pi / 180.0) - obsLonRad

		sinDLat := math.Sin(dlat / 2.0)
		sinDLon := math.Sin(dlon / 2.0)
		a := (sinDLat * sinDLat) + obsLatCos*math.Cos(latRad)*(sinDLon*sinDLon)

		if a > 1.0 {
			a = 1.0
		} else if a < 0.0 {
			a = 0.0
		}

		dist := earthRadiusKM * 2 * math.Asin(math.Sqrt(a))

		nameParts := strings.SplitN(raw.Hostname, ".", 2)
		name := nameParts[0]

		parsed = append(parsed, models.Server{
			Name:        name,
			Hostname:    raw.Hostname,
			Station:     raw.Station,
			Load:        raw.Load,
			Country:     loc.Country.Name,
			CountryCode: strings.ToLower(loc.Country.Code),
			City:        loc.Country.City.Name,
			Latitude:    loc.Latitude,
			Longitude:   loc.Longitude,
			PublicKey:   pubKey,
			Distance:    dist,
			Combo:       combo,
		})
	}

	return parsed
}

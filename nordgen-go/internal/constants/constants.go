package constants

const (
	ServersURL = "https://api.nordvpn.com/v1/servers?limit=16384&filters[servers_technologies][identifier]=wireguard_udp&fields[station]=1&fields[hostname]=1&fields[load]=1&fields[technologies.metadata]=1&fields[locations.country.name]=1&fields[locations.country.code]=1&fields[locations.country.city.name]=1&fields[locations.latitude]=1&fields[locations.longitude]=1&fields[groups.identifier]=1"
	GeoURL     = "https://api.nordvpn.com/v1/helpers/ips/insights"
	CredsURL   = "https://api.nordvpn.com/v1/users/services/credentials"
)

var TypeGroups = map[string]struct{}{
	"legacy_standard":       {},
	"legacy_p2p":            {},
	"legacy_dedicated_ip":   {},
	"legacy_onion_over_vpn": {},
	"legacy_double_vpn":     {},
}

var GroupIDToAlias = map[string]string{
	"legacy_standard":       "standard",
	"legacy_p2p":            "p2p",
	"legacy_dedicated_ip":   "dedicated",
	"legacy_onion_over_vpn": "onion",
	"legacy_double_vpn":     "double",
}

var AliasToGroupID = map[string]string{
	"standard":  "legacy_standard",
	"p2p":       "legacy_p2p",
	"dedicated": "legacy_dedicated_ip",
	"onion":     "legacy_onion_over_vpn",
	"double":    "legacy_double_vpn",
}

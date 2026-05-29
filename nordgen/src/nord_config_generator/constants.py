TYPE_GROUPS = frozenset(
    {
        "legacy_standard",
        "legacy_p2p",
        "legacy_dedicated_ip",
        "legacy_onion_over_vpn",
        "legacy_double_vpn",
    }
)

GROUP_ID_TO_ALIAS = {
    "legacy_standard": "standard",
    "legacy_p2p": "p2p",
    "legacy_dedicated_ip": "dedicated",
    "legacy_onion_over_vpn": "onion",
    "legacy_double_vpn": "double",
}

ALIAS_TO_GROUP_ID = {v: k for k, v in GROUP_ID_TO_ALIAS.items()}

SERVERS_URL = "https://api.nordvpn.com/v1/servers?limit=16384&filters[servers_technologies][identifier]=wireguard_udp&fields[station]=1&fields[hostname]=1&fields[load]=1&fields[technologies.metadata]=1&fields[locations.country.name]=1&fields[locations.country.code]=1&fields[locations.country.city.name]=1&fields[locations.latitude]=1&fields[locations.longitude]=1&fields[groups.identifier]=1"

GEO_URL = "https://api.nordvpn.com/v1/helpers/ips/insights"
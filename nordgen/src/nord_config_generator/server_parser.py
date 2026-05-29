from math import asin, cos, radians, sin, sqrt
from typing import Iterable

from .constants import GROUP_ID_TO_ALIAS, TYPE_GROUPS
from .models import Server

_EARTH_RADIUS_KM = 6371.0

def parse_servers(
    raw_servers: Iterable[dict],
    observer_lat: float,
    observer_lon: float,
    required_groups: set[str] | None = None,
    exclude_dedicated: bool = False,
) -> list[Server]:
    obs_lat_rad = radians(observer_lat)
    obs_lon_rad = radians(observer_lon)
    obs_lat_cos = cos(obs_lat_rad)
    parsed = []

    for data in raw_servers:
        try:
            raw_groups = data.get("groups", [])
            if not raw_groups:
                continue

            type_group_ids = [
                ident
                for g in raw_groups
                if (ident := g.get("identifier", "")) in TYPE_GROUPS
            ]
            if not type_group_ids:
                continue

            if exclude_dedicated and "legacy_dedicated_ip" in type_group_ids:
                continue

            if required_groups and not required_groups.issubset(type_group_ids):
                continue

            type_group_ids.sort()
            combo = "_".join(GROUP_ID_TO_ALIAS[g] for g in type_group_ids)

            public_key = None
            for tech in data.get("technologies", []):
                for meta in tech.get("metadata", []):
                    if meta.get("name") == "public_key" and meta.get("value"):
                        public_key = meta["value"]
                        break
                if public_key:
                    break

            if not public_key:
                continue

            loc = data["locations"][0]
            lat = float(loc["latitude"])
            lon = float(loc["longitude"])

            lat_rad = radians(lat)
            dlat = lat_rad - obs_lat_rad
            dlon = radians(lon) - obs_lon_rad
            a = sin(dlat * 0.5) ** 2 + obs_lat_cos * cos(lat_rad) * sin(dlon * 0.5) ** 2
            a = max(0.0, min(1.0, a))
            distance = _EARTH_RADIUS_KM * 2 * asin(sqrt(a))

            country = loc["country"]
            hostname = data["hostname"]

            parsed.append(
                Server(
                    name=hostname.split(".", 1)[0],
                    hostname=hostname,
                    station=data["station"],
                    load=int(data.get("load", 0)),
                    country=country["name"],
                    country_code=country["code"].lower(),
                    city=country["city"]["name"],
                    latitude=lat,
                    longitude=lon,
                    public_key=public_key,
                    distance=distance,
                    combo=combo,
                )
            )
        except (KeyError, IndexError, ValueError, TypeError):
            continue

    return parsed
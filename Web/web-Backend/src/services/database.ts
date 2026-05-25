import {
  KV_DATABASE_KEY,
  KV_SERVERS_JSON_KEY,
  KV_VERSION_KEY,
  NORDVPN_SERVERS_URL,
} from "../constants";
import { normalizeName, sanitizeIdentifier, extractNumber } from "../lib/naming";
import { validateVersion } from "../lib/version";
import type { CompactDatabase, ServerEntry } from "../types";

let databaseCache: CompactDatabase | null = null;

export async function loadDatabase(env: Env): Promise<CompactDatabase | null> {
  if (databaseCache) return databaseCache;
  const raw = await env.NORDGEN_KV.get(KV_DATABASE_KEY);
  if (!raw) return null;
  databaseCache = JSON.parse(raw) as CompactDatabase;
  return databaseCache;
}

export async function refreshServerDatabase(env: Env): Promise<void> {
  const response = await fetch(NORDVPN_SERVERS_URL);
  if (!response.ok) return;

  const rawServers: Array<Record<string, unknown>> = await response.json() as Array<Record<string, unknown>>;

  interface RawServer {
    name: string;
    station: string;
    hostname: string;
    load: number;
    locations: Array<{ country: { name: string; code: string; city: { name: string } } }>;
    specifications?: Array<{ identifier: string; values?: Array<{ value: string }> }>;
    technologies?: Array<{ metadata?: Array<{ name: string; value: string }> }>;
  }

  const processedServers: Array<{
    name: string;
    station: string;
    hostname: string;
    country: string;
    city: string;
    lowCode: string;
    number: string;
    fileName: string;
    keyIndex: number;
    load: number;
    rawCountryName: string;
    rawCityName: string;
    rawServerName: string;
    dedupSuffix: string;
  }> = [];

  const uniqueKeys: string[] = [];
  const keyIndexMap = new Map<string, number>();

  for (const raw of rawServers) {
    const server = raw as unknown as RawServer;
    if (!server.locations || server.locations.length === 0) continue;

    let version = "0.0.0";
    if (server.specifications) {
      for (const spec of server.specifications) {
        if (spec.identifier === "version" && spec.values && spec.values.length > 0) {
          version = spec.values[0].value;
          break;
        }
      }
    }

    if (!validateVersion(version)) continue;

    let publicKey = "";
    if (server.technologies) {
      for (const tech of server.technologies) {
        if (tech.metadata) {
          for (const meta of tech.metadata) {
            if (meta.name === "public_key") {
              publicKey = meta.value;
              break;
            }
          }
        }
        if (publicKey) break;
      }
    }

    const location = server.locations[0];
    if (!location.country || !location.country.code || !publicKey) continue;

    let keyIdx = keyIndexMap.get(publicKey);
    if (keyIdx === undefined) {
      keyIdx = uniqueKeys.length;
      uniqueKeys.push(publicKey);
      keyIndexMap.set(publicKey, keyIdx);
    }

    const countryName = normalizeName(location.country.name);
    const cityName = normalizeName(location.country.city.name);
    const lowCountryCode = location.country.code.toLowerCase();

    const serverName = normalizeName(server.name);
    const serverNumber = extractNumber(server.name) || "wg";
    const baseFileName = `${lowCountryCode}${serverNumber}.conf`;

    processedServers.push({
      name: serverName,
      station: server.station,
      hostname: server.hostname,
      country: countryName,
      city: cityName,
      lowCode: lowCountryCode,
      number: serverNumber,
      fileName: baseFileName,
      keyIndex: keyIdx,
      load: server.load,
      rawCountryName: location.country.name,
      rawCityName: location.country.city.name,
      rawServerName: server.name,
      dedupSuffix: "",
    });
  }

  processedServers.sort((a, b) => {
    if (a.country !== b.country) return a.country.localeCompare(b.country);
    if (a.city !== b.city) return a.city.localeCompare(b.city);
    return a.name.localeCompare(b.name);
  });

  let cityStart = 0;
  const totalServers = processedServers.length;
  while (cityStart < totalServers) {
    let cityEnd = cityStart + 1;
    while (
      cityEnd < totalServers &&
      processedServers[cityEnd].city === processedServers[cityStart].city &&
      processedServers[cityEnd].country === processedServers[cityStart].country
    ) {
      cityEnd++;
    }

    const nameCounts = new Map<string, number>();
    for (let i = cityStart; i < cityEnd; i++) {
      const srv = processedServers[i];
      const baseName = srv.fileName.slice(0, -5);
      const count = nameCounts.get(baseName) || 0;
      nameCounts.set(baseName, count + 1);
      srv.dedupSuffix = count > 0 ? `_${count}` : "";
    }
    cityStart = cityEnd;
  }

  const payloadByCountry: Record<string, Record<string, Array<[string, number, string]>>> = {};
  for (const srv of processedServers) {
    const countryKey = sanitizeIdentifier(srv.rawCountryName);
    const cityKey = sanitizeIdentifier(srv.rawCityName);

    if (!payloadByCountry[countryKey]) {
      payloadByCountry[countryKey] = {};
    }
    if (!payloadByCountry[countryKey][cityKey]) {
      payloadByCountry[countryKey][cityKey] = [];
    }
    payloadByCountry[countryKey][cityKey].push([sanitizeIdentifier(srv.rawServerName), srv.load, srv.station]);
  }

  const compactServers: Record<string, ServerEntry> = {};
  const regions: Record<string, string[]> = {};

  for (const srv of processedServers) {
    compactServers[srv.name] = {
      station: srv.station,
      hostname: srv.hostname,
      country: srv.country,
      city: srv.city,
      lowCode: srv.lowCode,
      number: srv.number,
      keyIndex: srv.keyIndex,
      dedupSuffix: srv.dedupSuffix,
    };

    const countryRegionKey = srv.country;
    const cityRegionKey = `${srv.country}/${srv.city}`;

    if (!regions[countryRegionKey]) regions[countryRegionKey] = [];
    regions[countryRegionKey].push(srv.name);

    if (!regions[cityRegionKey]) regions[cityRegionKey] = [];
    regions[cityRegionKey].push(srv.name);
  }

  const database: CompactDatabase = {
    keys: uniqueKeys,
    servers: compactServers,
    regions,
  };

  const apiResponse = {
    h: ["name", "load", "station"],
    l: payloadByCountry,
  };

  const version = Date.now().toString(16);

  await env.NORDGEN_KV.put(KV_DATABASE_KEY, JSON.stringify(database));
  await env.NORDGEN_KV.put(KV_SERVERS_JSON_KEY, JSON.stringify(apiResponse));
  await env.NORDGEN_KV.put(KV_VERSION_KEY, version);

  databaseCache = database;
}
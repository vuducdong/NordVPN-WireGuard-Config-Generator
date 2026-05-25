import {
  KV_INJECTION_KEY,
  KV_VERSION_KEY,
  NORDVPN_SERVERS_URL,
} from "../constants";
import { normalizeName, sanitizeIdentifier, extractNumber } from "../lib/naming";
import { validateVersion } from "../lib/version";
import { clearMemoryCache } from "../routes/servers";

function ipToNumeric(ip: string): number {
  const parts = ip.split(".").map(Number);
  return ((parts[0] << 24) >>> 0) + (parts[1] << 16) + (parts[2] << 8) + parts[3];
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
    keyIndex: number;
    load: number;
    rawCountryName: string;
    rawCityName: string;
    dedupSuffix: string;
  }> = [];

  const uniqueKeys: string[] = [];
  const keyMap = new Map<string, number>();

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

    let keyIdx = keyMap.get(publicKey);
    if (keyIdx === undefined) {
      keyIdx = uniqueKeys.length;
      uniqueKeys.push(publicKey);
      keyMap.set(publicKey, keyIdx);
    }

    const countryName = normalizeName(location.country.name);
    const cityName = normalizeName(location.country.city.name);
    const lowCountryCode = location.country.code.toLowerCase();
    const serverName = normalizeName(server.name);
    const serverNumber = extractNumber(server.name) || "wg";

    processedServers.push({
      name: serverName,
      station: server.station,
      hostname: server.hostname,
      country: countryName,
      city: cityName,
      lowCode: lowCountryCode,
      number: serverNumber,
      keyIndex: keyIdx,
      load: server.load,
      rawCountryName: location.country.name,
      rawCityName: location.country.city.name,
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
      const baseName = `${srv.lowCode}${srv.number}`;
      const count = nameCounts.get(baseName) || 0;
      nameCounts.set(baseName, count + 1);
      srv.dedupSuffix = count > 0 ? `_${count}` : "";
    }
    cityStart = cityEnd;
  }

  const l: Array<[string, string, Array<[string, Array<Array<number | string>>]>]> = [];
  
  let currentCountry = "";
  let currentCountryArr: [string, string, Array<[string, Array<Array<number | string>>]>] | null = null;
  let currentCity = "";
  let currentCityArr: [string, Array<Array<number | string>>] | null = null;

  for (const srv of processedServers) {
    const countryKey = sanitizeIdentifier(srv.rawCountryName);
    const cityKey = sanitizeIdentifier(srv.rawCityName);

    if (!currentCountryArr || currentCountry !== countryKey) {
      currentCountry = countryKey;
      currentCountryArr = [countryKey, srv.lowCode, []];
      l.push(currentCountryArr);
      currentCity = "";
    }

    if (!currentCityArr || currentCity !== cityKey) {
      currentCity = cityKey;
      currentCityArr = [cityKey, []];
      currentCountryArr[2].push(currentCityArr);
    }

    const ipNum = ipToNumeric(srv.station);
    const prefix = srv.lowCode === "gb" ? "uk" : srv.lowCode;
    const expectedHostname = `${prefix}${srv.number}.nordvpn.com`;
    const hName = srv.hostname === expectedHostname ? "" : srv.hostname;
    const serverNum = isNaN(Number(srv.number)) ? srv.number : Number(srv.number);

    const tuple: Array<number | string> = [serverNum, srv.load, ipNum, srv.keyIndex];

    if (srv.dedupSuffix !== "") {
      tuple.push(hName, srv.dedupSuffix);
    } else if (hName !== "") {
      tuple.push(hName);
    }

    currentCityArr[1].push(tuple);
  }

  const apiResponse = {
    k: uniqueKeys,
    l
  };

  const apiResponseJson = JSON.stringify(apiResponse);
  const safeServersJson = apiResponseJson.replace(/</g, "\\u003c");
  const injectionScript = `<script>window.__SERVER_LIST__=${safeServersJson};</script>`;
  const version = Date.now().toString(16);

  await Promise.all([
    env.NORDGEN_KV.put("global:api_response", apiResponseJson, {
      metadata: { version }
    }),
    env.NORDGEN_KV.put(KV_INJECTION_KEY, injectionScript),
    env.NORDGEN_KV.put(KV_VERSION_KEY, version)
  ]);

  clearMemoryCache();
}
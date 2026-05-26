import {
  KV_INJECTION_KEY,
  KV_VERSION_KEY,
  NORDVPN_SERVERS_URL,
} from "../constants";
import { normalizeName, sanitizeIdentifier, extractNumber } from "../lib/naming";
import { validateVersion } from "../lib/version";
import { clearMemoryCache } from "../routes/servers";

function ipToNumeric(ip: string): number {
  let val = 0;
  let octet = 0;
  let shift = 24;
  const len = ip.length;
  for (let i = 0; i < len; i++) {
    const charCode = ip.charCodeAt(i);
    if (charCode === 46) {
      val = (val | (octet << shift)) >>> 0;
      octet = 0;
      shift -= 8;
    } else {
      octet = octet * 10 + (charCode - 48);
    }
  }
  return (val | (octet << shift)) >>> 0;
}

export async function refreshServerDatabase(env: Env): Promise<void> {
  const response = await fetch(NORDVPN_SERVERS_URL);
  if (!response.ok) return;

  const rawServers: Array<Record<string, unknown>> = await response.json() as Array<Record<string, unknown>>;

  interface RawServer {
    station: string;
    hostname: string;
    load: number;
    locations: Array<{ country: { name: string; code: string; city: { name: string } } }>;
    specifications?: Array<{ identifier: string; values?: Array<{ value: string }> }>;
    technologies?: Array<{ metadata?: Array<{ name: string; value: string }> }>;
  }

  const processedServers: Array<{
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

  const normalizeCache = new Map<string, string>();
  const sanitizeCache = new Map<string, string>();
  const lowCodeCache = new Map<string, string>();

  function getNormalized(val: string): string {
    let res = normalizeCache.get(val);
    if (res === undefined) {
      res = normalizeName(val);
      normalizeCache.set(val, res);
    }
    return res;
  }

  function getSanitized(val: string): string {
    let res = sanitizeCache.get(val);
    if (res === undefined) {
      res = sanitizeIdentifier(val);
      sanitizeCache.set(val, res);
    }
    return res;
  }

  function getLowCode(val: string): string {
    let res = lowCodeCache.get(val);
    if (res === undefined) {
      res = val.toLowerCase();
      lowCodeCache.set(val, res);
    }
    return res;
  }

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

    const countryName = getNormalized(location.country.name);
    const cityName = getNormalized(location.country.city.name);
    const lowCountryCode = getLowCode(location.country.code);
    const serverNumber = extractNumber(server.hostname) || "wg";

    processedServers.push({
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
    if (a.country !== b.country) {
      return a.country < b.country ? -1 : 1;
    }
    if (a.city !== b.city) {
      return a.city < b.city ? -1 : 1;
    }
    const numA = parseInt(a.number, 10);
    const numB = parseInt(b.number, 10);
    if (!isNaN(numA) && !isNaN(numB)) {
      return numA - numB;
    }
    return a.number.localeCompare(b.number);
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
    const countryKey = getSanitized(srv.rawCountryName);
    const cityKey = getSanitized(srv.rawCityName);

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
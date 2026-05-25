import { Hono } from "hono";
import { batchRateLimit } from "../middleware/rate-limit";
import { loadDatabase } from "../services/database";
import { buildWireGuardConfig } from "../lib/wireguard";
import { normalizeName } from "../lib/naming";
import { createZipArchive } from "../lib/zip";
import { batchValidator, type BatchInput } from "../validation/schemas";
import type { ServerEntry, ZipEntry } from "../types";

const batchRoute = new Hono<{ Bindings: Env }>();

function buildBatchFilePath(
  batchCountry: string,
  batchCity: string,
  server: ServerEntry,
): string {
  const baseFileName = `${server.lowCode}${server.number}.conf`;
  if (batchCity !== "") {
    return server.dedupSuffix !== ""
      ? `${server.lowCode}${server.number}${server.dedupSuffix}.conf`
      : baseFileName;
  }
  if (batchCountry === "") {
    return `${server.country}/${server.city}/${baseFileName}`;
  }
  return `${server.city}/${baseFileName}`;
}

batchRoute.post("/", batchRateLimit(), batchValidator(), async (c) => {
  const parsed: BatchInput = c.req.valid("json");
  const database = await loadDatabase(c.env);
  if (!database) {
    return c.json({ error: "Initializing" }, 503);
  }

  const targetCountry = parsed.country ? normalizeName(parsed.country) : "";
  const targetCity = parsed.city ? normalizeName(parsed.city) : "";

  let serverNames: string[];
  if (targetCountry === "") {
    serverNames = Object.keys(database.servers);
  } else {
    const lookupKey = targetCity !== "" ? `${targetCountry}/${targetCity}` : targetCountry;
    serverNames = database.regions[lookupKey] || [];
  }

  if (serverNames.length === 0) {
    return c.json({ error: "No servers found" }, 404);
  }

  const useStation = parsed.endpoint === "station";
  const isClientMode = parsed.mode === "client";

  let archiveName = "NordVPN_All";
  if (targetCountry !== "") {
    archiveName = `NordVPN_${targetCountry.replace(/[^a-zA-Z0-9-_]/g, "_")}`;
    if (targetCity !== "") {
      archiveName += `_${targetCity.replace(/[^a-zA-Z0-9-_]/g, "_")}`;
    }
  }

  if (isClientMode) {
    const templates: Array<{ name: string; template: string }> = [];
    for (const serverName of serverNames) {
      const server = database.servers[serverName];
      if (!server) continue;

      const publicKey = database.keys[server.keyIndex];
      const endpoint = useStation ? server.station : server.hostname;
      const configText = buildWireGuardConfig(
        "__CLIENT_PK__",
        parsed.dns,
        publicKey,
        endpoint,
        parsed.keepalive,
      );
      const zipPath = buildBatchFilePath(targetCountry, targetCity, server);

      templates.push({
        name: zipPath,
        template: configText,
      });
    }

    return c.json({ archiveName, templates }, 200);
  }

  const entries: ZipEntry[] = [];
  for (const serverName of serverNames) {
    const server = database.servers[serverName];
    if (!server) continue;

    const publicKey = database.keys[server.keyIndex];
    const endpoint = useStation ? server.station : server.hostname;
    const configText = buildWireGuardConfig(
      "",
      parsed.dns,
      publicKey,
      endpoint,
      parsed.keepalive,
    );
    const zipPath = buildBatchFilePath(targetCountry, targetCity, server);

    entries.push({
      name: zipPath,
      data: new TextEncoder().encode(configText),
    });
  }

  const zipData = createZipArchive(entries);

  return new Response(zipData, {
    status: 200,
    headers: {
      "Content-Type": "application/octet-stream",
      "Content-Disposition": `attachment; filename="${archiveName}.nord"`,
      "Cache-Control": "no-store",
    },
  });
});

export { batchRoute };
import { Hono } from "hono";
import type { Context } from "hono";
import { configRateLimit } from "../middleware/rate-limit";
import { loadDatabase } from "../services/database";
import { buildWireGuardConfig } from "../lib/wireguard";
import { generateQRCodeSVG } from "../lib/qr";
import { configValidator, type ParsedConfigInput } from "../validation/schemas";
import { normalizeName } from "../lib/naming";

const configRoute = new Hono<{ Bindings: Env }>();

function createConfigTextResponse(
  configText: string,
  format: "text" | "download" | "qr",
  fileName?: string,
): Response {
  switch (format) {
    case "text":
      return new Response(configText, {
        status: 200,
        headers: {
          "Content-Type": "text/plain; charset=utf-8",
          "Cache-Control": "no-store",
        },
      });
    case "download": {
      const headers: Record<string, string> = {
        "Content-Type": "application/x-wireguard-config",
        "Content-Disposition": `attachment; filename="${fileName}"`,
        "Cache-Control": "no-store",
      };
      return new Response(configText, { status: 200, headers });
    }
    case "qr":
      return new Response(configText, {
        status: 200,
        headers: {
          "Content-Type": "image/svg+xml; charset=utf-8",
          "Cache-Control": "no-store",
        },
      });
  }
}

async function buildConfigAndRespond(
  c: Context<{ Bindings: Env }>,
  parsed: ParsedConfigInput,
  format: "text" | "download" | "qr",
): Promise<Response> {
  const database = await loadDatabase(c.env);
  if (!database) {
    return c.json({ error: "Initializing" }, 503);
  }

  const server = database.servers[normalizeName(parsed.name)];
  if (!server) {
    return c.json({ error: "Server not found" }, 404);
  }

  const publicKey = database.keys[server.keyIndex];
  const endpoint = parsed.endpoint === "station" ? server.station : server.hostname;

  if (parsed.mode === "client" && format === "qr") {
    return c.json({ error: "QR code generation is not supported in client mode" }, 400);
  }

  const effectivePrivateKey = parsed.mode === "client" ? "__CLIENT_PK__" : "";

  const configText = buildWireGuardConfig(
    effectivePrivateKey,
    parsed.dns,
    publicKey,
    endpoint,
    parsed.keepalive,
  );

  const fileName = `${server.lowCode}${server.number}.conf`;

  if (parsed.mode === "client") {
    return c.json({
      filename: fileName,
      template: configText,
    }, 200);
  }

  if (format === "qr") {
    const svg = generateQRCodeSVG(configText);
    return createConfigTextResponse(svg, "qr");
  }

  return createConfigTextResponse(
    configText,
    format,
    fileName,
  );
}

configRoute.post("/", configRateLimit(), configValidator(), async (c) => {
  const parsed = c.req.valid("json");
  return buildConfigAndRespond(c, parsed, "text");
});

configRoute.post("/download", configRateLimit(), configValidator(), async (c) => {
  const parsed = c.req.valid("json");
  return buildConfigAndRespond(c, parsed, "download");
});

configRoute.post("/qr", configRateLimit(), configValidator(), async (c) => {
  const parsed = c.req.valid("json");
  return buildConfigAndRespond(c, parsed, "qr");
});

export { configRoute };
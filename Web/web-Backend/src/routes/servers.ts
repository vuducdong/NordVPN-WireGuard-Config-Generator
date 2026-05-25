import { Hono } from "hono";
import { KV_SERVERS_JSON_KEY, KV_VERSION_KEY } from "../constants";
import { configRateLimit } from "../middleware/rate-limit";
import { refreshServerDatabase } from "../services/database";

const serversRoute = new Hono<{ Bindings: Env }>();

serversRoute.get("/", configRateLimit(), async (c) => {
  const cache = caches.default;
  const cacheKey = new Request(c.req.url);

  const cachedResponse = await cache.match(cacheKey);
  if (cachedResponse) {
    return cachedResponse;
  }

  let version = await c.env.NORDGEN_KV.get(KV_VERSION_KEY);
  if (!version) {
    await refreshServerDatabase(c.env);
    version = await c.env.NORDGEN_KV.get(KV_VERSION_KEY);
  }

  if (!version) {
    return c.json({ error: "Initializing" }, 503);
  }

  const clientETag = c.req.header("if-none-match");
  const currentETag = `"${version}"`;
  if (clientETag === currentETag || clientETag === `W/"${version}"`) {
    const res = new Response(null, { status: 304 });
    res.headers.set("ETag", currentETag);
    res.headers.set("Cache-Control", "public, no-transform, max-age=300");
    return res;
  }

  const serversJson = await c.env.NORDGEN_KV.get(KV_SERVERS_JSON_KEY);
  if (!serversJson) {
    return c.json({ error: "Initializing" }, 503);
  }

  const response = new Response(serversJson, {
    status: 200,
    headers: {
      "Content-Type": "application/json; charset=utf-8",
      "ETag": currentETag,
      "Cache-Control": "public, no-transform, max-age=300",
    },
  });

  c.executionCtx.waitUntil(cache.put(cacheKey, response.clone()));

  return response;
});

export { serversRoute };
import { Hono } from "hono";
import { configRateLimit } from "../middleware/rate-limit";
import { refreshServerDatabase } from "../services/database";

const serversRoute = new Hono<{ Bindings: Env }>();

let apiResponseMemoryCache: { value: string; version: string } | null = null;

export function clearMemoryCache(): void {
  apiResponseMemoryCache = null;
}

serversRoute.get("/", configRateLimit(), async (c) => {
  const cache = caches.default;
  const cacheKey = new Request(c.req.url);

  const cachedResponse = await cache.match(cacheKey);
  if (cachedResponse) {
    return cachedResponse;
  }

  let version = "";
  let value = "";

  if (apiResponseMemoryCache) {
    version = apiResponseMemoryCache.version;
    value = apiResponseMemoryCache.value;
  } else {
    const result = await c.env.NORDGEN_KV.getWithMetadata<{ version: string }>("global:api_response");
    if (!result.value || !result.metadata?.version) {
      c.executionCtx.waitUntil(refreshServerDatabase(c.env));
      return c.json({ error: "Initializing" }, 503);
    }
    version = result.metadata.version;
    value = result.value;
    apiResponseMemoryCache = { value, version };
  }

  const clientETag = c.req.header("if-none-match");
  const currentETag = `"${version}"`;
  if (clientETag === currentETag || clientETag === `W/"${version}"`) {
    const res = new Response(null, { status: 304 });
    res.headers.set("ETag", currentETag);
    res.headers.set("Cache-Control", "public, no-transform, max-age=300");
    return res;
  }

  const response = new Response(value, {
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
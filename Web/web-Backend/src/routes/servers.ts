import { Hono } from "hono";
import { configRateLimit } from "../middleware/rate-limit";
import { refreshServerDatabase } from "../services/database";
import { getOrInitializeState } from "../services/memory";

const serversRoute = new Hono<{ Bindings: Env }>();

serversRoute.get("/", configRateLimit(), async (c) => {
  const state = await getOrInitializeState(c.env);
  if (!state) {
    c.executionCtx.waitUntil(refreshServerDatabase(c.env));
    return c.json({ error: "Initializing" }, 503);
  }

  const clientETag = c.req.header("if-none-match");
  const currentETag = `"${state.version}"`;

  c.header("ETag", currentETag);
  c.header("Cache-Control", "public, no-transform, max-age=300");

  if (clientETag === currentETag || clientETag === `W/"${state.version}"`) {
    return c.body(null, 304);
  }

  c.header("Content-Type", "application/json; charset=utf-8");
  return c.body(state.apiResponse, 200);
});

export { serversRoute };
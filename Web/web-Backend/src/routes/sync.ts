import { Hono } from "hono";
import { bearerAuth } from "hono/bearer-auth";
import { KV_DEPLOY_TOKEN_KEY } from "../constants";
import { refreshServerDatabase } from "../services/database";

const syncRoute = new Hono<{ Bindings: Env }>();

syncRoute.post("/", async (c, next) => {
  const savedToken = await c.env.NORDGEN_KV.get(KV_DEPLOY_TOKEN_KEY);
  if (!savedToken) {
    return c.json({ error: "Unauthorized" }, 401);
  }

  const middleware = bearerAuth<{ Bindings: Env }>({ token: savedToken });
  return middleware(c, next);
}, async (c) => {
  c.executionCtx.waitUntil(c.env.NORDGEN_KV.delete(KV_DEPLOY_TOKEN_KEY));
  c.executionCtx.waitUntil(refreshServerDatabase(c.env));
  return c.json({ success: true }, 200);
});

export { syncRoute };
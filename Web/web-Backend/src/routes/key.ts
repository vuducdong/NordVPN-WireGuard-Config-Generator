import { Hono } from "hono";
import { configRateLimit } from "../middleware/rate-limit";
import { NORDVPN_CREDENTIALS_URL, UPSTREAM_USER_AGENT } from "../constants";
import { keyValidator } from "../validation/schemas";

const keyRoute = new Hono();

keyRoute.post("/", configRateLimit(), keyValidator(), async (c) => {
  const body = c.req.valid("json");

  const upstream = await fetch(NORDVPN_CREDENTIALS_URL, {
    headers: {
      Authorization: `Bearer token:${body.token}`,
      "User-Agent": UPSTREAM_USER_AGENT,
    },
  });

  if (upstream.status === 401) {
    return c.json({ error: "Expired token" }, 401);
  }
  if (upstream.status !== 200) {
    return c.json({ error: "Upstream error" }, 503);
  }

  let credentials: { nordlynx_private_key?: string };
  try {
    credentials = await upstream.json() as { nordlynx_private_key?: string };
  } catch {
    return c.json({ error: "Internal Server Error" }, 500);
  }

  return c.json({ key: credentials.nordlynx_private_key }, 200);
});

export { keyRoute };
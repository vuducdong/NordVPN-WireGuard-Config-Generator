import { Hono } from "hono";
import { cors } from "hono/cors";
import { serversRoute } from "./routes/servers";
import { keyRoute } from "./routes/key";
import { syncRoute } from "./routes/sync";

function createApp() {
  const app = new Hono<{ Bindings: Env }>();

  app.use("*", cors({
    origin: "*",
    allowMethods: ["GET", "POST", "OPTIONS"],
    allowHeaders: ["Content-Type", "If-None-Match"],
    maxAge: 86400,
  }));

  app.route("/api/servers", serversRoute);
  app.route("/api/key", keyRoute);
  app.route("/api/sync", syncRoute);

  app.get("/", (c) => c.text("NordGen API Active", 200));

  app.notFound((c) => c.json({ error: "Not Found" }, 404));

  app.onError((err, c) => {
    console.error("Unhandled error:", err);
    return c.json({ error: "Internal Server Error" }, 500);
  });

  return app;
}

export { createApp };
import { createApp } from "./app";
import { refreshServerDatabase } from "./services/database";
import { KV_SERVERS_JSON_KEY, KV_VERSION_KEY } from "./constants";

export default {
  async scheduled(_controller: ScheduledController, env: Env, ctx: ExecutionContext): Promise<void> {
    ctx.waitUntil(refreshServerDatabase(env));
  },
  async fetch(request: Request, env: Env, ctx: ExecutionContext): Promise<Response> {
    const url = new URL(request.url);
    const path = url.pathname;

    if (path.startsWith("/api/")) {
      const app = createApp();
      return app.fetch(request, env, ctx);
    }

    const assetResponse = await env.ASSETS.fetch(request);
    if (assetResponse.status !== 200) {
      return assetResponse;
    }

    const contentType = assetResponse.headers.get("content-type");
    if (!contentType?.includes("text/html") || (!path.endsWith("/") && !path.endsWith("index.html"))) {
      return assetResponse;
    }

    const serversJson = await env.NORDGEN_KV.get(KV_SERVERS_JSON_KEY);
    const version = await env.NORDGEN_KV.get(KV_VERSION_KEY);
    if (!serversJson || !version) {
      return assetResponse;
    }

    const html = await assetResponse.text();
    const safeServersJson = serversJson.replace(/</g, "\\u003c");
    const injectionScript = `<script>window.__SERVER_LIST__=${safeServersJson};</script>`;
    const injectedHtml = html.replace("</head>", `${injectionScript}</head>`);

    const response = new Response(injectedHtml, {
      status: 200,
      headers: {
        "content-type": "text/html;charset=utf-8",
        "cache-control": "public, max-age=60, must-revalidate",
        "etag": `"${version}"`,
        "vary": "Accept-Encoding",
        "x-frame-options": "DENY",
        "x-content-type-options": "nosniff",
        "referrer-policy": "strict-origin-when-cross-origin"
      }
    });

    return response;
  }
};
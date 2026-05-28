import { app } from "./app";
import { refreshServerDatabase } from "./services/database";
import { getOrInitializeState } from "./services/memory";

export default {
  async scheduled(_controller: ScheduledController, env: Env, _ctx: ExecutionContext): Promise<void> {
    await refreshServerDatabase(env);
  },
  async fetch(request: Request, env: Env, ctx: ExecutionContext): Promise<Response> {
    const url = new URL(request.url);
    const path = url.pathname;

    if (path.startsWith("/api/")) {
      return app.fetch(request, env, ctx);
    }

    const isHtmlRequest = path.endsWith("/") || path.endsWith("index.html");
    let state = null;

    if (isHtmlRequest) {
      state = await getOrInitializeState(env);
      if (state) {
        const clientETag = request.headers.get("if-none-match");
        const currentETag = `"${state.version}"`;
        if (clientETag === currentETag || clientETag === `W/"${state.version}"`) {
          return new Response(null, {
            status: 304,
            headers: {
              "etag": currentETag,
              "cache-control": "public, max-age=60, must-revalidate",
              "vary": "Accept-Encoding"
            }
          });
        }
      }
    }

    const assetResponse = await env.ASSETS.fetch(request);
    if (assetResponse.status !== 200 || !isHtmlRequest || !state) {
      return assetResponse;
    }

    const contentType = assetResponse.headers.get("content-type");
    if (!contentType?.includes("text/html")) {
      return assetResponse;
    }

    const html = await assetResponse.text();
    const injectedHtml = html.replace("</head>", `${state.injectionScript}</head>`);

    return new Response(injectedHtml, {
      status: 200,
      headers: {
        "content-type": "text/html;charset=utf-8",
        "cache-control": "public, max-age=60, must-revalidate",
        "etag": `"${state.version}"`,
        "vary": "Accept-Encoding",
        "x-frame-options": "DENY",
        "x-content-type-options": "nosniff",
        "referrer-policy": "strict-origin-when-cross-origin"
      }
    });
  }
};
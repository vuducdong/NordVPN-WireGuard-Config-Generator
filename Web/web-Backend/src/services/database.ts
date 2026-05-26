import {
  KV_INJECTION_KEY,
  KV_VERSION_KEY,
  CUSTOM_API_URL,
} from "../constants";
import { clearMemoryCache } from "../routes/servers";

export async function refreshServerDatabase(env: Env): Promise<void> {
  const response = await fetch(CUSTOM_API_URL, {
    headers: {
      "Accept-Encoding": "gzip"
    }
  });
  
  if (!response.ok) return;

  const rawETag = response.headers.get("ETag");
  const version = rawETag ? rawETag.replace(/"/g, "") : Date.now().toString(16);

  const apiResponseText = await response.text();
  const safeServersJson = apiResponseText.replace(/</g, "\\u003c");
  const injectionScript = `<script>window.__SERVER_LIST__=${safeServersJson};</script>`;

  await Promise.all([
    env.NORDGEN_KV.put("global:api_response", apiResponseText, {
      metadata: { version }
    }),
    env.NORDGEN_KV.put(KV_INJECTION_KEY, injectionScript),
    env.NORDGEN_KV.put(KV_VERSION_KEY, version)
  ]);

  clearMemoryCache();
}
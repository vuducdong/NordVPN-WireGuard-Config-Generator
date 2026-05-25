import { rateLimiter } from "hono-rate-limiter";
import type { MiddlewareHandler } from "hono";
import { RATE_LIMIT_CONFIG, RATE_LIMIT_WINDOW_MS } from "../constants";

function extractClientIP(c: { req: { header(name: string): string | undefined } }): string {
  return c.req.header("x-client-ip") ??
    c.req.header("cf-connecting-ip") ??
    "127.0.0.1";
}

function rateLimitReachedMessage(): Record<string, string> {
  return { error: "Rate limit exceeded" };
}

let configLimiterInstance: MiddlewareHandler | null = null;

function getConfigLimiter(): MiddlewareHandler {
  if (!configLimiterInstance) {
    configLimiterInstance = rateLimiter({
      windowMs: RATE_LIMIT_WINDOW_MS,
      limit: RATE_LIMIT_CONFIG,
      keyGenerator: extractClientIP,
      message: rateLimitReachedMessage,
    });
  }
  return configLimiterInstance;
}

export function configRateLimit(): MiddlewareHandler {
  return async (c, next) => {
    const limiter = getConfigLimiter();
    return limiter(c, next);
  };
}
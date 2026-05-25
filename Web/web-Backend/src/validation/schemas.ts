import { z } from "zod";
import { zValidator } from "@hono/zod-validator";
import type { Context } from "hono";

const HEX_64_REGEX = /^[0-9a-fA-F]{64}$/;

const hexTokenSchema = z.string().regex(HEX_64_REGEX, "Invalid token");

const keyRequestBodySchema = z.object({
  token: hexTokenSchema,
});

function joinValidationMessages(
  result: { success: false; error: { issues: { message: string }[] } },
  c: Context,
): Response {
  const messages = result.error.issues.map((issue) => issue.message).join(", ");
  return c.json({ error: messages }, 400);
}

export function keyValidator() {
  return zValidator("json", keyRequestBodySchema, (result, c) => {
    if (!result.success) {
      return joinValidationMessages(result as { success: false; error: { issues: { message: string }[] } }, c);
    }
  });
}
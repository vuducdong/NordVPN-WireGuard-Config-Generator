import { z } from "zod";
import { zValidator } from "@hono/zod-validator";
import type { Context } from "hono";
import {
  KEEPALIVE_MIN,
  KEEPALIVE_MAX,
  KEEPALIVE_DEFAULT,
  DEFAULT_DNS,
} from "../constants";

const IPV4_OCTET = "(?:25[0-5]|2[0-4]\\d|[01]?\\d\\d?)";
const IPV4_REGEX = new RegExp(`^${IPV4_OCTET}(?:\\.${IPV4_OCTET}){3}$`);
const HEX_64_REGEX = /^[0-9a-fA-F]{64}$/;

const hexTokenSchema = z.string().regex(HEX_64_REGEX, "Invalid token");

const keyRequestBodySchema = z.object({
  token: hexTokenSchema,
});

const baseWireGuardSchema = z.object({
  dns: z
    .string()
    .refine((v) => {
      if (v === "") return true;
      const parts = v.split(",").map((p) => p.trim());
      return parts.every((p) => IPV4_REGEX.test(p));
    }, "Invalid DNS IP")
    .optional()
    .default(DEFAULT_DNS),
  endpoint: z
    .enum(["hostname", "station"])
    .optional()
    .default("hostname"),
  keepalive: z
    .number()
    .int()
    .min(KEEPALIVE_MIN, "Invalid keepalive")
    .max(KEEPALIVE_MAX, "Invalid keepalive")
    .optional()
    .default(KEEPALIVE_DEFAULT),
  mode: z
    .enum(["server", "client"])
    .optional()
    .default("server"),
});

const configRequestBodySchema = baseWireGuardSchema.extend({
  name: z.string().min(1, "Missing name"),
});

const batchRequestBodySchema = baseWireGuardSchema.extend({
  country: z.string().optional(),
  city: z.string().optional(),
});

export type ParsedConfigInput = z.infer<typeof configRequestBodySchema>;
export type BatchInput = z.infer<typeof batchRequestBodySchema>;

function joinValidationMessages(
  result: { success: false; error: { issues: { message: string }[] } },
  c: Context,
): Response {
  const messages = result.error.issues.map((issue) => issue.message).join(", ");
  return c.json({ error: messages }, 400);
}

export function configValidator() {
  return zValidator("json", configRequestBodySchema, (result, c) => {
    if (!result.success) {
      return joinValidationMessages(result as { success: false; error: { issues: { message: string }[] } }, c);
    }
  });
}

export function batchValidator() {
  return zValidator("json", batchRequestBodySchema, (result, c) => {
    if (!result.success) {
      return joinValidationMessages(result as { success: false; error: { issues: { message: string }[] } }, c);
    }
  });
}

export function keyValidator() {
  return zValidator("json", keyRequestBodySchema, (result, c) => {
    if (!result.success) {
      return joinValidationMessages(result as { success: false; error: { issues: { message: string }[] } }, c);
    }
  });
}
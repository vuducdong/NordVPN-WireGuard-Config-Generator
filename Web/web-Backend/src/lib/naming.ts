const NON_ALPHANUMERIC_REGEX = /[^a-z0-9]+/g;
const IDENTIFIER_UNSAFE_REGEX = /[\s#]+/g;
const DIGIT_REGEX = /\d+/;

export function normalizeName(name: string): string {
  return name.toLowerCase().replace(NON_ALPHANUMERIC_REGEX, "_").replace(/^_|_$/g, "");
}

export function sanitizeIdentifier(name: string): string {
  return name.replace(IDENTIFIER_UNSAFE_REGEX, "_");
}

export function extractNumber(name: string): string {
  const match = name.match(DIGIT_REGEX);
  return match ? match[0] : "";
}
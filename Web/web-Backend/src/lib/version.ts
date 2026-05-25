export function validateVersion(version: string): boolean {
  const dotIndex = version.indexOf(".");
  if (dotIndex < 0 || dotIndex === 0) return false;

  const major = parseInt(version.slice(0, dotIndex), 10);
  if (isNaN(major)) return false;
  if (major > 2) return true;
  if (major < 2) return false;

  const rest = version.slice(dotIndex + 1);
  const secondDot = rest.indexOf(".");
  const minorStr = secondDot >= 0 ? rest.slice(0, secondDot) : rest;
  const minor = parseInt(minorStr, 10);
  if (isNaN(minor)) return false;

  return minor >= 1;
}
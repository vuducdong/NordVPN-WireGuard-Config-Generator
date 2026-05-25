import { generate } from "lean-qr";
import { toSvgSource } from "lean-qr/extras/svg";

export function generateQRCodeSVG(text: string): string {
  const code = generate(text);
  return toSvgSource(code, {
    on: "#000000",
    off: "#ffffff",
    pad: 1,
    width: 256,
  });
}
import type { ZipEntry } from "../types";

const CRC_TABLE = new Uint32Array(256);
for (let i = 0; i < 256; i++) {
  let remainder = i;
  for (let j = 0; j < 8; j++) {
    remainder = remainder & 1 ? 0xedb88320 ^ (remainder >>> 1) : remainder >>> 1;
  }
  CRC_TABLE[i] = remainder >>> 0;
}

function computeCRC32(data: Uint8Array): number {
  let crc = 0xffffffff;
  for (let i = 0; i < data.length; i++) {
    crc = (crc >>> 8) ^ CRC_TABLE[(crc ^ data[i]) & 0xff];
  }
  return (crc ^ 0xffffffff) >>> 0;
}

export function createZipArchive(entries: ZipEntry[]): Uint8Array {
  const metadata = entries.map((entry) => {
    const nameBytes = new TextEncoder().encode(entry.name);
    const crc = computeCRC32(entry.data);
    const size = entry.data.length;
    return { nameBytes, crc, size };
  });

  let localHeaderOffset = 0;
  let centralDirectorySize = 0;

  const offsets: number[] = [];

  for (const meta of metadata) {
    const localSize = 30 + meta.nameBytes.length + meta.size;
    const centralSize = 46 + meta.nameBytes.length;
    offsets.push(localHeaderOffset);
    localHeaderOffset += localSize;
    centralDirectorySize += centralSize;
  }

  const totalSize = localHeaderOffset + centralDirectorySize + 22;
  const buffer = new Uint8Array(totalSize);
  const view = new DataView(buffer.buffer);
  let cursor = 0;

  for (let i = 0; i < entries.length; i++) {
    const entry = entries[i];
    const meta = metadata[i];

    view.setUint32(cursor, 0x04034b50, true); cursor += 4;
    view.setUint16(cursor, 10, true); cursor += 2;
    view.setUint16(cursor, 0, true); cursor += 2;
    view.setUint16(cursor, 0, true); cursor += 2;
    view.setUint16(cursor, 0x2100, true); cursor += 2;
    view.setUint16(cursor, 0x5621, true); cursor += 2;
    view.setUint32(cursor, meta.crc, true); cursor += 4;
    view.setUint32(cursor, meta.size, true); cursor += 4;
    view.setUint32(cursor, meta.size, true); cursor += 4;
    view.setUint16(cursor, meta.nameBytes.length, true); cursor += 2;
    view.setUint16(cursor, 0, true); cursor += 2;

    buffer.set(meta.nameBytes, cursor); cursor += meta.nameBytes.length;
    buffer.set(entry.data, cursor); cursor += meta.size;
  }

  for (let i = 0; i < entries.length; i++) {
    const meta = metadata[i];

    view.setUint32(cursor, 0x02014b50, true); cursor += 4;
    view.setUint16(cursor, 20, true); cursor += 2;
    view.setUint16(cursor, 10, true); cursor += 2;
    view.setUint16(cursor, 0, true); cursor += 2;
    view.setUint16(cursor, 0, true); cursor += 2;
    view.setUint16(cursor, 0x2100, true); cursor += 2;
    view.setUint16(cursor, 0x5621, true); cursor += 2;
    view.setUint32(cursor, meta.crc, true); cursor += 4;
    view.setUint32(cursor, meta.size, true); cursor += 4;
    view.setUint32(cursor, meta.size, true); cursor += 4;
    view.setUint16(cursor, meta.nameBytes.length, true); cursor += 2;
    view.setUint16(cursor, 0, true); cursor += 2;
    view.setUint16(cursor, 0, true); cursor += 2;
    view.setUint16(cursor, 0, true); cursor += 2;
    view.setUint16(cursor, 0, true); cursor += 2;
    view.setUint32(cursor, 0, true); cursor += 4;
    view.setUint32(cursor, offsets[i], true); cursor += 4;

    buffer.set(meta.nameBytes, cursor); cursor += meta.nameBytes.length;
  }

  view.setUint32(cursor, 0x06054b50, true); cursor += 4;
  view.setUint16(cursor, 0, true); cursor += 2;
  view.setUint16(cursor, 0, true); cursor += 2;
  view.setUint16(cursor, entries.length, true); cursor += 2;
  view.setUint16(cursor, entries.length, true); cursor += 2;
  view.setUint32(cursor, centralDirectorySize, true); cursor += 4;
  view.setUint32(cursor, localHeaderOffset, true); cursor += 4;
  view.setUint16(cursor, 0, true); cursor += 2;

  return buffer;
}
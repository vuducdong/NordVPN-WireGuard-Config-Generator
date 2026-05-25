export interface ServerEntry {
  station: string;
  hostname: string;
  country: string;
  city: string;
  lowCode: string;
  number: string;
  keyIndex: number;
  dedupSuffix: string;
}

export interface CompactDatabase {
  keys: string[];
  servers: Record<string, ServerEntry>;
  regions: Record<string, string[]>;
}

export interface ZipEntry {
  name: string;
  data: Uint8Array;
}
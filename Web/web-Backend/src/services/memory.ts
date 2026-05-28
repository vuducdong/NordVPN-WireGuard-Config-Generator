import { KV_INJECTION_KEY } from "../constants";

export interface MemoryState {
  version: string;
  injectionScript: string;
  apiResponse: string;
}

export let memoryState: MemoryState | null = null;

export async function getOrInitializeState(env: Env): Promise<MemoryState | null> {
  if (memoryState) {
    return memoryState;
  }

  const result = await env.NORDGEN_KV.getWithMetadata<{ version: string }>("global:api_response");
  if (!result.value || !result.metadata?.version) {
    return null;
  }
  
  const version = result.metadata.version;
  const apiResponse = result.value;
  const injectionScript = await env.NORDGEN_KV.get(KV_INJECTION_KEY);
  
  if (!injectionScript) {
    return null;
  }

  memoryState = { version, injectionScript, apiResponse };
  return memoryState;
}

export function setMemoryState(state: MemoryState): void {
  memoryState = state;
}
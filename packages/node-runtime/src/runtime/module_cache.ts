import { log } from "../logging/logger.js";

const moduleCache = new Map<string, Record<string, unknown>>();

export async function importModule(
  fileUrl: URL
): Promise<Record<string, unknown>> {
  const key = fileUrl.href;
  const cached = moduleCache.get(key);
  if (cached) {
    log("module_cache_hit", { module_url: key });
    return cached;
  }

  log("module_import", { module_url: key });
  const mod = (await import(key)) as Record<string, unknown>;
  moduleCache.set(key, mod);
  return mod;
}

/** Test helper — reset module cache between tests. */
export function clearModuleCache(): void {
  moduleCache.clear();
}

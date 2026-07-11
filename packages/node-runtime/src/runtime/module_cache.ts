import { Effect } from "effect";

const moduleCache = new Map<string, Record<string, unknown>>();

export const importModule = Effect.fn("Runtime.importModule")(
  function* (fileUrl: URL) {
    const key = fileUrl.href;
    yield* Effect.annotateCurrentSpan("module_url", key);
    const cached = moduleCache.get(key);
    if (cached) {
      yield* Effect.annotateCurrentSpan("cache_hit", true);
      yield* Effect.logDebug("module_cache_hit").pipe(
        Effect.annotateLogs({ event: "module_cache_hit", module_url: key })
      );
      return cached;
    }

    yield* Effect.logDebug("module_import").pipe(
      Effect.annotateLogs({ event: "module_import", module_url: key })
    );
    const mod = yield* Effect.tryPromise({
      try: () => import(key) as Promise<Record<string, unknown>>,
      catch: (cause) =>
        cause instanceof Error ? cause : new Error(String(cause)),
    });
    moduleCache.set(key, mod);
    return mod;
  }
);

/** Test helper — reset module cache between tests. */
export function clearModuleCache(): void {
  moduleCache.clear();
}

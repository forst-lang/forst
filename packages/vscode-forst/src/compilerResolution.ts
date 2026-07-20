import type { ForstExtensionConfig } from "./configTypes.js";

interface CompilerResolutionCacheEntry {
  key: string;
  path?: string;
  error?: Error;
}

let compilerResolutionCache: CompilerResolutionCacheEntry | undefined;

/** Cache key for @forst/cli resolution (invalidated when compiler settings change). */
export function compilerResolutionCacheKey(cfg: ForstExtensionConfig): string {
  return JSON.stringify({
    forstPath: cfg.forstPath,
    downloadCompiler: cfg.downloadCompiler,
    preferLatestCompilerRelease: cfg.preferLatestCompilerRelease,
  });
}

/** Clears cached compiler resolution (success or failure). */
export function clearCompilerResolutionCache(): void {
  compilerResolutionCache = undefined;
}

/** Last successfully resolved compiler path from @forst/cli, if any. */
export function getCachedCompilerPath(): string | undefined {
  return compilerResolutionCache?.path;
}

/** Last compiler resolution failure, if any. */
export function getCachedCompilerError(): Error | undefined {
  return compilerResolutionCache?.error;
}

/** Returns the cached path/error for `cacheKey`, or undefined on a cache miss or key mismatch (settings changed). */
export function readCompilerResolutionCache(
  cacheKey: string
): { path: string } | { error: Error } | undefined {
  if (compilerResolutionCache?.key !== cacheKey) {
    return undefined;
  }
  if (compilerResolutionCache.path) {
    return { path: compilerResolutionCache.path };
  }
  if (compilerResolutionCache.error) {
    return { error: compilerResolutionCache.error };
  }
  return undefined;
}

/** Records a successful compiler resolution so subsequent calls with the same cacheKey skip re-resolving. */
export function writeCompilerResolutionCachePath(
  cacheKey: string,
  resolvedPath: string
): void {
  compilerResolutionCache = { key: cacheKey, path: resolvedPath };
}

/** Records a failed compiler resolution so subsequent calls with the same cacheKey surface the same error instead of retrying. */
export function writeCompilerResolutionCacheError(
  cacheKey: string,
  error: Error
): void {
  compilerResolutionCache = { key: cacheKey, error };
}

/**
 * When downloads are enabled, CLI failures must propagate — do not fall back to PATH `forst`.
 */
export function shouldFallbackToPathOnCliError(downloadCompiler: boolean): boolean {
  return !downloadCompiler;
}

/** Applies CLI failure policy: rethrow when downloads are enabled, else PATH fallback. */
export function handleCliResolutionFailure(
  e: unknown,
  pathFallback: string,
  downloadCompiler: boolean
): string {
  const err = e instanceof Error ? e : new Error(String(e));
  if (!shouldFallbackToPathOnCliError(downloadCompiler)) {
    throw err;
  }
  return pathFallback;
}

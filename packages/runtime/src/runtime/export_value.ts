/**
 * Resolves a named export from a dynamically imported module because ESM
 * loaders expose values on both top-level keys and nested `default` objects.
 */
export function resolveExportValue(
  mod: Record<string, unknown>,
  exportName: string
): unknown {
  if (Object.prototype.hasOwnProperty.call(mod, exportName)) {
    return mod[exportName];
  }
  const nested = mod.default;
  if (nested && typeof nested === "object" && !Array.isArray(nested)) {
    return (nested as Record<string, unknown>)[exportName];
  }
  return undefined;
}

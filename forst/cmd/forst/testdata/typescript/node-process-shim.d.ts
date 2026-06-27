/**
 * Minimal `process.env` for E2E typecheck when compilerOptions.types is [] (no @types/node).
 * Use `declare var` so this stays a single scope binding; do not load alongside @types/node.
 */
declare var process: {
  env: Record<string, string | undefined>;
};

import type { ForstConfig, SidecarRuntime, VersionCheckMode } from "./types";

function parsePortEnv(): number | undefined {
  const v = process.env.FORST_PORT;
  if (v === undefined || v === "") {
    return undefined;
  }
  const n = parseInt(v, 10);
  if (!Number.isFinite(n) || n < 1 || n > 65535) {
    return undefined;
  }
  return n;
}

/**
 * Merges {@link ForstConfig} with process env (`FORST_PORT`, `FORST_DIR`, `FORST_DEV_URL`, `FORST_SKIP_SPAWN`).
 * Call from {@link ForstSidecar.start} so values reflect the environment at startup time.
 */
export function mergeForstSidecarEnv(partial: Partial<ForstConfig>): ForstConfig {
  const devServerUrl =
    partial.devServerUrl ?? process.env.FORST_DEV_URL ?? undefined;

  let sidecarRuntime: SidecarRuntime = partial.sidecarRuntime ?? "spawn";
  if (partial.sidecarRuntime === undefined) {
    const skip = process.env.FORST_SKIP_SPAWN === "1";
    const hasUrl = devServerUrl != null && devServerUrl !== "";
    if (skip || hasUrl) {
      sidecarRuntime = "connect";
    }
  }

  const port = partial.port ?? parsePortEnv() ?? 8080;
  const host = partial.host ?? "localhost";
  const forstDir = partial.forstDir ?? process.env.FORST_DIR;

  const versionCheck: VersionCheckMode =
    partial.versionCheck ?? "warn";

  const downloadCompiler =
    partial.downloadCompiler !== undefined
      ? partial.downloadCompiler
      : process.env.FORST_DOWNLOAD_COMPILER === "1" ||
        process.env.FORST_DOWNLOAD_COMPILER === "true";

  return {
    ...partial,
    mode: partial.mode ?? "development",
    port,
    host,
    logLevel: partial.logLevel ?? "info",
    forstDir,
    devServerUrl: devServerUrl ?? undefined,
    sidecarRuntime,
    versionCheck,
    downloadCompiler,
  };
}

/** Normalizes a dev server base URL (no trailing slash) for HTTP clients. */
export function normalizeDevServerBaseUrl(url: string): string {
  return url.replace(/\/+$/, "");
}

/** Parses host and port from an HTTP(S) URL for {@link ServerInfo}. */
export function parseDevServerUrlParts(url: string): { host: string; port: number } {
  const u = new URL(url);
  const port =
    u.port !== ""
      ? parseInt(u.port, 10)
      : u.protocol === "https:"
        ? 443
        : 80;
  return { host: u.hostname, port };
}

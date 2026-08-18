import { dirname } from "node:path";
import { existsSync, readFileSync } from "node:fs";
import { join } from "node:path";
import { getCompilerModuleDirForVersion } from "./compiler-module.js";
import { shouldDownloadPluginsForSpawn } from "./ftconfig-plugins.js";
import {
  ensurePluginsForVersion,
  pluginsReady,
  prependPluginDirsToPath,
} from "./plugins-module.js";
import {
  resolveForstBinaryDetailed,
  type ResolveForstBinaryOptions,
} from "./resolve.js";

export interface BuildForstSpawnEnvOptions extends ResolveForstBinaryOptions {
  /** Forst argv slice after the binary name (default: empty). */
  argv?: string[];
  /** Working directory for ftconfig discovery (default: process.cwd()). */
  cwd?: string;
}

function compilerModuleReady(moduleDir: string): boolean {
  return existsSync(join(moduleDir, "cmd", "forst"));
}

/** Resolves binary path and env for spawning forst (sets FORST_GOMOD_ROOT when using CLI cache). */
export async function buildForstSpawnEnv(
  options: BuildForstSpawnEnvOptions = {}
): Promise<{ bin: string; env: NodeJS.ProcessEnv }> {
  const baseEnv = options.env ?? process.env;
  const allowDownload = options.allowDownload !== false;
  const argv = options.argv ?? [];
  const cwd = options.cwd ?? process.cwd();
  const fs = {
    existsSync: options.fs?.existsSync ?? existsSync,
    readFileSync: options.fs?.readFileSync ?? readFileSync,
  };
  const { binaryPath, version } = await resolveForstBinaryDetailed(options);
  let env: NodeJS.ProcessEnv = { ...baseEnv };
  if (!env.FORST_GOMOD_ROOT?.trim() && version) {
    const moduleDir = getCompilerModuleDirForVersion(version, {
      env,
      homedirFn: options.homedirFn,
    });
    if (compilerModuleReady(moduleDir)) {
      env.FORST_GOMOD_ROOT = moduleDir;
    }
  }

  const downloadPlugins =
    version &&
    allowDownload &&
    shouldDownloadPluginsForSpawn({ argv, cwd, fs });

  if (downloadPlugins) {
    await ensurePluginsForVersion({
      version,
      env,
      fetchFn: options.fetchImpl,
      fs: options.fs,
      homedirFn: options.homedirFn,
    });
  }

  const binDir = dirname(binaryPath);
  if (pluginsReady(binDir, options.fs)) {
    env = prependPluginDirsToPath(env, binDir);
  }
  return { bin: binaryPath, env };
}

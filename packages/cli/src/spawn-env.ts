import { dirname } from "node:path";
import { existsSync } from "node:fs";
import { join } from "node:path";
import { getCompilerModuleDirForVersion } from "./compiler-module.js";
import {
  ensurePluginsForVersion,
  pluginsReady,
  prependPluginDirsToPath,
} from "./plugins-module.js";
import {
  resolveForstBinaryDetailed,
  type ResolveForstBinaryOptions,
} from "./resolve.js";

function compilerModuleReady(moduleDir: string): boolean {
  return existsSync(join(moduleDir, "cmd", "forst"));
}

/** Resolves binary path and env for spawning forst (sets FORST_GOMOD_ROOT when using CLI cache). */
export async function buildForstSpawnEnv(
  options: ResolveForstBinaryOptions = {}
): Promise<{ bin: string; env: NodeJS.ProcessEnv }> {
  const baseEnv = options.env ?? process.env;
  const allowDownload = options.allowDownload !== false;
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

  if (version && allowDownload) {
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

import { existsSync } from "node:fs";
import { join } from "node:path";
import { getCompilerModuleDirForVersion } from "./compiler-module.js";
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
  const { binaryPath, version } = await resolveForstBinaryDetailed(options);
  const env: NodeJS.ProcessEnv = { ...baseEnv };
  if (!env.FORST_GOMOD_ROOT?.trim() && version) {
    const moduleDir = getCompilerModuleDirForVersion(version, {
      env,
      homedirFn: options.homedirFn,
    });
    if (compilerModuleReady(moduleDir)) {
      env.FORST_GOMOD_ROOT = moduleDir;
    }
  }
  return { bin: binaryPath, env };
}

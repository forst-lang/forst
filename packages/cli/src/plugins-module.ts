import { execFileSync } from "node:child_process";
import {
  chmodSync,
  existsSync,
  mkdirSync,
  readFileSync,
  renameSync,
  statSync,
  unlinkSync,
  writeFileSync,
} from "node:fs";
import { mkdir, mkdtemp, rm, writeFile } from "node:fs/promises";
import { delimiter } from "node:path";
import { tmpdir } from "node:os";
import { join } from "node:path";
import {
  getPluginsArtifactName,
  OFFICIAL_PLUGIN_COMMANDS,
} from "./artifact.js";
import { CompilerBinaryDownloadFailed } from "./errors.js";
import { fetchWithRetry, type FetchImpl } from "./http.js";
import {
  getCompilerCacheDirForVersion,
  type ResolveForstBinaryFs,
} from "./resolve.js";
import { buildPluginsArtifactDownloadUrl } from "./urls.js";

const defaultFs: ResolveForstBinaryFs = {
  existsSync,
  mkdirSync,
  readFileSync,
  writeFileSync,
  chmodSync,
  renameSync,
  unlinkSync,
  statSync,
};

/** Cache directory where plugin binaries for one compiler version are extracted. */
export function getPluginsDirForVersion(
  version: string,
  options?: { env?: NodeJS.ProcessEnv; homedirFn?: () => string }
): string {
  return getCompilerCacheDirForVersion(version, options);
}

function pluginBinaryName(base: string): string {
  return process.platform === "win32" ? `${base}.exe` : base;
}

/** True when at least one official plugin binary exists in dir. */
export function pluginsReady(
  dir: string,
  fs: ResolveForstBinaryFs = defaultFs
): boolean {
  for (const name of OFFICIAL_PLUGIN_COMMANDS) {
    if (fs.existsSync(join(dir, pluginBinaryName(name)))) {
      return true;
    }
  }
  return false;
}

async function extractPluginsArchive(
  archivePath: string,
  destDir: string
): Promise<void> {
  await mkdir(destDir, { recursive: true });
  if (archivePath.endsWith(".zip")) {
    if (process.platform === "win32") {
      execFileSync(
        "powershell",
        [
          "-NoProfile",
          "-Command",
          `Expand-Archive -LiteralPath '${archivePath.replace(/'/g, "''")}' -DestinationPath '${destDir.replace(/'/g, "''")}' -Force`,
        ],
        { stdio: "pipe" }
      );
      return;
    }
    execFileSync("unzip", ["-o", archivePath, "-d", destDir], {
      stdio: "pipe",
    });
    return;
  }
  execFileSync("tar", ["-xzf", archivePath, "-C", destDir], {
    stdio: "pipe",
  });
}

export interface EnsurePluginsOptions {
  version: string;
  env?: NodeJS.ProcessEnv;
  fetchFn?: FetchImpl;
  fs?: ResolveForstBinaryFs;
  homedirFn?: () => string;
}

/**
 * Downloads and extracts official semantic plugin binaries into the compiler
 * version cache directory when missing. Missing release assets (404) are ignored
 * so older compiler versions keep working.
 */
export async function ensurePluginsForVersion(
  options: EnsurePluginsOptions
): Promise<string> {
  const fs = options.fs ?? defaultFs;
  const pluginsDir = getPluginsDirForVersion(options.version, {
    env: options.env,
    homedirFn: options.homedirFn,
  });
  if (pluginsReady(pluginsDir, fs)) {
    return pluginsDir;
  }

  const fetchFn = options.fetchFn ?? fetch;
  const artifact = getPluginsArtifactName(process.platform, process.arch);
  const url = buildPluginsArtifactDownloadUrl(options.version, artifact);

  let response: Response;
  try {
    response = await fetchWithRetry(fetchFn, url);
  } catch (e: unknown) {
    return pluginsDir;
  }
  if (response.status === 404) {
    return pluginsDir;
  }
  if (!response.ok) {
    throw new CompilerBinaryDownloadFailed(
      `Failed to download Forst plugins from ${url}: HTTP ${response.status}`
    );
  }

  const tmpRoot = await mkdtemp(join(tmpdir(), "forst-plugins-"));
  const archivePath = join(tmpRoot, artifact);
  try {
    const buf = Buffer.from(await response.arrayBuffer());
    await writeFile(archivePath, buf);
    await extractPluginsArchive(archivePath, pluginsDir);
    if (!pluginsReady(pluginsDir, fs)) {
      throw new CompilerBinaryDownloadFailed(
        `Forst plugins at ${pluginsDir} are missing after extract`
      );
    }
    if (process.platform !== "win32") {
      for (const name of OFFICIAL_PLUGIN_COMMANDS) {
        const path = join(pluginsDir, name);
        if (fs.existsSync(path)) {
          fs.chmodSync(path, 0o755);
        }
      }
    }
    return pluginsDir;
  } finally {
    await rm(tmpRoot, { recursive: true, force: true });
  }
}

/** Prepends plugin search directories to PATH for spawning forst. */
export function prependPluginDirsToPath(
  env: NodeJS.ProcessEnv,
  ...dirs: string[]
): NodeJS.ProcessEnv {
  const existing = env.PATH ?? env.Path ?? "";
  const parts = dirs.filter((d) => d.length > 0);
  if (parts.length === 0) {
    return env;
  }
  const merged = [...parts, existing].filter(Boolean).join(delimiter);
  if (process.platform === "win32") {
    return { ...env, Path: merged, PATH: merged };
  }
  return { ...env, PATH: merged };
}

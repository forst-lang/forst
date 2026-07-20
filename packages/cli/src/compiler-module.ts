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
import { tmpdir } from "node:os";
import { join } from "node:path";
import { CompilerBinaryDownloadFailed } from "./errors.js";
import { fetchWithRetry, type FetchImpl } from "./http.js";
import {
  getCompilerCacheDirForVersion,
  type ResolveForstBinaryFs,
} from "./resolve.js";
import { buildCompilerModuleDownloadUrl } from "./urls.js";

/** Cache directory holding the extracted Forst compiler Go module sources for one release version. */
export function getCompilerModuleDirForVersion(
  version: string,
  options?: { env?: NodeJS.ProcessEnv; homedirFn?: () => string }
): string {
  return join(getCompilerCacheDirForVersion(version, options), "module");
}

/** Filename of the downloaded module tarball for a given compiler version. */
export function getCompilerModuleArtifactName(version: string): string {
  return `forst-module-${version}.tar.gz`;
}

function compilerModuleReady(moduleDir: string, fs: ResolveForstBinaryFs): boolean {
  return fs.existsSync(join(moduleDir, "cmd", "forst"));
}

async function extractModuleTarball(
  tarballPath: string,
  destDir: string
): Promise<void> {
  await mkdir(destDir, { recursive: true });
  execFileSync("tar", ["-xzf", tarballPath, "-C", destDir], {
    stdio: "pipe",
  });
}

/** Options for ensureCompilerModuleForVersion; mirrors resolveForstBinary's fs/fetch/env injection points for testability. */
export interface EnsureCompilerModuleOptions {
  version: string;
  env?: NodeJS.ProcessEnv;
  fetchFn?: FetchImpl;
  fs?: ResolveForstBinaryFs;
  homedirFn?: () => string;
}

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

/** Downloads and extracts the Forst compiler Go module sources when missing from cache. */
export async function ensureCompilerModuleForVersion(
  options: EnsureCompilerModuleOptions
): Promise<string> {
  const fs = options.fs ?? defaultFs;
  const moduleDir = getCompilerModuleDirForVersion(options.version, {
    env: options.env,
    homedirFn: options.homedirFn,
  });
  if (compilerModuleReady(moduleDir, fs)) {
    return moduleDir;
  }

  const fetchFn = options.fetchFn ?? fetch;
  const url = buildCompilerModuleDownloadUrl(options.version);
  const response = await fetchWithRetry(fetchFn, url).catch((e: unknown) => {
    throw new CompilerBinaryDownloadFailed(
      `Failed to download Forst compiler module from ${url}`,
      { cause: e }
    );
  });
  if (!response.ok) {
    throw new CompilerBinaryDownloadFailed(
      `Failed to download Forst compiler module from ${url}: HTTP ${response.status}`
    );
  }

  const tmpRoot = await mkdtemp(join(tmpdir(), "forst-module-"));
  const tarballPath = join(
    tmpRoot,
    getCompilerModuleArtifactName(options.version)
  );
  try {
    const buf = Buffer.from(await response.arrayBuffer());
    await writeFile(tarballPath, buf);
    await extractModuleTarball(tarballPath, moduleDir);
    if (!compilerModuleReady(moduleDir, fs)) {
      throw new CompilerBinaryDownloadFailed(
        `Forst compiler module at ${moduleDir} is missing cmd/forst after extract`
      );
    }
    return moduleDir;
  } finally {
    await rm(tmpRoot, { recursive: true, force: true });
  }
}

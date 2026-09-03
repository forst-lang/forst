import * as fs from "fs";
import * as os from "os";
import * as path from "path";

export type ToolchainPathOptions = {
  home?: string;
  platform?: NodeJS.Platform;
  env?: NodeJS.ProcessEnv;
  /**
   * Optional path to the `go` binary or its containing directory.
   * Resolved and prepended first (override), then heuristic prefixes apply.
   */
  goPath?: string;
};

function pathApi(platform: NodeJS.Platform): path.PlatformPath {
  return platform === "win32" ? path.win32 : path.posix;
}

function pathDelimiter(platform: NodeJS.Platform): string {
  return platform === "win32" ? ";" : ":";
}

function pathKeyEqual(a: string, b: string, platform: NodeJS.Platform): boolean {
  return platform === "win32"
    ? a.toLowerCase() === b.toLowerCase()
    : a === b;
}

/**
 * Normalize `forst.go.path`: directory as-is, or parent of a `go` / `go.exe` file.
 * Returns undefined when empty.
 */
export function resolveGoPathDir(
  goPath: string,
  platform: NodeJS.Platform = process.platform
): string | undefined {
  const trimmed = goPath.trim();
  if (!trimmed) {
    return undefined;
  }
  const p = pathApi(platform);
  try {
    const st = fs.statSync(trimmed);
    if (st.isDirectory()) {
      return trimmed;
    }
    if (st.isFile()) {
      return p.dirname(trimmed);
    }
  } catch {
    const base = p.basename(trimmed).toLowerCase();
    if (base === "go" || base === "go.exe") {
      return p.dirname(trimmed);
    }
    return trimmed;
  }
  return undefined;
}

/** Resolve PATH / Path key (Windows env is case-insensitive; Node may expose either). */
export function pathEnvKey(env: NodeJS.ProcessEnv): string {
  const hit = Object.keys(env).find((k) => k.toUpperCase() === "PATH");
  return hit ?? "PATH";
}

/**
 * Candidate dirs GUI-spawned editors often omit from PATH.
 * Only dirs that exist are prepended (see envWithToolchainPath).
 * Covers default Go installs, Homebrew, Scoop/Chocolatey, and common version managers.
 */
export function toolchainPathPrefixes(
  homeOrOpts: string | ToolchainPathOptions = {}
): string[] {
  const opts: ToolchainPathOptions =
    typeof homeOrOpts === "string" ? { home: homeOrOpts } : homeOrOpts;
  const home = opts.home ?? os.homedir();
  const platform = opts.platform ?? process.platform;
  const env = opts.env ?? process.env;
  const p = pathApi(platform);
  const delim = pathDelimiter(platform);
  const out: string[] = [];

  const add = (dir: string | undefined | null) => {
    if (dir && dir.length > 0) {
      out.push(dir);
    }
  };

  // Explicit toolchain from the parent environment (often set in shells but not GUI apps).
  if (env.GOROOT) {
    add(p.join(env.GOROOT, "bin"));
  }
  if (env.GOPATH) {
    for (const gp of env.GOPATH.split(delim)) {
      if (gp) {
        add(p.join(gp, "bin"));
      }
    }
  }

  // Default user Go bin (GOPATH default) and user-local bins.
  add(p.join(home, "go", "bin"));
  add(p.join(home, ".local", "bin"));

  // Official go.dev installers place versioned SDKs under ~/sdk/go*.
  addGoSdkBins(home, p, out);

  // Version managers / shims (cross-platform).
  add(p.join(home, ".asdf", "shims"));
  add(p.join(home, ".local", "share", "mise", "shims"));
  add(p.join(home, ".mise", "shims"));
  add(p.join(home, ".bun", "bin"));

  if (platform === "win32") {
    addWindowsPrefixes(home, env, p, add);
  } else {
    addUnixPrefixes(add);
  }

  return dedupePreserveOrder(out, platform);
}

function addGoSdkBins(
  home: string,
  p: path.PlatformPath,
  out: string[]
): void {
  const sdkRoot = p.join(home, "sdk");
  let entries: string[];
  try {
    entries = fs.readdirSync(sdkRoot);
  } catch {
    return;
  }
  for (const name of entries) {
    if (!name.startsWith("go")) {
      continue;
    }
    out.push(p.join(sdkRoot, name, "bin"));
  }
}

function addWindowsPrefixes(
  home: string,
  env: NodeJS.ProcessEnv,
  p: path.PlatformPath,
  add: (dir: string | undefined | null) => void
): void {
  const programFiles = env.ProgramFiles ?? env.PROGRAMFILES;
  const programFilesX86 = env["ProgramFiles(x86)"] ?? env.PROGRAMFILES_X86;
  const localAppData = env.LOCALAPPDATA ?? p.join(home, "AppData", "Local");
  const chocolatey = env.ChocolateyInstall ?? "C:\\ProgramData\\chocolatey";

  add(programFiles ? p.join(programFiles, "Go", "bin") : undefined);
  add(programFilesX86 ? p.join(programFilesX86, "Go", "bin") : undefined);
  add(p.join(localAppData, "Programs", "Go", "bin"));
  // Scoop
  add(p.join(home, "scoop", "shims"));
  add(p.join(home, "scoop", "apps", "go", "current", "bin"));
  // Chocolatey
  add(p.join(chocolatey, "bin"));
  // Common winget / user-local links
  add(p.join(localAppData, "Microsoft", "WinGet", "Links"));
}

function addUnixPrefixes(add: (dir: string | undefined | null) => void): void {
  // macOS Homebrew (Apple Silicon + Intel)
  add("/opt/homebrew/bin");
  add("/opt/homebrew/sbin");
  add("/usr/local/go/bin");
  add("/usr/local/bin");
  // Linux distro / snap packages
  add("/usr/lib/go/bin");
  add("/usr/lib/golang/bin");
  add("/snap/bin");
}

function dedupePreserveOrder(
  dirs: string[],
  platform: NodeJS.Platform
): string[] {
  const seen = new Set<string>();
  const out: string[] = [];
  for (const dir of dirs) {
    const key = platform === "win32" ? dir.toLowerCase() : dir;
    if (seen.has(key)) {
      continue;
    }
    seen.add(key);
    out.push(dir);
  }
  return out;
}

/** Prepend known toolchain dirs that exist on disk and are not already on PATH. */
export function envWithToolchainPath(
  env: NodeJS.ProcessEnv = process.env,
  homeOrOpts: string | ToolchainPathOptions = {}
): NodeJS.ProcessEnv {
  const opts: ToolchainPathOptions =
    typeof homeOrOpts === "string"
      ? { home: homeOrOpts, env }
      : { ...homeOrOpts, env: homeOrOpts.env ?? env };
  const home = opts.home ?? os.homedir();
  const platform = opts.platform ?? process.platform;
  const key = pathEnvKey(env);
  const delim = pathDelimiter(platform);
  const current = env[key] ?? env.PATH ?? env.Path ?? "";
  let parts = current.split(delim).filter(Boolean);

  const overrideDir = resolveGoPathDir(opts.goPath ?? "", platform);
  if (overrideDir) {
    parts = parts.filter((part) => !pathKeyEqual(part, overrideDir, platform));
  }

  const seen = new Set(
    platform === "win32" ? parts.map((part) => part.toLowerCase()) : parts
  );
  if (overrideDir) {
    const overrideKey =
      platform === "win32" ? overrideDir.toLowerCase() : overrideDir;
    seen.add(overrideKey);
  }

  const prepend: string[] = [];
  if (overrideDir) {
    prepend.push(overrideDir);
  }
  for (const dir of toolchainPathPrefixes({
    home,
    platform,
    env: opts.env ?? env,
  })) {
    const seenKey = platform === "win32" ? dir.toLowerCase() : dir;
    if (seen.has(seenKey)) {
      continue;
    }
    try {
      if (!fs.statSync(dir).isDirectory()) {
        continue;
      }
    } catch {
      continue;
    }
    seen.add(seenKey);
    prepend.push(dir);
  }
  if (prepend.length === 0) {
    return env;
  }
  return {
    ...env,
    [key]: [...prepend, ...parts].join(delim),
  };
}

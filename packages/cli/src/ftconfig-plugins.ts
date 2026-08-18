import { isAbsolute, resolve, dirname } from "node:path";
import { OFFICIAL_PLUGIN_COMMANDS } from "./artifact.js";

type ReadFileSync = (path: string, encoding: "utf8") => string;
type ExistsSync = (path: string) => boolean;

export interface GenerateArgvInfo {
  target: string;
  dumpSemantic: boolean;
  listManifest: boolean;
  configPath?: string;
}

/** Parses `forst generate …` argv (slice after the binary name). */
export function parseGenerateArgv(args: string[]): GenerateArgvInfo | null {
  if (args[0] !== "generate") {
    return null;
  }
  let configPath: string | undefined;
  let dumpSemantic = false;
  let listManifest = false;
  const positionals: string[] = [];

  for (let i = 1; i < args.length; i++) {
    const arg = args[i]!;
    if (arg === "--dump-semantic") {
      dumpSemantic = true;
      continue;
    }
    if (arg === "--list" || arg === "--json") {
      listManifest = true;
      continue;
    }
    if (arg === "--config" || arg === "-config") {
      const next = args[++i];
      if (next) {
        configPath = next;
      }
      continue;
    }
    if (arg.startsWith("--config=")) {
      configPath = arg.slice("--config=".length);
      continue;
    }
    if (arg.startsWith("-")) {
      continue;
    }
    positionals.push(arg);
  }

  if (positionals.length === 0) {
    return null;
  }

  return {
    target: positionals[0]!,
    dumpSemantic,
    listManifest,
    configPath,
  };
}

/** Walks upward from startDir to find ftconfig.json. */
export function findFtconfigPath(
  startDir: string,
  fs: { existsSync: ExistsSync }
): string | undefined {
  let dir = resolve(startDir);
  for (;;) {
    const candidate = resolve(dir, "ftconfig.json");
    if (fs.existsSync(candidate)) {
      return candidate;
    }
    const parent = dirname(dir);
    if (parent === dir) {
      return undefined;
    }
    dir = parent;
  }
}

function readGeneratePluginCmds(
  ftconfigPath: string,
  fs: { readFileSync: ReadFileSync }
): string[] {
  const raw = fs.readFileSync(ftconfigPath, "utf8");
  const cfg = JSON.parse(raw) as {
    generate?: { plugins?: Array<{ cmd?: string }> };
  };
  const plugins = cfg.generate?.plugins;
  if (!Array.isArray(plugins) || plugins.length === 0) {
    return [];
  }
  return plugins
    .map((p) => (typeof p.cmd === "string" ? p.cmd.trim() : ""))
    .filter((cmd) => cmd.length > 0);
}

/** True when ftconfig references an official bare plugin name (not a relative/absolute path). */
export function needsOfficialPluginBundle(pluginCmds: string[]): boolean {
  const official = new Set<string>(OFFICIAL_PLUGIN_COMMANDS);
  return pluginCmds.some((cmd) => {
    if (
      cmd.startsWith(".") ||
      cmd.includes("/") ||
      cmd.includes("\\") ||
      isAbsolute(cmd)
    ) {
      return false;
    }
    return official.has(cmd);
  });
}

export interface ShouldDownloadPluginsParams {
  argv: string[];
  cwd?: string;
  fs: { existsSync: ExistsSync; readFileSync: ReadFileSync };
}

/** Whether @forst/cli should fetch the official plugins release bundle before spawn. */
export function shouldDownloadPluginsForSpawn(
  params: ShouldDownloadPluginsParams
): boolean {
  const gen = parseGenerateArgv(params.argv);
  if (!gen) {
    return false;
  }
  if (gen.dumpSemantic || gen.listManifest) {
    return false;
  }

  const ftconfigPath =
    gen.configPath && params.fs.existsSync(gen.configPath)
      ? gen.configPath
      : findFtconfigPath(resolve(params.cwd ?? process.cwd(), gen.target), params.fs);
  if (!ftconfigPath) {
    return false;
  }

  const cmds = readGeneratePluginCmds(ftconfigPath, params.fs);
  if (cmds.length === 0) {
    return false;
  }
  return needsOfficialPluginBundle(cmds);
}

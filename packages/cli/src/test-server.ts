import {
  spawn as nodeSpawn,
  type ChildProcess,
  type SpawnOptions,
} from "node:child_process";
import { existsSync, readFileSync } from "node:fs";
import { createServer as nodeCreateServer } from "node:net";
import { join, resolve } from "node:path";
import {
  ForstInvokeServerExitedEarly,
  ForstInvokeServerStartTimeout,
  ForstInvokeServerUnreachable,
} from "./errors.js";
import {
  readInvokeReadySocketPath,
  readInvokeReadyUrl,
} from "./invoke-ready.js";
import { buildForstSpawnEnv } from "./spawn-env.js";
import type { ResolveForstBinaryOptions } from "./resolve.js";
import {
  isUnixSocketSupported,
  requestOverUnixSocket,
} from "./unix-transport.js";

export type ForstInvokeServerMode = "auto" | "dev" | "embedded";

export interface StartForstInvokeServerOptions {
  root?: string;
  mode?: ForstInvokeServerMode;
  /** Embedded mode only. Defaults to main.ft under root. */
  entry?: string;
  port?: number;
  /** Attach to an existing server and never spawn. */
  baseUrl?: string;
  env?: Record<string, string>;
  /** Default 60_000. Embedded mode may pay a Go build. */
  timeoutMs?: number;
  logLevel?: "error" | "warn" | "info" | "debug" | "trace";
  onLog?: (line: string, stream: "stdout" | "stderr") => void;
  /** Optional binary resolution overrides (FORST_BINARY, cache, download). */
  resolveOptions?: ResolveForstBinaryOptions;
  /** Optional explicit ftconfig path passed to `forst dev -config`. */
  configPath?: string;
}

export interface ForstInvokeServerHandle {
  /** Loopback HTTP base URL when the server advertises TCP; empty for UDS-only. */
  readonly baseUrl: string;
  readonly port: number;
  /** Absolute Unix socket path when the server advertises UDS. */
  readonly socketPath?: string;
  readonly pid?: number;
  readonly connection: "spawn" | "connect";
  stop(): Promise<void>;
  [Symbol.asyncDispose](): Promise<void>;
}

export type SpawnFn = (
  command: string,
  args: readonly string[],
  options: SpawnOptions
) => ChildProcess;

export interface StartForstInvokeServerDeps {
  spawn?: SpawnFn;
  fetch?: typeof fetch;
  existsSync?: typeof existsSync;
  readFileSync?: typeof readFileSync;
  createServer?: typeof nodeCreateServer;
  buildSpawnEnv?: typeof buildForstSpawnEnv;
  env?: NodeJS.ProcessEnv;
  now?: () => number;
  sleep?: (ms: number) => Promise<void>;
  /** Override SIGTERM→SIGKILL grace (default 5_000). Tests may shrink this. */
  stopGraceMs?: number;
}

const DEFAULT_TIMEOUT_MS = 60_000;
const HEALTH_POLL_MS = 100;
const STOP_GRACE_MS = 5_000;
const STDERR_TAIL_MAX = 4_000;

function stripTrailingSlash(url: string): string {
  return url.replace(/\/$/, "");
}

function portFromBaseUrl(baseUrl: string): number {
  try {
    const u = new URL(baseUrl);
    if (u.port) return Number(u.port);
    return u.protocol === "https:" ? 443 : 80;
  } catch {
    return 0;
  }
}

function envInvokeUrls(env: NodeJS.ProcessEnv): string | undefined {
  const raw =
    env.FORST_INVOKE_URL?.trim() ||
    env.FORST_BASE_URL?.trim() ||
    env.FORST_DEV_URL?.trim();
  return raw ? stripTrailingSlash(raw) : undefined;
}

/** Precedence: options.baseUrl → FORST_SKIP_SPAWN → invoke.ready → env URLs. */
function resolveAttachTarget(
  options: StartForstInvokeServerOptions,
  root: string,
  deps: Required<
    Pick<StartForstInvokeServerDeps, "existsSync" | "readFileSync" | "env">
  >
): { kind: "attach"; url: string } | { kind: "spawn" } {
  if (options.baseUrl?.trim()) {
    return { kind: "attach", url: stripTrailingSlash(options.baseUrl.trim()) };
  }
  const skip =
    deps.env.FORST_SKIP_SPAWN === "1" || deps.env.FORST_SKIP_SPAWN === "true";
  const fromReady = readInvokeReadyUrl(root, {
    existsSync: deps.existsSync,
    readFileSync: deps.readFileSync,
  });
  const fromEnv = envInvokeUrls(deps.env);
  if (skip) {
    const url = fromReady ?? fromEnv;
    if (!url) {
      throw new ForstInvokeServerUnreachable(
        "FORST_SKIP_SPAWN is set but no invoke URL was found (set FORST_BASE_URL or write .forst/invoke.ready)"
      );
    }
    return { kind: "attach", url };
  }
  if (fromReady) return { kind: "attach", url: fromReady };
  if (fromEnv) return { kind: "attach", url: fromEnv };
  return { kind: "spawn" };
}

function detectMode(
  root: string,
  mode: ForstInvokeServerMode | undefined,
  deps: Required<Pick<StartForstInvokeServerDeps, "existsSync" | "readFileSync">>
): "dev" | "embedded" {
  if (mode === "dev" || mode === "embedded") return mode;
  const cfgPath = join(root, "ftconfig.json");
  if (!deps.existsSync(cfgPath)) return "dev";
  try {
    const raw = deps.readFileSync(cfgPath, "utf8");
    const cfg = JSON.parse(raw) as {
      server?: { embedded?: boolean };
      node?: { hostMode?: boolean };
    };
    if (cfg.server?.embedded === true || cfg.node?.hostMode === true) {
      return "embedded";
    }
  } catch {
    // ignore malformed ftconfig; fall back to dev
  }
  return "dev";
}

async function pickFreePort(
  createServer: typeof nodeCreateServer
): Promise<number> {
  return new Promise((resolvePort, reject) => {
    const server = createServer();
    server.listen(0, "127.0.0.1", () => {
      const addr = server.address();
      if (!addr || typeof addr === "string") {
        server.close(() => reject(new Error("failed to allocate ephemeral port")));
        return;
      }
      const { port } = addr;
      server.close((err) => {
        if (err) reject(err);
        else resolvePort(port);
      });
    });
    server.on("error", reject);
  });
}

async function waitForHealth(
  baseUrl: string,
  timeoutMs: number,
  deps: {
    fetch: typeof fetch;
    now: () => number;
    sleep: (ms: number) => Promise<void>;
    isAlive?: () => boolean;
    onDead?: () => Error;
  }
): Promise<void> {
  const healthUrl = `${stripTrailingSlash(baseUrl)}/health`;
  const deadline = deps.now() + timeoutMs;
  let lastErr: unknown;
  while (deps.now() < deadline) {
    if (deps.isAlive && !deps.isAlive()) {
      throw deps.onDead?.() ?? new Error("invoke server process exited");
    }
    try {
      const res = await deps.fetch(healthUrl);
      if (res.ok) return;
      lastErr = new Error(`health status ${res.status}`);
    } catch (e) {
      lastErr = e;
    }
    await deps.sleep(HEALTH_POLL_MS);
  }
  const detail =
    lastErr instanceof Error ? lastErr.message : String(lastErr ?? "unknown");
  throw new ForstInvokeServerStartTimeout(
    `timed out waiting for ${healthUrl} (${detail})`,
    { baseUrl, stderrTail: detail }
  );
}

async function waitForInvokeReady(
  root: string,
  fallbackBaseUrl: string,
  timeoutMs: number,
  deps: {
    fetch: typeof fetch;
    now: () => number;
    sleep: (ms: number) => Promise<void>;
    existsSync: typeof existsSync;
    readFileSync: typeof readFileSync;
    isAlive?: () => boolean;
    onDead?: () => Error;
  }
): Promise<{ baseUrl: string; socketPath?: string }> {
  const deadline = deps.now() + timeoutMs;
  let lastErr: unknown;
  while (deps.now() < deadline) {
    if (deps.isAlive && !deps.isAlive()) {
      throw deps.onDead?.() ?? new Error("invoke server process exited");
    }
    const socketPath = readInvokeReadySocketPath(root, {
      existsSync: deps.existsSync,
      readFileSync: deps.readFileSync,
    });
    const readyUrl = readInvokeReadyUrl(root, {
      existsSync: deps.existsSync,
      readFileSync: deps.readFileSync,
    });
    try {
      if (socketPath && isUnixSocketSupported()) {
        const res = await requestOverUnixSocket(socketPath, "/health", {
          method: "GET",
        });
        if (res.ok) {
          return { baseUrl: readyUrl ?? "", socketPath };
        }
        lastErr = new Error(`unix health status ${res.status}`);
      } else {
        const baseUrl = readyUrl ?? fallbackBaseUrl;
        const res = await deps.fetch(`${stripTrailingSlash(baseUrl)}/health`);
        if (res.ok) {
          return { baseUrl: stripTrailingSlash(baseUrl), socketPath };
        }
        lastErr = new Error(`health status ${res.status}`);
      }
    } catch (e) {
      lastErr = e;
    }
    await deps.sleep(HEALTH_POLL_MS);
  }
  const detail =
    lastErr instanceof Error ? lastErr.message : String(lastErr ?? "unknown");
  throw new ForstInvokeServerStartTimeout(
    `timed out waiting for invoke ready under ${root} (${detail})`,
    { baseUrl: fallbackBaseUrl, stderrTail: detail }
  );
}

function buildSpawnArgs(
  mode: "dev" | "embedded",
  options: StartForstInvokeServerOptions,
  root: string,
  port: number
): string[] {
  const logLevel = options.logLevel ?? "error";
  if (mode === "dev") {
    const args = [
      "dev",
      "-port",
      String(port),
      "-root",
      root,
      "-log-level",
      logLevel,
    ];
    if (options.configPath) {
      args.push("-config", resolve(options.configPath));
    }
    return args;
  }
  const entry = resolve(root, options.entry ?? "main.ft");
  return [
    "run",
    "-export-struct-fields",
    "-root",
    root,
    "--",
    entry,
  ];
}

function looksLikeBindFailure(stderr: string): boolean {
  return /address already in use|bind:|EADDRINUSE/i.test(stderr);
}

function captureStream(
  child: ChildProcess,
  onLog: StartForstInvokeServerOptions["onLog"] | undefined,
  sink: { stdout: string; stderr: string }
): void {
  const append = (stream: "stdout" | "stderr", chunk: Buffer | string) => {
    const text = typeof chunk === "string" ? chunk : chunk.toString("utf8");
    sink[stream] += text;
    if (sink[stream].length > STDERR_TAIL_MAX * 2) {
      sink[stream] = sink[stream].slice(-STDERR_TAIL_MAX);
    }
    if (!onLog) return;
    for (const line of text.split(/\r?\n/)) {
      if (line) onLog(line, stream);
    }
  };
  child.stdout?.on("data", (c) => append("stdout", c));
  child.stderr?.on("data", (c) => append("stderr", c));
}

function waitForExit(child: ChildProcess, timeoutMs: number): Promise<void> {
  if (child.exitCode !== null || child.signalCode) return Promise.resolve();
  return new Promise((resolveWait) => {
    let done = false;
    const finish = () => {
      if (done) return;
      done = true;
      resolveWait();
    };
    child.once("exit", finish);
    child.once("error", finish);
    setTimeout(finish, timeoutMs).unref?.();
  });
}

async function stopChild(
  child: ChildProcess,
  stopGraceMs: number = STOP_GRACE_MS
): Promise<void> {
  if (child.exitCode !== null || child.signalCode) return;
  try {
    child.kill("SIGTERM");
  } catch {
    return;
  }
  await waitForExit(child, stopGraceMs);
  if (child.exitCode === null && !child.signalCode) {
    try {
      child.kill("SIGKILL");
    } catch {
      // ignore
    }
    await waitForExit(child, 2_000);
  }
}

function childAlreadyExited(child: ChildProcess): boolean {
  return child.exitCode !== null || child.signalCode !== null;
}

function connectHandle(baseUrl: string): ForstInvokeServerHandle {
  const url = stripTrailingSlash(baseUrl);
  const port = portFromBaseUrl(url);
  const stop = async () => {};
  return {
    baseUrl: url,
    port,
    connection: "connect",
    stop,
    [Symbol.asyncDispose]: stop,
  };
}

/**
 * Starts a Forst HTTP invoke server, or attaches to one that is already running.
 * Orthogonal to `@forst/node-runtime` (Forst→Node RPC).
 */
export async function startForstInvokeServer(
  options: StartForstInvokeServerOptions = {},
  deps: StartForstInvokeServerDeps = {}
): Promise<ForstInvokeServerHandle> {
  const root = resolve(options.root ?? process.cwd());
  // options.env overlays process/deps env for both attach discovery and spawn.
  const env = { ...(deps.env ?? process.env), ...options.env };
  const exists = deps.existsSync ?? existsSync;
  const readFile = deps.readFileSync ?? readFileSync;
  const fetchFn = deps.fetch ?? fetch.bind(globalThis);
  const spawnFn = deps.spawn ?? nodeSpawn;
  const createServer = deps.createServer ?? nodeCreateServer;
  const buildSpawnEnv = deps.buildSpawnEnv ?? buildForstSpawnEnv;
  const now = deps.now ?? Date.now;
  const sleep =
    deps.sleep ?? ((ms: number) => new Promise((r) => setTimeout(r, ms)));
  const stopGraceMs = deps.stopGraceMs ?? STOP_GRACE_MS;

  const attach = resolveAttachTarget(options, root, {
    existsSync: exists,
    readFileSync: readFile,
    env,
  });
  if (attach.kind === "attach") {
    try {
      await waitForHealth(
        attach.url,
        Math.min(options.timeoutMs ?? 10_000, 10_000),
        { fetch: fetchFn, now, sleep }
      );
    } catch (e) {
      throw new ForstInvokeServerUnreachable(
        e instanceof Error ? e.message : `unreachable ${attach.url}`,
        { baseUrl: attach.url, port: portFromBaseUrl(attach.url) }
      );
    }
    return connectHandle(attach.url);
  }

  const mode = detectMode(root, options.mode, {
    existsSync: exists,
    readFileSync: readFile,
  });
  const timeoutMs = options.timeoutMs ?? DEFAULT_TIMEOUT_MS;
  const { bin, env: spawnEnvBase } = await buildSpawnEnv({
    ...options.resolveOptions,
    env: { ...env, ...options.env },
  });

  let lastBindFailure: string | undefined;
  const preferUnix = isUnixSocketSupported() && options.port === undefined;
  for (let attempt = 0; attempt < 2; attempt++) {
    const port = options.port ?? (await pickFreePort(createServer));
    const argv = buildSpawnArgs(mode, options, root, port);
    const fallbackBaseUrl = `http://127.0.0.1:${port}`;
    const childEnv: NodeJS.ProcessEnv = {
      ...spawnEnvBase,
      ...options.env,
      FORST_BOUNDARY_ROOT: root,
    };
    // Unix is the default local transport. Only force a TCP port when the
    // caller asked for one or the platform has no AF_UNIX support.
    if (mode === "embedded" && !preferUnix) {
      childEnv.FORST_INVOKE_PORT = String(port);
      childEnv.FORST_INVOKE_TRANSPORT = "tcp";
    }

    const sink = { stdout: "", stderr: "" };
    const child = spawnFn(bin, argv, {
      cwd: root,
      env: childEnv,
      stdio: ["ignore", "pipe", "pipe"],
    });
    captureStream(child, options.onLog, sink);

    let exitedEarly = childAlreadyExited(child);
    let exitCode: number | null = child.exitCode;
    child.once("exit", (code) => {
      exitedEarly = true;
      exitCode = code;
    });
    child.once("error", () => {
      exitedEarly = true;
    });
    // Spawn may have exited synchronously before listeners attached.
    if (childAlreadyExited(child)) {
      exitedEarly = true;
      exitCode = child.exitCode;
    }

    let ready: { baseUrl: string; socketPath?: string };
    try {
      ready = await waitForInvokeReady(root, fallbackBaseUrl, timeoutMs, {
        fetch: fetchFn,
        now,
        sleep,
        existsSync: exists,
        readFileSync: readFile,
        isAlive: () => !exitedEarly && !childAlreadyExited(child),
        onDead: () =>
          new ForstInvokeServerExitedEarly(
            `forst ${mode} exited before ready (code=${exitCode ?? child.exitCode ?? "?"})`,
            {
              mode,
              port,
              argv,
              exitCode: exitCode ?? child.exitCode,
              stderrTail: sink.stderr.slice(-STDERR_TAIL_MAX),
            }
          ),
      });
    } catch (err) {
      await stopChild(child, stopGraceMs);
      const fromErr =
        err instanceof ForstInvokeServerExitedEarly ? err.stderrTail ?? "" : "";
      const stderrTail =
        sink.stderr.slice(-STDERR_TAIL_MAX) || fromErr;
      if (
        attempt === 0 &&
        options.port === undefined &&
        (looksLikeBindFailure(stderrTail) || looksLikeBindFailure(sink.stdout))
      ) {
        lastBindFailure = stderrTail;
        continue;
      }
      if (err instanceof ForstInvokeServerExitedEarly) {
        throw err;
      }
      if (err instanceof ForstInvokeServerStartTimeout) {
        throw new ForstInvokeServerStartTimeout(err.message, {
          mode,
          port,
          argv,
          baseUrl: fallbackBaseUrl,
          stderrTail: stderrTail || lastBindFailure,
        });
      }
      throw err;
    }

    let stopped = false;
    const stop = async () => {
      if (stopped) return;
      stopped = true;
      await stopChild(child, stopGraceMs);
    };
    return {
      baseUrl: ready.baseUrl || fallbackBaseUrl,
      port,
      socketPath: ready.socketPath,
      pid: child.pid,
      connection: "spawn",
      stop,
      [Symbol.asyncDispose]: stop,
    };
  }

  throw new ForstInvokeServerExitedEarly(
    "failed to bind invoke port after retry",
    { mode, stderrTail: lastBindFailure }
  );
}

/** Exported for unit tests. */
export const __test__ = {
  detectMode,
  resolveAttachTarget,
  buildSpawnArgs,
  looksLikeBindFailure,
  portFromBaseUrl,
};

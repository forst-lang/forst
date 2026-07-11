import * as fs from "node:fs";
import * as net from "node:net";
import * as path from "node:path";
import type { AddressInfo } from "node:net";
import { log } from "./logging/logger.js";
import { startRpcServer } from "./rpc/server.js";

const envHostEnabled = "FORST_NODE_HOST";
const envSocketPath = "FORST_NODE_SOCKET";
const envReadyPath = "FORST_NODE_HOST_READY";

export type HostReadyPhase = "listening" | "app";

export interface HostOptions {
  socketPath?: string;
  readyPath?: string;
  /** When true, listen on the socket but defer the ready file until signalForstAppReady(). */
  deferAppReady?: boolean;
}

export interface HostHandle {
  socketPath: string;
  close(): Promise<void>;
}

let startPromise: Promise<HostHandle> | null = null;
let activeServer: net.Server | null = null;
let activeConnection: net.Socket | null = null;
let activeReadyPath = "";
let activeSocketPath = "";
let appReadySignaled = false;

function hostEnabled(): boolean {
  return process.env[envHostEnabled] === "1";
}

function isWindows(): boolean {
  return process.platform === "win32";
}

function writeReadyFile(
  readyPath: string,
  socketPath: string,
  phase: HostReadyPhase
): void {
  writeReadyFilePayload(readyPath, {
    pid: process.pid,
    socket: socketPath,
    phase,
    tcpPort: isWindows() ? Number(socketPath.split(":").pop()) : undefined,
  });
}

function writeReadyFilePayload(
  readyPath: string,
  payload: Record<string, unknown>
): void {
  fs.writeFileSync(readyPath, `${JSON.stringify(payload)}\n`, { encoding: "utf8" });
}

function chmodSocket(socketPath: string): void {
  if (isWindows()) {
    return;
  }
  try {
    fs.chmodSync(socketPath, 0o600);
  } catch {
    // best effort
  }
}

async function listenUnix(socketPath: string): Promise<net.Server> {
  const dir = path.dirname(socketPath);
  fs.mkdirSync(dir, { recursive: true, mode: 0o750 });
  if (fs.existsSync(socketPath)) {
    try {
      fs.unlinkSync(socketPath);
    } catch {
      // stale socket may be removed by Go before spawn
    }
  }

  return new Promise((resolve, reject) => {
    const server = net.createServer();
    server.on("error", reject);
    server.listen(socketPath, () => {
      chmodSocket(socketPath);
      resolve(server);
    });
  });
}

async function listenTcp(): Promise<{ server: net.Server; port: number }> {
  return new Promise((resolve, reject) => {
    const server = net.createServer();
    server.on("error", reject);
    server.listen(0, "127.0.0.1", () => {
      const addr = server.address() as AddressInfo;
      resolve({ server, port: addr.port });
    });
  });
}

function attachConnectionHandler(server: net.Server): void {
  server.on("connection", (conn) => {
    if (activeConnection) {
      log("host_reject_duplicate_client", { pid: process.pid });
      conn.destroy();
      return;
    }
    activeConnection = conn;
    log("host_client_connected", { pid: process.pid });
    conn.on("close", () => {
      if (activeConnection === conn) {
        activeConnection = null;
      }
    });
    void startRpcServer(conn, conn, { exitProcessOnShutdown: false }).catch((err) => {
      log("host_rpc_fatal", {
        message: err instanceof Error ? err.message : String(err),
      });
    });
  });
}

export function resetHostForTest(): void {
  if (activeConnection) {
    activeConnection.destroy();
    activeConnection = null;
  }
  if (activeServer) {
    activeServer.close();
    activeServer = null;
  }
  startPromise = null;
  activeReadyPath = "";
  activeSocketPath = "";
  appReadySignaled = false;
}

/**
 * Marks app initialization complete and writes the ready file with phase "app".
 * Idempotent. Requires a prior startForstNodeHost() with deferAppReady or a ready path.
 */
export async function signalForstAppReady(): Promise<void> {
  if (!hostEnabled()) {
    log("host_app_ready_skip", { reason: "FORST_NODE_HOST unset" });
    return;
  }
  if (appReadySignaled) {
    return;
  }
  if (!activeReadyPath) {
    throw new Error(
      "signalForstAppReady: host not started or ready path unset; call startForstNodeHost first"
    );
  }
  if (!activeSocketPath) {
    throw new Error("signalForstAppReady: host socket path unset");
  }

  writeReadyFile(activeReadyPath, activeSocketPath, "app");
  appReadySignaled = true;
  log("host_app_ready", { socket: activeSocketPath, pid: process.pid });
}

/**
 * No-op unless FORST_NODE_HOST=1. Idempotent. Logs to stderr only. Single client connection.
 */
export function startForstNodeHost(options: HostOptions = {}): Promise<HostHandle> {
  if (!hostEnabled()) {
    log("host_skip", { reason: "FORST_NODE_HOST unset" });
    return Promise.resolve({
      socketPath: "",
      close: async () => {},
    });
  }

  if (startPromise) {
    return startPromise;
  }

  startPromise = (async () => {
    const readyPath =
      options.readyPath ?? process.env[envReadyPath] ?? "";
    let socketPath = options.socketPath ?? process.env[envSocketPath] ?? "";
    const deferAppReady = options.deferAppReady ?? false;

    let server: net.Server;
    if (isWindows()) {
      const tcp = await listenTcp();
      server = tcp.server;
      socketPath = `tcp://127.0.0.1:${tcp.port}`;
    } else {
      if (!socketPath) {
        throw new Error("FORST_NODE_SOCKET is required on Unix");
      }
      server = await listenUnix(socketPath);
    }

    activeServer = server;
    activeReadyPath = readyPath;
    activeSocketPath = socketPath;
    attachConnectionHandler(server);

    if (readyPath && !deferAppReady) {
      writeReadyFile(readyPath, socketPath, "app");
      appReadySignaled = true;
    }

    log("host_listening", {
      socket: socketPath,
      pid: process.pid,
      defer_app_ready: deferAppReady,
    });

    return {
      socketPath,
      close: async () => {
        if (activeConnection) {
          activeConnection.destroy();
          activeConnection = null;
        }
        await new Promise<void>((resolve, reject) => {
          if (!activeServer) {
            resolve();
            return;
          }
          activeServer.close((err) => {
            if (err) {
              reject(err);
              return;
            }
            resolve();
          });
        });
        activeServer = null;
        if (!isWindows() && socketPath && fs.existsSync(socketPath)) {
          try {
            fs.unlinkSync(socketPath);
          } catch {
            // best effort
          }
        }
        if (readyPath && fs.existsSync(readyPath)) {
          try {
            fs.unlinkSync(readyPath);
          } catch {
            // best effort
          }
        }
        activeReadyPath = "";
        activeSocketPath = "";
        appReadySignaled = false;
      },
    };
  })();

  return startPromise;
}

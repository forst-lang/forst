import { createReadStream } from "node:fs";
import type { Readable } from "node:stream";

import type { InvokeAuthState } from "./invoke-auth.js";
import { type AuthHandoff } from "./invoke-auth-handoff.js";
import {
  readInvokeReadySocketPath,
  readInvokeReadyUrl,
} from "./invoke-ready.js";

export const envInvokeAuthRecvFd = "FORST_INVOKE_AUTH_RECV_FD";

interface AuthHandoffPayload {
  generation?: number;
  token?: string;
}

let hostInvokeAuth: InvokeAuthState | undefined;
let hostInvokeAuthListenerStarted = false;

function parseHandoffLine(line: string): AuthHandoff {
  let payload: AuthHandoffPayload;
  try {
    payload = JSON.parse(line) as AuthHandoffPayload;
  } catch {
    throw new Error("invoke auth handoff: invalid JSON");
  }
  if (
    payload.generation === undefined ||
    typeof payload.generation !== "number" ||
    !Number.isSafeInteger(payload.generation) ||
    payload.generation < 0 ||
    typeof payload.token !== "string" ||
    payload.token.trim() === ""
  ) {
    throw new Error("invoke auth handoff: missing generation or token");
  }
  const token = Uint8Array.from(Buffer.from(payload.token, "base64url"));
  if (!token.length) {
    throw new Error("invoke auth handoff: empty token");
  }
  return { generation: payload.generation, token };
}

function storeHostInvokeAuth(handoff: AuthHandoff): void {
  hostInvokeAuth?.token.fill(0);
  hostInvokeAuth = {
    generation: handoff.generation,
    token: handoff.token,
  };
}

async function consumeHostInvokeAuthStream(stream: Readable): Promise<void> {
  let buffer = "";
  for await (const chunk of stream) {
    buffer += chunk.toString();
    for (;;) {
      const newline = buffer.indexOf("\n");
      if (newline < 0) {
        break;
      }
      const line = buffer.slice(0, newline).trim();
      buffer = buffer.slice(newline + 1);
      if (line === "") {
        continue;
      }
      try {
        storeHostInvokeAuth(parseHandoffLine(line));
      } catch {
        // ignore malformed handoff lines
      }
    }
  }
}

/**
 * Reads invoke auth handoff lines from `FORST_INVOKE_AUTH_RECV_FD` when set by
 * forst dev host mode. Safe to call multiple times; starts at most one listener.
 */
export function startHostInvokeAuthRecvListener(): void {
  if (hostInvokeAuthListenerStarted) {
    return;
  }
  hostInvokeAuthListenerStarted = true;
  const raw = process.env[envInvokeAuthRecvFd]?.trim();
  if (!raw) {
    return;
  }
  const fd = Number(raw);
  if (!Number.isInteger(fd) || fd < 3) {
    return;
  }
  const stream = createReadStream(null as unknown as string, {
    fd,
    autoClose: false,
  });
  void consumeHostInvokeAuthStream(stream).catch(() => {
    // host relay closed
  });
}

/** Returns auth delivered by {@link startHostInvokeAuthRecvListener}, if any. */
export function resolveHostInvokeAuthHandoff():
  | InvokeAuthState
  | undefined {
  return hostInvokeAuth;
}

/** Resets cached host auth (tests). */
export function resetHostInvokeAuthHandoffForTest(): void {
  hostInvokeAuth?.token.fill(0);
  hostInvokeAuth = undefined;
  hostInvokeAuthListenerStarted = false;
}

/**
 * Sets connect-mode env for embedded invoke: skip spawn, boundary root, prefer UDS.
 */
export function prepareConnectInvokeEnv(boundaryRoot?: string): string {
  const root = boundaryRoot?.trim() || process.cwd();
  process.env.FORST_SKIP_SPAWN = "1";
  process.env.FORST_BOUNDARY_ROOT = root;

  const readyUrl = readInvokeReadyUrl(root);
  const socketPath = readInvokeReadySocketPath(root);
  if (readyUrl) {
    process.env.FORST_BASE_URL = readyUrl;
  } else if (socketPath) {
    delete process.env.FORST_BASE_URL;
    delete process.env.FORST_INVOKE_URL;
    delete process.env.FORST_DEV_URL;
  } else if (!process.env.FORST_BASE_URL) {
    process.env.FORST_BASE_URL = "http://127.0.0.1:6321";
  }
  return root;
}

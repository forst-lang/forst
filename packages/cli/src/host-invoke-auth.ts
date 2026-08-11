import { createReadStream } from "node:fs";
import type { Readable } from "node:stream";

import { DEFAULT_EMBEDDED_INVOKE_BASE_URL } from "./constants.js";
import type { InvokeAuthState } from "./invoke-auth.js";
import { type AuthHandoff } from "./invoke-auth-handoff.js";
import {
  readInvokeReadySocketPath,
  readInvokeReadyUrl,
} from "./invoke-ready.js";

export const envInvokeAuthRecvFd = "FORST_INVOKE_AUTH_RECV_FD";

let hostInvokeAuth: InvokeAuthState | undefined;
let hostInvokeAuthListenerStarted = false;

/**
 * Parses one newline-delimited JSON handoff line into {@link AuthHandoff}.
 * Validates the JSON root as a non-null object before reading fields so values
 * such as `null` become handoff errors instead of TypeErrors.
 */
function parseHandoffLine(line: string): AuthHandoff {
  let parsed: unknown;
  try {
    parsed = JSON.parse(line) as unknown;
  } catch {
    throw new Error("invoke auth handoff: invalid JSON");
  }
  if (parsed === null || typeof parsed !== "object" || Array.isArray(parsed)) {
    throw new Error("invoke auth handoff: missing generation or token");
  }
  const record = parsed as Record<string, unknown>;
  const generation = record.generation;
  const tokenRaw = record.token;
  if (
    typeof generation !== "number" ||
    !Number.isSafeInteger(generation) ||
    generation < 0 ||
    typeof tokenRaw !== "string" ||
    tokenRaw.trim() === ""
  ) {
    throw new Error("invoke auth handoff: missing generation or token");
  }
  const token = Uint8Array.from(Buffer.from(tokenRaw, "base64url"));
  if (!token.length) {
    throw new Error("invoke auth handoff: empty token");
  }
  return { generation, token };
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
  startHostInvokeAuthRecvListener();
  return hostInvokeAuth;
}

/** Resets cached host auth (tests). */
export function resetHostInvokeAuthHandoffForTest(): void {
  hostInvokeAuth?.token.fill(0);
  hostInvokeAuth = undefined;
  hostInvokeAuthListenerStarted = false;
}

/** Ingests handoff lines into the host auth cache (tests). */
export async function consumeHostInvokeAuthStreamForTest(
  stream: Readable
): Promise<void> {
  await consumeHostInvokeAuthStream(stream);
}

/**
 * Sets connect-mode env for embedded invoke: skip spawn, boundary root, prefer UDS.
 * Starts the host auth recv listener when `FORST_INVOKE_AUTH_RECV_FD` is set.
 */
export function prepareConnectInvokeEnv(boundaryRoot?: string): string {
  startHostInvokeAuthRecvListener();
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
    process.env.FORST_BASE_URL = DEFAULT_EMBEDDED_INVOKE_BASE_URL;
  }
  return root;
}

// Consume an inherited recv fd as soon as this module loads in a host process.
startHostInvokeAuthRecvListener();

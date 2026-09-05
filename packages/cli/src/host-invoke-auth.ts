import { createReadStream } from "node:fs";
import type { Readable } from "node:stream";

import { DEFAULT_EMBEDDED_INVOKE_BASE_URL } from "./constants.js";
import type { InvokeAuthState } from "./invoke-auth.js";
import { type AuthHandoff } from "./invoke-auth-handoff.js";
import {
  readInvokeReadySocketPath,
  readInvokeReadyUrl,
} from "./invoke-ready.js";

const envInvokeAuthRecvFd = "FORST_INVOKE_AUTH_RECV_FD";

let hostInvokeAuth: InvokeAuthState | undefined;
let hostInvokeAuthListenerStarted = false;

/**
 * Parses one newline-delimited JSON handoff line into {@link AuthHandoff}.
 * Returns `undefined` for invalid JSON, non-object roots, or missing fields.
 */
function parseHandoffLine(line: string): AuthHandoff | undefined {
  let parsed: unknown;
  try {
    parsed = JSON.parse(line) as unknown;
  } catch {
    return undefined;
  }
  if (parsed === null || typeof parsed !== "object" || Array.isArray(parsed)) {
    return undefined;
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
    return undefined;
  }
  const token = Uint8Array.from(Buffer.from(tokenRaw, "base64url"));
  if (!token.length) {
    return undefined;
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
      const handoff = parseHandoffLine(line);
      if (handoff) {
        storeHostInvokeAuth(handoff);
      }
    }
  }
}

function startHostInvokeAuthRecvListener(): void {
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

/**
 * Returns invoke auth delivered over the host-mode pipe when
 * `FORST_INVOKE_AUTH_RECV_FD` is set by `forst dev`.
 *
 * Use this with a generated client's `resolveAuth` option when the Node host
 * process starts before the embedded invoke server is ready and auth arrives
 * through the inherited recv fd rather than `FORST_INVOKE_TOKEN` or a token file.
 *
 * The recv listener starts automatically when this module loads and when
 * {@link prepareInvokeConnect} runs. Safe to call before auth has arrived;
 * returns `undefined` until a valid handoff line is received.
 */
export function getInvokeAuthHandoff(): InvokeAuthState | undefined {
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
 * Configures connect-mode env for embedded invoke in a Node host process.
 *
 * Sets `FORST_SKIP_SPAWN=1` and `FORST_ROOT` (defaults to `process.cwd()`).
 * Resolves the invoke base URL from `.forst/invoke.ready` when present, clears URL
 * env when only a Unix socket is advertised, or falls back to
 * {@link DEFAULT_EMBEDDED_INVOKE_BASE_URL}.
 *
 * Also ensures the `FORST_INVOKE_AUTH_RECV_FD` listener is running so host-mode
 * auth handoff works without a separate setup call.
 *
 * @param boundaryRoot Project root containing `.forst/invoke.ready`.
 * @returns The resolved boundary root path.
 */
export function prepareInvokeConnect(boundaryRoot?: string): string {
  startHostInvokeAuthRecvListener();
  const root = boundaryRoot?.trim() || process.cwd();
  process.env.FORST_SKIP_SPAWN = "1";
  process.env.FORST_ROOT = root;

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

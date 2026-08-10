import * as http from "node:http";
import type { IncomingMessage } from "node:http";
import {
  normalizeHeaders,
  type InvokeHeadersInit,
} from "./invoke-headers.js";

/** True when the platform can dial AF_UNIX sockets. */
export function isUnixSocketSupported(): boolean {
  return process.platform !== "win32";
}

/** Minimal request init for Unix dial (avoids DOM `RequestInit` typings). */
export type UnixRequestInit = {
  method?: string;
  headers?: InvokeHeadersInit;
  body?: string | Uint8Array;
  signal?: AbortSignal;
};

/**
 * HTTP request over a Unix domain socket (Node `http.request` + `socketPath`).
 */
export function requestOverUnixSocket(
  socketPath: string,
  path: string,
  init: UnixRequestInit = {}
): Promise<Response> {
  return new Promise((resolve, reject) => {
    const headers = normalizeHeaders(init.headers);
    const body = init.body;
    const req = http.request(
      {
        socketPath,
        path,
        method: init.method ?? "GET",
        headers,
        signal: init.signal,
      },
      (res: IncomingMessage) => {
        const chunks: Buffer[] = [];
        res.on("data", (chunk) => {
          chunks.push(Buffer.isBuffer(chunk) ? chunk : Buffer.from(chunk));
        });
        res.on("end", () => {
          const responseBody = Buffer.concat(chunks);
          resolve(
            new Response(responseBody, {
              status: res.statusCode ?? 500,
              headers: headersToRecord(res.headers),
            })
          );
        });
      }
    );
    req.on("error", reject);
    if (init.signal?.aborted) {
      req.destroy();
      reject(new DOMException("Aborted", "AbortError"));
      return;
    }
    init.signal?.addEventListener(
      "abort",
      () => {
        req.destroy();
        reject(new DOMException("Aborted", "AbortError"));
      },
      { once: true }
    );
    if (typeof body === "string") {
      req.write(body);
    } else if (body instanceof Uint8Array) {
      req.write(body);
    }
    req.end();
  });
}

function headersToRecord(
  headers: IncomingMessage["headers"]
): Record<string, string> {
  const out: Record<string, string> = {};
  for (const [key, value] of Object.entries(headers)) {
    if (value === undefined) {
      continue;
    }
    out[key] = Array.isArray(value) ? value.join(", ") : value;
  }
  return out;
}

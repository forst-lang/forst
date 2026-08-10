import * as http from "node:http";
import type { IncomingMessage } from "node:http";

export function isUnixSocketSupported(): boolean {
  return process.platform !== "win32";
}

export function requestOverUnixSocket(
  socketPath: string,
  path: string,
  init: RequestInit = {}
): Promise<Response> {
  return new Promise((resolve, reject) => {
    const headers = normalizeHeaderRecord(init.headers);
    const body = init.body;
    const req = http.request(
      {
        socketPath,
        path,
        method: init.method ?? "GET",
        headers,
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
    if (typeof body === "string") {
      req.write(body);
    } else if (body instanceof Uint8Array) {
      req.write(body);
    }
    req.end();
  });
}

function normalizeHeaderRecord(
  headers: HeadersInit | undefined
): Record<string, string> {
  if (!headers) {
    return {};
  }
  if (headers instanceof Headers) {
    const out: Record<string, string> = {};
    headers.forEach((value, key) => {
      out[key] = value;
    });
    return out;
  }
  if (Array.isArray(headers)) {
    return Object.fromEntries(headers);
  }
  return { ...headers };
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

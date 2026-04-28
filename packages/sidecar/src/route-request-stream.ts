import { PassThrough, Readable, Transform } from "node:stream";
import type { Request } from "express";

/** Metadata for a routed request before the raw body is attached (see {@link ForstRoutedRequest}). */
export type ForstRoutedRequestMeta = {
  method: string;
  url: string;
  path: string;
  query: unknown;
  headers: Record<string, string | string[]>;
};

/** True when the body is already a string/buffer/object on `req` (body parser or rawBody). */
export function hasMaterializedBody(req: Request): boolean {
  const anyReq = req as any;
  return anyReq.rawBody !== undefined || anyReq.body !== undefined;
}

/**
 * Split `[{ ...meta, bodyBase64: "" }]` so the base64 payload can be inserted as a byte stream
 * between prefix and suffix (UTF-8 JSON).
 */
export function splitRawInvokeJsonArray(meta: ForstRoutedRequestMeta): {
  prefix: string;
  suffix: string;
} {
  const s = JSON.stringify([{ ...meta, bodyBase64: "" }]);
  const marker = '"bodyBase64":""';
  const i = s.indexOf(marker);
  if (i === -1) {
    throw new Error("routeToForst: expected bodyBase64 in JSON envelope");
  }
  const prefix = s.slice(0, i + '"bodyBase64":"'.length);
  // After the marker `"bodyBase64":""`, the remainder is `}]` — we still owe the closing `"` of the string value before `}`.
  const suffix = `"${s.slice(i + marker.length)}`;
  return { prefix, suffix };
}

function base64EncodeStream(): Transform {
  let carry = Buffer.alloc(0);
  return new Transform({
    transform(chunk: Buffer | string | Uint8Array, _enc, cb) {
      const b = Buffer.concat([carry, Buffer.from(chunk)]);
      const n = b.length - (b.length % 3);
      if (n > 0) {
        this.push(b.subarray(0, n).toString("base64"), "utf8");
        carry = b.subarray(n);
      } else {
        carry = b;
      }
      cb();
    },
    flush(cb) {
      if (carry.length > 0) {
        this.push(carry.toString("base64"), "utf8");
      }
      cb();
    },
  });
}

/**
 * Readable of the full `POST /invoke/raw` JSON body bytes: prefix + streamed base64(raw body) + suffix.
 * Pipes **`req`** (must still be readable). Do not use when {@link hasMaterializedBody} is true.
 */
export function createInvokeRawJsonBodyReadable(
  req: Request,
  meta: ForstRoutedRequestMeta
): Readable {
  const { prefix, suffix } = splitRawInvokeJsonArray(meta);
  const b64 = base64EncodeStream();
  const incoming = req as NodeJS.ReadableStream;
  const out = new PassThrough();

  b64.on("end", () => {
    out.write(Buffer.from(suffix, "utf8"));
    out.end();
  });
  b64.on("error", (err) => {
    out.destroy(err);
  });
  incoming.on("error", (err) => {
    out.destroy(err);
  });

  out.write(Buffer.from(prefix, "utf8"));
  b64.pipe(out, { end: false });
  incoming.pipe(b64);

  return out;
}

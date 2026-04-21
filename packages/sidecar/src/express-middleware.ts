import { Readable } from "node:stream";

import type { Request, RequestHandler, Response } from "express";

export type { ForstRoutedRequestMeta } from "./route-request-stream";

import type { ForstSidecar } from "./forst-sidecar";
import {
  createInvokeRawJsonBodyReadable,
  hasMaterializedBody,
  type ForstRoutedRequestMeta,
} from "./route-request-stream";

/**
 * Express.js middleware for easy integration
 */
export function createExpressMiddleware(sidecar: ForstSidecar): RequestHandler {
  return async (req, res, next) => {
    (req as Request & { forst: ForstSidecar }).forst = sidecar;
    next();
  };
}

export type ForstRoutedRequest = ForstRoutedRequestMeta & {
  /**
   * Raw request body bytes (base64). Always present (may be empty string).
   * This preserves binary payloads exactly across the JSON boundary.
   */
  bodyBase64: string;
};

export type ForstRoutedResponse = {
  status?: number;
  headers?: Record<string, string>;
  /** UTF-8 body. If `bodyBase64` is present, it wins. */
  body?: string;
  /** Raw bytes (base64). */
  bodyBase64?: string;
};

export type ForstRouteToForstOptions = {
  packageName: string;
  functionName: string;
  /**
   * When true (default), call `POST /invoke/raw` with a JSON array body — avoids the `/invoke`
   * JSON envelope and matches how Node forwards a single positional arg.
   */
  useRawInvoke?: boolean;
  /**
   * When true, stream NDJSON from `POST /invoke/raw?streaming=true` directly to the Express
   * response (passthrough). Your Forst handler must support streaming; the middleware does not
   * parse {@link ForstRoutedResponse} in this mode. Default **false** (single JSON response from Forst).
   */
  streaming?: boolean;
  /**
   * Optional hook to transform the request before invoking Forst.
   * Useful for injecting auth context, request ids, etc.
   */
  mapRequest?: (req: ForstRoutedRequest) => ForstRoutedRequest;
};

function toHeaderRecord(
  headers: Request["headers"]
): Record<string, string | string[]> {
  const out: Record<string, string | string[]> = {};
  for (const [k, v] of Object.entries(headers)) {
    if (v === undefined) continue;
    if (Array.isArray(v)) out[k] = v;
    else out[k] = String(v);
  }
  return out;
}

async function readRawBodyBase64(req: Request): Promise<string> {
  const anyReq = req as any;

  // Common conventions: body parser puts Buffer on req.body, some frameworks set req.rawBody.
  if (typeof anyReq.rawBody === "string") {
    return Buffer.from(anyReq.rawBody, "utf8").toString("base64");
  }
  if (anyReq.rawBody instanceof Buffer) {
    return anyReq.rawBody.toString("base64");
  }
  if (anyReq.body instanceof Buffer) {
    return anyReq.body.toString("base64");
  }
  if (typeof anyReq.body === "string") {
    return Buffer.from(anyReq.body, "utf8").toString("base64");
  }
  if (anyReq.body !== undefined) {
    // Best-effort: preserve whatever object body we got (not truly "raw", but still complete payload).
    return Buffer.from(JSON.stringify(anyReq.body), "utf8").toString("base64");
  }

  // Fall back to reading the stream (must be mounted before any body-parsing middleware).
  const chunks: Buffer[] = [];
  await new Promise<void>((resolve, reject) => {
    req.on("data", (c) => chunks.push(Buffer.isBuffer(c) ? c : Buffer.from(c)));
    req.on("end", () => resolve());
    req.on("error", (e) => reject(e));
  });
  return Buffer.concat(chunks).toString("base64");
}

/** JSON array body for `POST /invoke/raw`, always as a Web `ReadableStream` (streaming or one chunk). */
async function invokeRawBodyAsWebStream(
  req: Request,
  meta: ForstRoutedRequestMeta,
  mapRequest: ForstRouteToForstOptions["mapRequest"]
): Promise<ReadableStream<Uint8Array>> {
  if (mapRequest === undefined && !hasMaterializedBody(req)) {
    return Readable.toWeb(
      createInvokeRawJsonBodyReadable(req, meta)
    ) as ReadableStream<Uint8Array>;
  }
  const base: ForstRoutedRequest = {
    ...meta,
    bodyBase64: await readRawBodyBase64(req),
  };
  const routed = mapRequest ? mapRequest(base) : base;
  return Readable.toWeb(
    Readable.from([Buffer.from(JSON.stringify([routed]), "utf8")])
  ) as ReadableStream<Uint8Array>;
}

/**
 * Express middleware that forwards the **entire incoming HTTP request** (headers + raw body)
 * to a Forst function as a single argument.
 *
 * The invoked Forst function should return a JSON object matching {@link ForstRoutedResponse}.
 */
export function createRouteToForstMiddleware(
  sidecar: ForstSidecar,
  opts: ForstRouteToForstOptions
): RequestHandler {
  return async (req, res, next) => {
    try {
      const useRaw = opts.useRawInvoke !== false;
      const responseStreaming = opts.streaming === true;

      const meta: ForstRoutedRequestMeta = {
        method: req.method,
        url: req.originalUrl ?? req.url ?? "",
        path: req.path ?? "",
        query: req.query,
        headers: toHeaderRecord(req.headers),
      };

      const client = sidecar.getClient();

      if (responseStreaming) {
        const body = await invokeRawBodyAsWebStream(req, meta, opts.mapRequest);
        await client.pipeInvokeRawStream(
          opts.packageName,
          opts.functionName,
          body,
          res
        );
        return;
      }

      if (useRaw) {
        const body = await invokeRawBodyAsWebStream(req, meta, opts.mapRequest);
        const invoke = await client.invokeFunctionRawWithReadableBody<ForstRoutedResponse>(
          opts.packageName,
          opts.functionName,
          body
        );
        writeForstRoutedResponse(res, invoke.result ?? {});
        return;
      }

      const base: ForstRoutedRequest = {
        ...meta,
        bodyBase64: await readRawBodyBase64(req),
      };
      const routed = opts.mapRequest ? opts.mapRequest(base) : base;
      const args: unknown[] = [routed];

      const invoke = await client.invokeFunction<ForstRoutedResponse>(
        opts.packageName,
        opts.functionName,
        args
      );

      writeForstRoutedResponse(res, invoke.result ?? {});
    } catch (e) {
      next(e);
    }
  };
}

function writeForstRoutedResponse(
  res: Response,
  payload: ForstRoutedResponse
): void {
  const status = payload.status ?? 200;
  res.status(status);

  if (payload.headers) {
    for (const [k, v] of Object.entries(payload.headers)) {
      if (v !== undefined) res.setHeader(k, v);
    }
  }

  if (payload.bodyBase64) {
    res.send(Buffer.from(payload.bodyBase64, "base64"));
    return;
  }

  res.send(payload.body ?? "");
}

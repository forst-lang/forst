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

/**
 * Successful `invoke` `result` when the handler answers with an HTTP response (RFC §12 `kind: "answer"`).
 */
export type ForstRoutedAnswer = {
  kind: "answer";
  status?: number;
  headers?: Record<string, string>;
  /** UTF-8 body. If `bodyBase64` is present, it wins. */
  body?: string;
  /** Raw bytes (base64). */
  bodyBase64?: string;
};

/**
 * Successful `invoke` `result` when the handler passes control to the next Express middleware
 * (RFC §12 `kind: "pass"`). Optional JSON objects are shallow-merged.
 */
export type ForstRoutedPass<
  Locals extends Record<string, unknown> = Record<string, unknown>,
  ReqPatch extends object = Record<string, never>
> = {
  kind: "pass";
  locals?: Locals;
  /** Shallow-merged onto `req`. Maps from §12 JSON field `request`. */
  request?: ReqPatch;
};

/**
 * JSON your Forst gateway function returns: §12 `answer` vs `pass`.
 *
 * @typeParam Locals - Shape merged into `res.locals` when Forst returns `{ kind: "pass", locals }`.
 * @typeParam ReqPatch - Shape merged onto `req` when Forst returns `{ kind: "pass", request }`.
 */
export type ForstRoutedResponse<
  Locals extends Record<string, unknown> = Record<string, unknown>,
  ReqPatch extends object = Record<string, never>
> = ForstRoutedAnswer | ForstRoutedPass<Locals, ReqPatch>;

/** Narrowing helper for custom handlers that read invoke `result` payloads. */
export function isPass<
  Locals extends Record<string, unknown> = Record<string, unknown>,
  ReqPatch extends object = Record<string, never>
>(r: unknown): r is ForstRoutedPass<Locals, ReqPatch> {
  return (
    r !== null &&
    typeof r === "object" &&
    (r as { kind?: unknown }).kind === "pass"
  );
}

/** Narrowing helper for the `answer` arm. */
export function isAnswer(r: unknown): r is ForstRoutedAnswer {
  return (
    r !== null &&
    typeof r === "object" &&
    (r as { kind?: unknown }).kind === "answer"
  );
}

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
 *
 * @typeParam Locals - Shape merged into `res.locals` when Forst returns `{ kind: "pass", locals }`.
 * @typeParam ReqPatch - Shape merged onto `req` when Forst returns `{ kind: "pass", request }`.
 */
export function createRouteToForstMiddleware<
  Locals extends Record<string, unknown> = Record<string, unknown>,
  ReqPatch extends object = Record<string, never>
>(
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
        const invoke = await client.invokeFunctionRawWithReadableBody<
          ForstRoutedResponse<Locals, ReqPatch>
        >(opts.packageName, opts.functionName, body);
        applyForstRoutedResult(req, res, next, invoke.result ?? {});
        return;
      }

      const base: ForstRoutedRequest = {
        ...meta,
        bodyBase64: await readRawBodyBase64(req),
      };
      const routed = opts.mapRequest ? opts.mapRequest(base) : base;
      const args: unknown[] = [routed];

      const invoke = await client.invokeFunction<ForstRoutedResponse<Locals, ReqPatch>>(
        opts.packageName,
        opts.functionName,
        args
      );

      applyForstRoutedResult(req, res, next, invoke.result ?? {});
    } catch (e) {
      next(e);
    }
  };
}

function applyForstRoutedResult(
  req: Request,
  res: Response,
  next: Parameters<RequestHandler>[2],
  result: ForstRoutedResponse<Record<string, unknown>, object>
): void {
  if (isPass(result)) {
    if (result.locals !== undefined) {
      Object.assign(res.locals, result.locals);
    }
    if (result.request !== undefined) {
      Object.assign(req as object, result.request as object);
    }
    next();
    return;
  }
  if (isAnswer(result)) {
    writeForstRoutedAnswer(res, result);
    return;
  }
  next(
    new Error(
      'Invalid Forst gateway invoke result: expected kind "answer" or "pass"'
    )
  );
}

function writeForstRoutedAnswer(res: Response, payload: ForstRoutedAnswer): void {
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

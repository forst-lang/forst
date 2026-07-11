import type { Writable } from "node:stream";
import { Effect } from "effect";
import {
  defaultNodeRuntimeSetup,
  type ForstNodeRuntime,
} from "../effect/runtime.js";
import {
  JsonRpcError,
  PARSE_ERROR,
} from "./errors.js";
import {
  DEFAULT_MAX_MESSAGE_BYTES,
  newErrorResponseFrame,
  newOkResponseFrame,
  parseRequestFrame,
  ProtoFrameReader,
  writeProtoFrame,
  type Frame,
} from "./frame.js";
import type { JsonRpcId, JsonRpcRequest, JsonRpcResponse } from "./protocol.js";

export { DEFAULT_MAX_MESSAGE_BYTES } from "./frame.js";

export interface ProtoLoopOptions {
  maxMessageBytes?: number;
  onRequest: (
    request: JsonRpcRequest
  ) => Effect.Effect<JsonRpcResponse | null, never, never>;
  onParseError?: (err: unknown, frame: Frame | null) => JsonRpcResponse;
  /** Runtime for async dispatch; must match the layer provided at the process boundary. */
  runtime?: ForstNodeRuntime;
}

export function frameToJsonRpcRequest(frame: Frame): JsonRpcRequest {
  const parsed = parseRequestFrame(frame);
  return {
    jsonrpc: "2.0",
    id: parsed.id,
    method: parsed.method,
    params: parsed.params,
  };
}

export function writeJsonRpcResponse(
  stdout: Writable,
  response: JsonRpcResponse,
  maxLen = DEFAULT_MAX_MESSAGE_BYTES
): void {
  const id = typeof response.id === "number" ? response.id : 0;
  if ("error" in response && response.error !== undefined) {
    const err = new JsonRpcError(
      response.error.code,
      response.error.message,
      response.error.data
    );
    writeProtoFrame(stdout, newErrorResponseFrame(id, err), maxLen);
    return;
  }
  if ("result" in response) {
    writeProtoFrame(stdout, newOkResponseFrame(id, response.result), maxLen);
  }
}

export function errorResponse(
  id: JsonRpcId,
  err: JsonRpcError
): JsonRpcResponse {
  return {
    jsonrpc: "2.0",
    id,
    error: err.toObject(),
  };
}

export function successResponse(
  id: JsonRpcId,
  result: unknown
): JsonRpcResponse {
  return {
    jsonrpc: "2.0",
    id,
    result,
  };
}

/** Runs the length-prefixed proto RPC loop until stdin closes. */
export async function runProtoLoop(
  stdin: NodeJS.ReadableStream,
  stdout: Writable,
  options: ProtoLoopOptions
): Promise<void> {
  const maxBytes = options.maxMessageBytes ?? DEFAULT_MAX_MESSAGE_BYTES;
  const reader = new ProtoFrameReader();
  const runtime = options.runtime ?? defaultNodeRuntimeSetup.runtime;

  if ("resume" in stdin && typeof stdin.resume === "function") {
    stdin.resume();
  }

  for await (const chunk of stdin) {
    reader.append(Buffer.from(chunk as Buffer));

    for (;;) {
      let frame: Frame | null;
      try {
        frame = reader.tryReadFrame(maxBytes);
      } catch (err) {
        const response =
          options.onParseError?.(err, null) ??
          errorResponse(
            null,
            err instanceof JsonRpcError
              ? err
              : new JsonRpcError(PARSE_ERROR, "parse error")
          );
        writeJsonRpcResponse(stdout, response, maxBytes);
        continue;
      }

      if (frame === null) {
        break;
      }

      let request: JsonRpcRequest;
      try {
        request = frameToJsonRpcRequest(frame);
      } catch (err) {
        const response =
          options.onParseError?.(err, frame) ??
          errorResponse(
            frame.id,
            err instanceof JsonRpcError
              ? err
              : new JsonRpcError(PARSE_ERROR, "parse error")
          );
        writeJsonRpcResponse(stdout, response, maxBytes);
        continue;
      }

      const response = await runtime.runPromise(options.onRequest(request));
      if (response !== null) {
        writeJsonRpcResponse(stdout, response, maxBytes);
      }
    }
  }
}

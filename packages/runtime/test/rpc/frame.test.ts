import { describe, expect, test } from "bun:test";
import { PassThrough } from "node:stream";
import * as Errors from "../../src/rpc/errors.js";
import {
  decodeFrame,
  encodeFrame,
  newErrorResponseFrame,
  newOkResponseFrame,
  newRequestFrame,
  parseRequestFrame,
  ProtoFrameReader,
  writeProtoFrame,
} from "../../src/rpc/frame.js";
import { METHOD_PING } from "../../src/rpc/protocol.js";

describe("proto frame codec", () => {
  test("request and ok response round-trip", () => {
    const req = newRequestFrame(42, METHOD_PING, {});
    const reqBytes = encodeFrame(req);
    const decodedReq = decodeFrame(reqBytes);
    expect(decodedReq.id).toBe(42);
    expect(decodedReq.request?.method).toBe(METHOD_PING);
    expect(JSON.parse(new TextDecoder().decode(decodedReq.request!.payloadJson))).toEqual(
      {}
    );

    const resp = newOkResponseFrame(42, { pong: true });
    const respBytes = encodeFrame(resp);
    const decodedResp = decodeFrame(respBytes);
    expect(decodedResp.id).toBe(42);
    expect(JSON.parse(new TextDecoder().decode(decodedResp.response!.okJson!))).toEqual(
      { pong: true }
    );
  });

  test("error response preserves negative code", () => {
    const err = Errors.methodNotFound();
    const frame = newErrorResponseFrame(7, err);
    const decoded = decodeFrame(encodeFrame(frame));
    expect(decoded.response?.err?.code).toBe(Errors.METHOD_NOT_FOUND);
    expect(decoded.response?.err?.message).toBe("Method not found");
  });

  test("parseRequestFrame rejects missing request", () => {
    expect(() => parseRequestFrame({ id: 1 })).toThrow(
      expect.objectContaining({
        code: Errors.INVALID_REQUEST,
        message: "frame missing request",
      })
    );
  });

  test("parseRequestFrame rejects empty method", () => {
    expect(() =>
      parseRequestFrame({
        id: 1,
        request: { method: "", payloadJson: new Uint8Array(0) },
      })
    ).toThrow(
      expect.objectContaining({
        code: Errors.INVALID_REQUEST,
        message: "method must be a string",
      })
    );
  });

  test("parseRequestFrame rejects invalid payload JSON", () => {
    expect(() =>
      parseRequestFrame({
        id: 1,
        request: {
          method: METHOD_PING,
          payloadJson: new TextEncoder().encode("{"),
        },
      })
    ).toThrow(
      expect.objectContaining({
        code: Errors.PARSE_ERROR,
        message: "invalid JSON in request payload",
      })
    );
  });

  test("length prefix read/write", () => {
    const stdout = new PassThrough();
    const chunks: Buffer[] = [];
    stdout.on("data", (chunk) => chunks.push(Buffer.from(chunk)));

    writeProtoFrame(stdout, newRequestFrame(1, METHOD_PING, {}));
    stdout.end();

    const reader = new ProtoFrameReader();
    reader.append(Buffer.concat(chunks));
    const frame = reader.tryReadFrame();
    expect(frame?.id).toBe(1);
    expect(frame?.request?.method).toBe(METHOD_PING);
  });
});

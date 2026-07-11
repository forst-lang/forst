import { describe, expect, test } from "bun:test";
import { PassThrough } from "node:stream";
import { JsonRpcError, METHOD_NOT_FOUND } from "../../src/rpc/errors.js";
import {
  decodeFrame,
  encodeFrame,
  newErrorResponseFrame,
  newOkResponseFrame,
  newRequestFrame,
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
    const err = new JsonRpcError(METHOD_NOT_FOUND, "Method not found");
    const frame = newErrorResponseFrame(7, err);
    const decoded = decodeFrame(encodeFrame(frame));
    expect(decoded.response?.err?.code).toBe(METHOD_NOT_FOUND);
    expect(decoded.response?.err?.message).toBe("Method not found");
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

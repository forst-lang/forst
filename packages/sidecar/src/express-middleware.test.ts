import { Readable } from "node:stream";

import { describe, expect, it, jest } from "bun:test";
import { createRouteToForstMiddleware } from "./express-middleware";

describe("createRouteToForstMiddleware", () => {
  it("serializes method/url/headers and forwards raw body as base64", async () => {
    const invokeFunctionRawWithReadableBody = jest.fn().mockResolvedValue({
      success: true,
      result: { status: 204 },
    });
    const sidecar = {
      getClient: () => ({ invokeFunctionRawWithReadableBody }),
    } as any;

    const mw = createRouteToForstMiddleware(sidecar, {
      packageName: "main",
      functionName: "Handle",
    });

    const req = {
      method: "POST",
      originalUrl: "/api/echo?x=1",
      url: "/api/echo?x=1",
      path: "/api/echo",
      query: { x: "1" },
      headers: { "content-type": "application/octet-stream", "x-req": "abc" },
      body: Buffer.from("hello"),
      on: () => {},
    } as any;

    const resHeaders: Record<string, string> = {};
    const res = {
      status: jest.fn().mockReturnThis(),
      setHeader: (k: string, v: string) => {
        resHeaders[k.toLowerCase()] = v;
      },
      send: jest.fn(),
    } as any;

    const next = jest.fn();
    await mw(req, res, next);

    expect(next).not.toHaveBeenCalled();
    expect(invokeFunctionRawWithReadableBody).toHaveBeenCalledWith(
      "main",
      "Handle",
      expect.any(Object)
    );
    expect(res.status).toHaveBeenCalledWith(204);
    expect(res.send).toHaveBeenCalledWith("");
  });

  it("applies status, headers, and binary body from Forst response", async () => {
    const invokeFunctionRawWithReadableBody = jest.fn().mockResolvedValue({
      success: true,
      result: {
        status: 200,
        headers: { "content-type": "application/octet-stream", "x-out": "1" },
        bodyBase64: Buffer.from("bin").toString("base64"),
      },
    });
    const sidecar = {
      getClient: () => ({ invokeFunctionRawWithReadableBody }),
    } as any;

    const mw = createRouteToForstMiddleware(sidecar, {
      packageName: "main",
      functionName: "Handle",
    });

    const req = {
      method: "GET",
      originalUrl: "/x",
      url: "/x",
      path: "/x",
      query: {},
      headers: {},
      body: Buffer.alloc(0),
      on: () => {},
    } as any;

    const resHeaders: Record<string, string> = {};
    const res = {
      status: jest.fn().mockReturnThis(),
      setHeader: (k: string, v: string) => {
        resHeaders[k.toLowerCase()] = v;
      },
      send: jest.fn(),
    } as any;

    const next = jest.fn();
    await mw(req, res, next);

    expect(next).not.toHaveBeenCalled();
    expect(res.status).toHaveBeenCalledWith(200);
    expect(resHeaders["content-type"]).toBe("application/octet-stream");
    expect(resHeaders["x-out"]).toBe("1");
    expect(res.send).toHaveBeenCalledWith(Buffer.from("bin"));
  });

  it("useRawInvoke: false uses POST /invoke envelope via invokeFunction", async () => {
    const invokeFunction = jest.fn().mockResolvedValue({
      success: true,
      result: { status: 200, body: "ok" },
    });
    const invokeFunctionRawWithReadableBody = jest.fn();
    const sidecar = {
      getClient: () => ({ invokeFunction, invokeFunctionRawWithReadableBody }),
    } as any;

    const mw = createRouteToForstMiddleware(sidecar, {
      packageName: "main",
      functionName: "Handle",
      useRawInvoke: false,
    });

    const req = {
      method: "GET",
      originalUrl: "/x",
      url: "/x",
      path: "/x",
      query: {},
      headers: {},
      body: Buffer.alloc(0),
      on: () => {},
    } as any;

    const res = {
      status: jest.fn().mockReturnThis(),
      setHeader: jest.fn(),
      send: jest.fn(),
    } as any;

    await mw(req, res, jest.fn());

    expect(invokeFunctionRawWithReadableBody).not.toHaveBeenCalled();
    expect(invokeFunction).toHaveBeenCalled();
    expect(res.send).toHaveBeenCalledWith("ok");
  });

  it("streams outbound invoke body when request is not materialized (default)", async () => {
    const invokeFunctionRawWithReadableBody = jest.fn().mockResolvedValue({
      success: true,
      result: { status: 200, body: "ok" },
    });
    const stream = Readable.from([Buffer.from("hello")]) as any;
    stream.method = "POST";
    stream.originalUrl = "/x";
    stream.url = "/x";
    stream.path = "/x";
    stream.query = {};
    stream.headers = {};

    const sidecar = {
      getClient: () => ({ invokeFunctionRawWithReadableBody }),
    } as any;

    const mw = createRouteToForstMiddleware(sidecar, {
      packageName: "main",
      functionName: "Handle",
    });

    const res = {
      status: jest.fn().mockReturnThis(),
      setHeader: jest.fn(),
      send: jest.fn(),
    } as any;

    await mw(stream, res, jest.fn());

    expect(invokeFunctionRawWithReadableBody).toHaveBeenCalledWith(
      "main",
      "Handle",
      expect.any(Object)
    );
    expect(res.send).toHaveBeenCalledWith("ok");
  });

  it("streaming: true pipes via pipeInvokeRawStream (invoke/raw + streaming)", async () => {
    const pipeInvokeRawStream = jest.fn().mockResolvedValue(undefined);
    const invokeFunctionRawWithReadableBody = jest.fn();
    const sidecar = {
      getClient: () => ({ pipeInvokeRawStream, invokeFunctionRawWithReadableBody }),
    } as any;

    const mw = createRouteToForstMiddleware(sidecar, {
      packageName: "main",
      functionName: "StreamHandle",
      streaming: true,
    });

    const req = {
      method: "POST",
      originalUrl: "/x",
      url: "/x",
      path: "/x",
      query: {},
      headers: {},
      body: Buffer.alloc(0),
      on: () => {},
    } as any;

    const res = { status: jest.fn(), setHeader: jest.fn(), end: jest.fn() } as any;

    await mw(req, res, jest.fn());

    expect(invokeFunctionRawWithReadableBody).not.toHaveBeenCalled();
    expect(pipeInvokeRawStream).toHaveBeenCalledWith(
      "main",
      "StreamHandle",
      expect.any(Object),
      res
    );
  });
});

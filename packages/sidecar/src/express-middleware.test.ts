import { Readable } from "node:stream";
import { text } from "node:stream/consumers";

import { describe, expect, it, jest } from "bun:test";
import { createRouteToForstMiddleware } from "./express-middleware";

describe("createRouteToForstMiddleware", () => {
  it("serializes method/url/headers and forwards raw body as base64", async () => {
    const invokeFunctionRawWithReadableBody = jest.fn().mockResolvedValue({
      success: true,
      result: { kind: "answer", status: 204 },
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
      on: () => { },
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
        kind: "answer",
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
      on: () => { },
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
      result: { kind: "answer", status: 200, body: "ok" },
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
      on: () => { },
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
      result: { kind: "answer", status: 200, body: "ok" },
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

  it("mapRequest can inject headers; Cookie and Authorization pass through from req", async () => {
    let capturedBody: ReadableStream<Uint8Array> | null = null;
    const invokeFunctionRawWithReadableBody = jest
      .fn()
      .mockImplementation(
        async (_pkg: string, _fn: string, body: ReadableStream<Uint8Array>) => {
          capturedBody = body;
          return {
            success: true,
            result: { status: 200, body: "ok" },
          };
        }
      );
    const sidecar = {
      getClient: () => ({ invokeFunctionRawWithReadableBody }),
    } as any;

    const mw = createRouteToForstMiddleware(sidecar, {
      packageName: "main",
      functionName: "Handle",
      mapRequest: (r) => ({
        ...r,
        headers: {
          ...r.headers,
          "x-context": "injected",
        },
      }),
    });

    const req = {
      method: "GET",
      originalUrl: "/x",
      url: "/x",
      path: "/x",
      query: {},
      headers: {
        cookie: "sid=1",
        authorization: "Bearer z",
        host: "localhost",
      },
      body: Buffer.alloc(0),
      on: () => { },
    } as any;

    const res = {
      status: jest.fn().mockReturnThis(),
      setHeader: jest.fn(),
      send: jest.fn(),
    } as any;

    await mw(req, res, jest.fn());

    expect(invokeFunctionRawWithReadableBody).toHaveBeenCalled();
    expect(capturedBody).not.toBeNull();
    const raw = await text(Readable.fromWeb(capturedBody!) as any);
    const parsed = JSON.parse(raw) as Array<{
      headers: Record<string, string | string[]>;
    }>;
    expect(parsed[0].headers["x-context"]).toBe("injected");
    expect(parsed[0].headers["cookie"]).toBe("sid=1");
    expect(parsed[0].headers["authorization"]).toBe("Bearer z");
  });

  it("calls next with the error when invokeFunctionRawWithReadableBody rejects", async () => {
    const err = new Error("invoke failed");
    const invokeFunctionRawWithReadableBody = jest.fn().mockRejectedValue(err);
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
      on: () => { },
    } as any;

    const res = {
      status: jest.fn().mockReturnThis(),
      setHeader: jest.fn(),
      send: jest.fn(),
    } as any;

    const next = jest.fn();
    await mw(req, res, next);

    expect(next).toHaveBeenCalledWith(err);
    expect(res.status).not.toHaveBeenCalled();
    expect(res.send).not.toHaveBeenCalled();
  });

  it("kind pass merges locals and request then calls next without sending", async () => {
    const invokeFunctionRawWithReadableBody = jest.fn().mockResolvedValue({
      success: true,
      result: {
        kind: "pass",
        locals: { sessionId: "s1" },
        request: { forstCtx: { n: 1 } },
      },
    });
    const sidecar = {
      getClient: () => ({ invokeFunctionRawWithReadableBody }),
    } as any;

    const mw = createRouteToForstMiddleware<
      { sessionId: string },
      { forstCtx: { n: number } }
    >(sidecar, {
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
      on: () => { },
    } as any;

    const res = {
      locals: {} as Record<string, unknown>,
      status: jest.fn().mockReturnThis(),
      setHeader: jest.fn(),
      send: jest.fn(),
    } as any;

    const next = jest.fn();
    await mw(req, res, next);

    expect(next).toHaveBeenCalledTimes(1);
    expect(next.mock.calls[0] ?? []).toEqual([]);
    expect(res.locals.sessionId).toBe("s1");
    expect(req.forstCtx).toEqual({ n: 1 });
    expect(res.status).not.toHaveBeenCalled();
    expect(res.send).not.toHaveBeenCalled();
  });

  it("calls next with error when invoke result has unknown kind", async () => {
    const invokeFunctionRawWithReadableBody = jest.fn().mockResolvedValue({
      success: true,
      result: { kind: "other" },
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
      on: () => { },
    } as any;

    const res = {
      status: jest.fn().mockReturnThis(),
      send: jest.fn(),
    } as any;

    const next = jest.fn();
    await mw(req, res, next);

    expect(next).toHaveBeenCalledTimes(1);
    const errArg = next.mock.calls[0]?.[0];
    expect(errArg).toBeInstanceOf(Error);
    expect(String((errArg as Error).message)).toContain("answer");
    expect(res.send).not.toHaveBeenCalled();
  });

  it("kind pass with no locals or request still calls next", async () => {
    const invokeFunctionRawWithReadableBody = jest.fn().mockResolvedValue({
      success: true,
      result: { kind: "pass" },
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
      on: () => { },
    } as any;

    const res = {
      locals: {},
      status: jest.fn().mockReturnThis(),
      send: jest.fn(),
    } as any;

    const next = jest.fn();
    await mw(req, res, next);

    expect(next).toHaveBeenCalledTimes(1);
    expect(res.send).not.toHaveBeenCalled();
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
      on: () => { },
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

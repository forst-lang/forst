/**
 * Uses Express 4 as devDependency: Express 5’s router + Bun hit a path-to-regexp
 * resolution bug; production apps still target peer express ^5.
 */
import { createServer, type IncomingMessage, type Server } from "node:http";
import { text } from "node:stream/consumers";

import { afterEach, describe, expect, it } from "bun:test";
import express from "express";

import { ForstSidecar } from "./forst-sidecar";
import {
  createRouteToForstMiddleware,
  type ForstRoutedRequest,
} from "./express-middleware";

const PKG = "gw";
const FN = "Handle";
const STREAM_FN = "StreamHandle";
const FN_CONTINUE = "ContinueHandle";

function readStubBody(req: IncomingMessage): Promise<Buffer> {
  return new Promise((resolve, reject) => {
    const chunks: Buffer[] = [];
    req.on("data", (c: Buffer | string) =>
      chunks.push(Buffer.isBuffer(c) ? c : Buffer.from(c))
    );
    req.on("end", () => resolve(Buffer.concat(chunks)));
    req.on("error", reject);
  });
}

function headerVal(
  headers: Record<string, string | string[]>,
  name: string
): string {
  const direct = headers[name];
  if (direct !== undefined) {
    return Array.isArray(direct) ? String(direct[0]) : String(direct);
  }
  const lower = name.toLowerCase();
  for (const [k, v] of Object.entries(headers)) {
    if (k.toLowerCase() === lower) {
      return Array.isArray(v) ? String(v[0]) : String(v);
    }
  }
  return "";
}

async function listenOnRandomPort(server: Server): Promise<number> {
  return new Promise((resolve, reject) => {
    server.once("error", reject);
    server.listen(0, "127.0.0.1", () => {
      const a = server.address();
      if (a && typeof a === "object") {
        resolve(a.port);
      } else {
        reject(new Error("no port"));
      }
    });
  });
}

function assertRoutedMatchesExpressSessionRequest(
  routed: ForstRoutedRequest,
  expected: Readonly<{
    method: string;
    url: string;
    path: string;
    query: Record<string, unknown>;
    bodyBytes: Buffer;
    cookie: string;
    authorization: string;
    requestId: string;
  }>
): void {
  expect(routed.method).toBe(expected.method);
  expect(routed.url).toBe(expected.url);
  expect(routed.path).toBe(expected.path);
  expect(routed.query).toEqual(expected.query);
  expect([...Buffer.from(routed.bodyBase64, "base64")]).toEqual([
    ...expected.bodyBytes,
  ]);

  expect(headerVal(routed.headers, "cookie")).toBe(expected.cookie);
  expect(headerVal(routed.headers, "authorization")).toBe(expected.authorization);
  expect(headerVal(routed.headers, "x-request-id")).toBe(expected.requestId);
}

async function withStubDevServer(
  handler: (req: IncomingMessage, res: import("node:http").ServerResponse) => void
): Promise<{ baseUrl: string; server: Server; close: () => Promise<void> }> {
  const server = createServer(handler);
  const port = await listenOnRandomPort(server);
  const baseUrl = `http://127.0.0.1:${port}`;
  return {
    baseUrl,
    server,
    close: () =>
      new Promise((resolve, reject) =>
        server.close((err) => (err ? reject(err) : resolve()))
      ),
  };
}

describe("createRouteToForstMiddleware (integration)", () => {
  let sidecar: ForstSidecar | null = null;
  let stubClose: (() => Promise<void>) | null = null;
  let expressServer: Server | null = null;

  afterEach(async () => {
    if (sidecar) {
      await sidecar.stop();
      sidecar = null;
    }
    if (stubClose) {
      await stubClose();
      stubClose = null;
    }
    if (expressServer) {
      await new Promise<void>((resolve, reject) =>
        expressServer!.close((err) => (err ? reject(err) : resolve()))
      );
      expressServer = null;
    }
  });

  it("forwards method, url, query, session/auth headers, and raw body to invoke/raw (session-style)", async () => {
    const clientBody = Buffer.from(
      JSON.stringify({ action: "ping", token: "t1" }),
      "utf8"
    );
    const cookie = "sessionid=abc; Path=/";
    const authorization = "Bearer test-token";
    const requestId = "rid-session-7";

    let stub = await withStubDevServer(async (req, res) => {
      if (req.method === "GET" && req.url?.startsWith("/health")) {
        res.writeHead(200, { "Content-Type": "application/json" });
        res.end(JSON.stringify({ success: true }));
        return;
      }
      if (req.method === "POST" && req.url?.startsWith("/invoke/raw")) {
        const u = new URL(req.url ?? "", "http://127.0.0.1");
        expect(u.searchParams.get("package")).toBe(PKG);
        expect(u.searchParams.get("function")).toBe(FN);
        expect(u.searchParams.get("streaming")).toBeNull();

        const raw = await readStubBody(req);
        const parsed = JSON.parse(raw.toString("utf8")) as unknown[];
        expect(Array.isArray(parsed)).toBe(true);
        expect(parsed.length).toBe(1);
        const routed = parsed[0] as ForstRoutedRequest;

        assertRoutedMatchesExpressSessionRequest(routed, {
          method: "POST",
          url: "/session/api?foo=bar&baz=2",
          path: "/session/api",
          query: { foo: "bar", baz: "2" },
          bodyBytes: clientBody,
          cookie,
          authorization,
          requestId,
        });

        res.writeHead(200, { "Content-Type": "application/json" });
        res.end(
          JSON.stringify({
            success: true,
            result: {
              kind: "answer",
              status: 201,
              headers: { "x-from-forst": "1", "content-type": "text/plain" },
              body: "session-ok",
            },
          })
        );
        return;
      }
      res.writeHead(404);
      res.end();
    });
    stubClose = stub.close;

    sidecar = new ForstSidecar({
      sidecarRuntime: "connect",
      devServerUrl: stub.baseUrl,
      versionCheck: "off",
    });
    await sidecar.start();

    const app = express();
    app.use(express.raw({ type: "*/*" }));
    const mw = createRouteToForstMiddleware(sidecar, {
      packageName: PKG,
      functionName: FN,
    });
    app.use((req, res, next) => {
      if (!req.path.startsWith("/session")) {
        res.status(404).end();
        return;
      }
      void mw(req, res, next);
    });

    expressServer = createServer(app);
    const expressPort = await listenOnRandomPort(expressServer);

    const url = `http://127.0.0.1:${expressPort}/session/api?foo=bar&baz=2`;
    const upstream = await fetch(url, {
      method: "POST",
      headers: {
        Cookie: cookie,
        Authorization: authorization,
        "X-Request-Id": requestId,
        "Content-Type": "application/json",
      },
      body: clientBody,
    });

    expect(upstream.status).toBe(201);
    expect(upstream.headers.get("x-from-forst")).toBe("1");
    expect(await upstream.text()).toBe("session-ok");
  });

  it("streaming: true still sends full routed metadata to invoke/raw and pipes dev response", async () => {
    const clientBody = Buffer.from("raw-bytes", "utf8");
    const cookie = "sid=stream";
    const authorization = "Bearer s";

    let stub = await withStubDevServer(async (req, res) => {
      if (req.method === "GET" && req.url?.startsWith("/health")) {
        res.writeHead(200, { "Content-Type": "application/json" });
        res.end(JSON.stringify({ success: true }));
        return;
      }
      if (req.method === "POST" && req.url?.startsWith("/invoke/raw")) {
        const u = new URL(req.url ?? "", "http://127.0.0.1");
        expect(u.searchParams.get("package")).toBe(PKG);
        expect(u.searchParams.get("function")).toBe(STREAM_FN);
        expect(u.searchParams.get("streaming")).toBe("true");

        const raw = await readStubBody(req);
        const parsed = JSON.parse(raw.toString("utf8")) as unknown[];
        const routed = parsed[0] as ForstRoutedRequest;

        assertRoutedMatchesExpressSessionRequest(routed, {
          method: "POST",
          url: "/session/stream?q=9",
          path: "/session/stream",
          query: { q: "9" },
          bodyBytes: clientBody,
          cookie,
          authorization,
          requestId: "req-stream",
        });

        res.writeHead(200, {
          "Content-Type": "application/octet-stream",
          "X-Stream-Stub": "yes",
        });
        res.end("ndjson-chunk\n");
        return;
      }
      res.writeHead(404);
      res.end();
    });
    stubClose = stub.close;

    sidecar = new ForstSidecar({
      sidecarRuntime: "connect",
      devServerUrl: stub.baseUrl,
      versionCheck: "off",
    });
    await sidecar.start();

    const app = express();
    app.use(express.raw({ type: "*/*" }));
    const mwStream = createRouteToForstMiddleware(sidecar, {
      packageName: PKG,
      functionName: STREAM_FN,
      streaming: true,
    });
    app.use((req, res, next) => {
      if (!req.path.startsWith("/session")) {
        res.status(404).end();
        return;
      }
      void mwStream(req, res, next);
    });

    expressServer = createServer(app);
    const expressPort = await listenOnRandomPort(expressServer);

    const upstream = await fetch(
      `http://127.0.0.1:${expressPort}/session/stream?q=9`,
      {
        method: "POST",
        headers: {
          Cookie: cookie,
          Authorization: authorization,
          "X-Request-Id": "req-stream",
        },
        body: clientBody,
        duplex: "half",
      }
    );

    expect(upstream.status).toBe(200);
    expect(upstream.headers.get("x-stream-stub")).toBe("yes");
    expect(await text(upstream.body!)).toBe("ndjson-chunk\n");
  });

  it("kind pass merges locals; next middleware sees res.locals (typed flow)", async () => {
    let stub = await withStubDevServer(async (req, res) => {
      if (req.method === "GET" && req.url?.startsWith("/health")) {
        res.writeHead(200, { "Content-Type": "application/json" });
        res.end(JSON.stringify({ success: true }));
        return;
      }
      if (req.method === "POST" && req.url?.startsWith("/invoke/raw")) {
        const u = new URL(req.url ?? "", "http://127.0.0.1");
        expect(u.searchParams.get("package")).toBe(PKG);
        expect(u.searchParams.get("function")).toBe(FN_CONTINUE);

        res.writeHead(200, { "Content-Type": "application/json" });
        res.end(
          JSON.stringify({
            success: true,
            result: {
              kind: "pass",
              locals: { sessionId: "integration-sid" },
            },
          })
        );
        return;
      }
      res.writeHead(404);
      res.end();
    });
    stubClose = stub.close;

    sidecar = new ForstSidecar({
      sidecarRuntime: "connect",
      devServerUrl: stub.baseUrl,
      versionCheck: "off",
    });
    await sidecar.start();

    const app = express();
    app.use(express.raw({ type: "*/*" }));
    const routeMw = createRouteToForstMiddleware<{ sessionId: string }>(
      sidecar,
      {
        packageName: PKG,
        functionName: FN_CONTINUE,
      }
    );
    app.use((req, res, next) => {
      if (!req.path.startsWith("/session")) {
        res.status(404).end();
        return;
      }
      void routeMw(req, res, next);
    });
    app.use((req, res, next) => {
      if (!req.path.startsWith("/session")) {
        next();
        return;
      }
      res
        .status(200)
        .type("application/json")
        .send(JSON.stringify({ sid: res.locals.sessionId }));
    });

    expressServer = createServer(app);
    const expressPort = await listenOnRandomPort(expressServer);

    const upstream = await fetch(
      `http://127.0.0.1:${expressPort}/session/pass`,
      {
        method: "POST",
        headers: { "Content-Type": "application/octet-stream" },
        body: Buffer.from("x"),
      }
    );

    expect(upstream.status).toBe(200);
    expect(await upstream.json()).toEqual({ sid: "integration-sid" });
  });
});

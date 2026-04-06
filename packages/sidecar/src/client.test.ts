import { afterEach, describe, expect, it, jest } from "bun:test";
import { ForstSidecarClient } from "./client";
import {
  DevServerHttpFailure,
  DevServerInvokeRejected,
} from "./errors";
import type { FunctionInfo } from "./types";

function sampleFunctionInfo(): FunctionInfo {
  return {
    package: "demo",
    name: "Echo",
    supportsStreaming: false,
    inputType: "string",
    outputType: "string",
    parameters: [],
    returnType: "string",
    filePath: "/tmp/demo.ft",
  };
}

describe("ForstSidecarClient", () => {
  const originalFetch = global.fetch;

  afterEach(() => {
    global.fetch = originalFetch;
  });

  it("GET /functions returns parsed FunctionInfo list", async () => {
    const fn = sampleFunctionInfo();
    global.fetch = jest.fn().mockResolvedValue({
      ok: true,
      status: 200,
      headers: new Headers({ "content-type": "application/json" }),
      json: async () => ({
        success: true,
        result: [fn],
      }),
    }) as unknown as typeof fetch;

    const client = new ForstSidecarClient({
      baseUrl: "http://127.0.0.1:8080",
      retries: 0,
    });
    const out = await client.discoverFunctions();

    expect(out).toHaveLength(1);
    expect(out[0].package).toBe("demo");
    expect(out[0].name).toBe("Echo");
    expect(global.fetch).toHaveBeenCalledWith(
      "http://127.0.0.1:8080/functions",
      expect.objectContaining({ method: "GET" })
    );
  });

  it("POST /invoke sends package, function, and args JSON", async () => {
    global.fetch = jest.fn().mockResolvedValue({
      ok: true,
      status: 200,
      headers: new Headers({ "content-type": "application/json" }),
      json: async () => ({
        success: true,
        result: [42],
      }),
    }) as unknown as typeof fetch;

    const client = new ForstSidecarClient({
      baseUrl: "http://127.0.0.1:8080",
      retries: 0,
    });
    await client.invokeFunction("demo", "Echo", ["hello"]);

    expect(global.fetch).toHaveBeenCalledWith(
      "http://127.0.0.1:8080/invoke",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({
          package: "demo",
          function: "Echo",
          args: ["hello"],
          streaming: false,
        }),
      })
    );
  });

  it("POST /invoke with success: true but no result throws DevServerInvokeRejected", async () => {
    global.fetch = jest.fn().mockResolvedValue({
      ok: true,
      status: 200,
      headers: new Headers({ "content-type": "application/json" }),
      json: async () => ({
        success: true,
      }),
    }) as unknown as typeof fetch;

    const client = new ForstSidecarClient({
      baseUrl: "http://127.0.0.1:8080",
      retries: 0,
    });
    try {
      await client.invokeFunction("demo", "Echo", []);
      expect(true).toBe(false);
    } catch (e) {
      expect(e).toBeInstanceOf(DevServerInvokeRejected);
    }
  });

  it("POST /invoke with success: false throws DevServerInvokeRejected", async () => {
    global.fetch = jest.fn().mockResolvedValue({
      ok: true,
      status: 200,
      headers: new Headers({ "content-type": "application/json" }),
      json: async () => ({
        success: false,
        error: "type mismatch",
      }),
    }) as unknown as typeof fetch;

    const client = new ForstSidecarClient({
      baseUrl: "http://127.0.0.1:8080",
      retries: 0,
    });
    try {
      await client.invokeFunction("demo", "Echo", []);
      expect(true).toBe(false);
    } catch (e) {
      expect(e).toBeInstanceOf(DevServerInvokeRejected);
      const err = e as DevServerInvokeRejected;
      expect(err.invokeResponse.error).toBe("type mismatch");
    }
  });

  it("non-OK response with JSON body exposes serverErrorFromBody on DevServerHttpFailure", async () => {
    const body = JSON.stringify({
      success: false,
      error: "Package main not found",
    });
    global.fetch = jest.fn().mockResolvedValue({
      ok: false,
      status: 404,
      headers: new Headers({ "content-type": "application/json" }),
      text: async () => body,
    }) as unknown as typeof fetch;

    const client = new ForstSidecarClient({
      baseUrl: "http://127.0.0.1:8080",
      retries: 0,
    });
    try {
      await client.invokeFunction("main", "Nope", []);
      expect(true).toBe(false);
    } catch (e) {
      expect(e).toBeInstanceOf(DevServerHttpFailure);
      const err = e as DevServerHttpFailure;
      expect(err.status).toBe(404);
      expect(err.serverErrorFromBody).toBe("Package main not found");
    }
  });

  it("GET /version returns ServerVersionInfo", async () => {
    global.fetch = jest.fn().mockResolvedValue({
      ok: true,
      status: 200,
      headers: new Headers({ "content-type": "application/json" }),
      json: async () => ({
        success: true,
        result: {
          version: "0.9.0",
          commit: "abc",
          date: "2024-01-01",
          contractVersion: "1",
        },
      }),
    }) as unknown as typeof fetch;

    const client = new ForstSidecarClient({
      baseUrl: "http://127.0.0.1:8080",
      retries: 0,
    });
    const v = await client.getVersion();
    expect(v.version).toBe("0.9.0");
    expect(v.contractVersion).toBe("1");
    expect(global.fetch).toHaveBeenCalledWith(
      "http://127.0.0.1:8080/version",
      expect.objectContaining({ method: "GET" })
    );
  });

  it("healthCheck uses GET /health", async () => {
    global.fetch = jest.fn().mockResolvedValue({
      ok: true,
      status: 200,
      headers: new Headers({ "content-type": "application/json" }),
      json: async () => ({ success: true, output: "ok" }),
    }) as unknown as typeof fetch;

    const client = new ForstSidecarClient({
      baseUrl: "http://127.0.0.1:8080",
      retries: 0,
    });
    await client.healthCheck();

    expect(global.fetch).toHaveBeenCalledWith(
      "http://127.0.0.1:8080/health",
      expect.objectContaining({ method: "GET" })
    );
  });
});

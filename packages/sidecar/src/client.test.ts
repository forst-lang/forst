import { afterEach, describe, expect, it, jest } from "bun:test";
import { ForstSidecarClient } from "./client";
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
    });

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
    });

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

  it("healthCheck uses GET /health", async () => {
    global.fetch = jest.fn().mockResolvedValue({
      ok: true,
      status: 200,
      headers: new Headers({ "content-type": "application/json" }),
      json: async () => ({ success: true, output: "ok" }),
    });

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

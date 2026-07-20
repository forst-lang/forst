import { describe, expect, test, afterEach } from "bun:test";
import ForstClient, { resetDefaultInvokeClientForTest } from "./index.js";

describe("ForstClient", () => {
  afterEach(() => {
    resetDefaultInvokeClientForTest();
  });

  test("getPackage exposes async function properties", () => {
    const client = new ForstClient({
      transport: "http",
      baseUrl: "http://127.0.0.1:6321",
      customSidecar: {
        start: async () => {},
        stop: async () => {},
        getServerInfo: () => ({}),
        discoverFunctions: async () => [],
      } as never,
    });
    const bcrypt = client.getPackage("bcrypt");
    expect(typeof bcrypt.hash).toBe("function");
  });

  test("proxy allows packageName.functionName access", () => {
    const client = new ForstClient({
      transport: "http",
      baseUrl: "http://127.0.0.1:6321",
      customSidecar: {
        start: async () => {},
        stop: async () => {},
        getServerInfo: () => ({}),
        discoverFunctions: async () => [],
      } as never,
    });
    const proxied = client as ForstClient & {
      bcrypt: { hash: (...args: unknown[]) => Promise<unknown> };
    };
    expect(typeof proxied.bcrypt.hash).toBe("function");
  });

  test("stop is safe with custom sidecar", async () => {
    const client = new ForstClient({
      customSidecar: {
        start: async () => {},
        stop: async () => {},
        getServerInfo: () => ({}),
        discoverFunctions: async () => [],
      } as never,
    });
    await client.stop();
  });
});

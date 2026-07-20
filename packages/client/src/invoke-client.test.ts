import { describe, expect, test, afterEach } from "bun:test";
import {
  createInvokeClient,
  getDefaultInvokeClient,
  resetDefaultInvokeClientForTest,
} from "./invoke-client.js";

describe("createInvokeClient", () => {
  afterEach(() => {
    resetDefaultInvokeClientForTest();
    delete process.env.FORST_SKIP_SPAWN;
    delete process.env.FORST_BASE_URL;
    delete process.env.FORST_INVOKE_URL;
    delete process.env.FORST_DEV_URL;
  });

  test("getDefaultInvokeClient returns a stable singleton", () => {
    resetDefaultInvokeClientForTest();
    const a = getDefaultInvokeClient();
    const b = getDefaultInvokeClient();
    expect(a).toBe(b);
  });

  test("resetDefaultInvokeClientForTest clears the singleton", () => {
    const first = getDefaultInvokeClient();
    resetDefaultInvokeClientForTest();
    const second = getDefaultInvokeClient();
    expect(first).not.toBe(second);
  });

  test("uses connect mode when transport is http with baseUrl", () => {
    process.env.FORST_SKIP_SPAWN = "1";
    const client = createInvokeClient({
      transport: "http",
      baseUrl: "http://127.0.0.1:6321",
    });
    expect(client).toBeDefined();
    expect(typeof client.healthCheck).toBe("function");
  });

  test("uses spawn mode when sidecarRuntime is spawn", () => {
    const client = createInvokeClient({
      sidecarRuntime: "spawn",
      rootDir: process.cwd(),
    });
    expect(client).toBeDefined();
    expect(typeof client.invokeFunction).toBe("function");
  });
});

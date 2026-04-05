import { afterEach, describe, expect, it, jest } from "bun:test";
import { ForstSidecar } from "./sidecar";

describe("ForstSidecar connect mode", () => {
  const originalFetch = global.fetch;

  afterEach(() => {
    global.fetch = originalFetch;
  });

  it("attaches HTTP client without spawning when dev server responds to health", async () => {
    global.fetch = jest.fn().mockResolvedValue({
      ok: true,
      status: 200,
      headers: new Headers({ "content-type": "application/json" }),
      json: async () => ({ success: true, status: "ok" }),
      text: async () => '{"success":true}',
    }) as unknown as typeof fetch;

    const sidecar = new ForstSidecar({
      sidecarRuntime: "connect",
      devServerUrl: "http://127.0.0.1:65520",
    });
    await sidecar.start();
    expect(await sidecar.healthCheck()).toBe(true);
    const info = sidecar.getServerInfo();
    expect(info.connection).toBe("connect");
    expect(info.pid).toBe(0);
    expect(info.port).toBe(65520);
    await sidecar.stop();
  });

  it("throws when connect mode has no URL", async () => {
    const sidecar = new ForstSidecar({
      sidecarRuntime: "connect",
    });
    await expect(sidecar.start()).rejects.toThrow(/Connect mode requires/);
  });
});

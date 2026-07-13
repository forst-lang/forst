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
      versionCheck: "off",
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

  it("parks invoke on 503 during reload instead of exhausting retries", async () => {
    let invokeCalls = 0;
    let healthCalls = 0;

    global.fetch = jest.fn((url: string | URL | Request) => {
      const path = String(url);
      if (path.endsWith("/health")) {
        healthCalls += 1;
        const reloading = path.includes("/health") && healthCalls === 1;
        return Promise.resolve({
          ok: true,
          status: 200,
          statusText: "OK",
          headers: new Headers({ "content-type": "application/json" }),
          json: async () => ({
            success: true,
            result: { reloading: reloading && healthCalls < 2 },
          }),
          text: async () => '{"success":true}',
        });
      }
      if (path.endsWith("/invoke")) {
        invokeCalls += 1;
        if (invokeCalls === 1) {
          return Promise.resolve({
            ok: false,
            status: 503,
            statusText: "Service Unavailable",
            headers: new Headers({
              "content-type": "application/json",
              "Retry-After": "0",
            }),
            text: async () =>
              JSON.stringify({ success: false, error: "reloading" }),
          });
        }
        return Promise.resolve({
          ok: true,
          status: 200,
          statusText: "OK",
          headers: new Headers({ "content-type": "application/json" }),
          json: async () => ({ success: true, result: "ok" }),
        });
      }
      return Promise.reject(new Error(`unexpected url: ${path}`));
    }) as unknown as typeof fetch;

    const sidecar = new ForstSidecar({
      sidecarRuntime: "connect",
      devServerUrl: "http://127.0.0.1:65520",
      versionCheck: "off",
    });
    await sidecar.start();
    const out = await sidecar.invoke("demo", "Echo", []);
    expect(out.result).toBe("ok");
    expect(invokeCalls).toBe(2);
    await sidecar.stop();
  });
});

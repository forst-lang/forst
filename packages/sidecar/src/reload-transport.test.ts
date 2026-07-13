import { Effect } from "effect";
import { describe, expect, it } from "vitest";
import { createReloadAwareTransport } from "./reload-transport";
import type { InvokeTransport } from "./transport";

describe("createReloadAwareTransport", () => {
  it("parks invoke on connection refused and replays when health is ready", async () => {
    let invokeCalls = 0;
    let healthCalls = 0;

    const inner: InvokeTransport = {
      request(endpoint, init) {
        if (endpoint === "/health") {
          healthCalls += 1;
          if (healthCalls < 2) {
            return Effect.fail(new Error("fetch failed: ECONNREFUSED"));
          }
          return Effect.succeed(
            new Response(JSON.stringify({ success: true, reloading: false }), {
              status: 200,
              headers: { "Content-Type": "application/json" },
            })
          );
        }
        invokeCalls += 1;
        if (invokeCalls === 1) {
          return Effect.fail(new Error("fetch failed: ECONNREFUSED"));
        }
        return Effect.succeed(
          new Response(JSON.stringify({ success: true, result: { ok: true } }), {
            status: 200,
            headers: { "Content-Type": "application/json" },
          })
        );
      },
    };

    const transport = createReloadAwareTransport(inner, {
      fetchHealth: () =>
        Effect.gen(function* () {
          healthCalls += 1;
          if (healthCalls < 2) {
            return { success: false, reloading: true };
          }
          return { success: true, reloading: false };
        }),
    });

    const response = await Effect.runPromise(
      transport.request("/invoke", {
        method: "POST",
        body: JSON.stringify({ package: "main", function: "Echo", args: {} }),
      })
    );

    expect(response.status).toBe(200);
    expect(invokeCalls).toBe(2);
  });

  it("replays after connection refused once health is ready", async () => {
    let invokeCalls = 0;
    let healthChecks = 0;

    const inner: InvokeTransport = {
      request(endpoint) {
        if (endpoint === "/health") {
          return Effect.succeed(
            new Response(JSON.stringify({ success: true, reloading: false }), {
              status: 200,
              headers: { "Content-Type": "application/json" },
            })
          );
        }
        invokeCalls += 1;
        if (invokeCalls < 3) {
          return Effect.fail(new Error("fetch failed: ECONNREFUSED"));
        }
        return Effect.succeed(
          new Response(JSON.stringify({ success: true, result: { ok: true } }), {
            status: 200,
            headers: { "Content-Type": "application/json" },
          })
        );
      },
    };

    const transport = createReloadAwareTransport(inner, {
      fetchHealth: () =>
        Effect.sync(() => {
          healthChecks += 1;
          if (healthChecks < 2) {
            return { success: false, reloading: true };
          }
          return { success: true, reloading: false };
        }),
    });

    const response = await Effect.runPromise(
      transport.request("/invoke", { method: "POST", body: "{}" })
    );
    expect(response.status).toBe(200);
    expect(invokeCalls).toBe(3);
  });
});

import { Effect } from "effect";
import { describe, expect, it } from "vitest";
import { createHttpInvokeTransport } from "./transport";

describe("createHttpInvokeTransport resolveBaseUrl", () => {
  it("uses resolveBaseUrl on each request", async () => {
    let current = "http://127.0.0.1:6321";
    const seen: string[] = [];
    const transport = createHttpInvokeTransport({
      resolveBaseUrl: () => current,
      fetchFn: async (input) => {
        seen.push(String(input));
        return new Response(JSON.stringify({ success: true }), { status: 200 });
      },
    });

    await Effect.runPromise(transport.request("/health", { method: "GET" }));
    current = "http://127.0.0.1:6323";
    await Effect.runPromise(transport.request("/health", { method: "GET" }));

    expect(seen).toEqual([
      "http://127.0.0.1:6321/health",
      "http://127.0.0.1:6323/health",
    ]);
  });
});

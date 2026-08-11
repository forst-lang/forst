import { Effect } from "effect";
import { describe, expect, it } from "bun:test";
import { computeInvokeProof } from "./invoke-auth";
import { createHttpInvokeTransport } from "./transport";

describe("createHttpInvokeTransport resolveBaseUrl", () => {
  it("uses resolveBaseUrl on each request", async () => {
    let current = "http://127.0.0.1:6321";
    const seen: string[] = [];
    const transport = createHttpInvokeTransport({
      resolveBaseUrl: () => current,
      authDisabled: true,
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

describe("createHttpInvokeTransport auth headers", () => {
  it("computes and attaches a fresh proof per request", async () => {
    const token = Uint8Array.from([1, 2, 3, 4]);
    let challengeCount = 0;
    const nonces = ["nonce-a", "nonce-b"];
    const transport = createHttpInvokeTransport({
      baseUrl: "http://127.0.0.1:6320",
      resolveAuth: () => ({ token, generation: 1 }),
      fetchFn: async (input, init) => {
        const url = String(input);
        if (url.endsWith("/invoke/challenge")) {
          const nonce = nonces[challengeCount] ?? nonces[nonces.length - 1];
          challengeCount++;
          return new Response(
            JSON.stringify({
              success: true,
              result: {
                nonce,
                generation: 1,
                expiresAt: "2026-01-01T00:00:00Z",
              },
            }),
            { status: 200 }
          );
        }
        const headers = init?.headers as Record<string, string>;
        const nonce = headers["X-Forst-Invoke-Nonce"];
        expect(headers["X-Forst-Invoke-Generation"]).toBe("1");
        expect(headers["X-Forst-Invoke-Proof"]).toBe(
          computeInvokeProof(token, 1, nonce)
        );
        return new Response(JSON.stringify({ success: true }), { status: 200 });
      },
    });

    await Effect.runPromise(transport.request("/functions", { method: "GET" }));
    await Effect.runPromise(transport.request("/functions", { method: "GET" }));
    expect(challengeCount).toBe(2);
  });

  it("extraHeaders cannot override the proof header", async () => {
    const token = Uint8Array.from([9, 9, 9]);
    const transport = createHttpInvokeTransport({
      baseUrl: "http://127.0.0.1:6320",
      resolveAuth: () => ({ token, generation: 2 }),
      extraHeaders: {
        "X-Forst-Invoke-Proof": "forged",
        "X-Forst-Invoke-Generation": "99",
      },
      fetchFn: async (input, init) => {
        const url = String(input);
        if (url.endsWith("/invoke/challenge")) {
          return new Response(
            JSON.stringify({
              success: true,
              result: {
                nonce: "nonce-b",
                generation: 2,
                expiresAt: "2026-01-01T00:00:00Z",
              },
            }),
            { status: 200 }
          );
        }
        const headers = init?.headers as Record<string, string>;
        expect(headers["X-Forst-Invoke-Generation"]).toBe("2");
        expect(headers["X-Forst-Invoke-Proof"]).toBe(
          computeInvokeProof(token, 2, "nonce-b")
        );
        return new Response(JSON.stringify({ success: true }), { status: 200 });
      },
    });

    await Effect.runPromise(transport.request("/functions", { method: "GET" }));
  });
});

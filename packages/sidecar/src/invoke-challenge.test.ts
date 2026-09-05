import { Effect } from "effect";
import { describe, expect, it } from "bun:test";
import { fetchInvokeChallenge } from "./invoke-challenge";
import type { InvokeTransport } from "./transport";

describe("fetchInvokeChallenge", () => {
  it("parses a challenge response", async () => {
    const transport: InvokeTransport = {
      request() {
        return Effect.succeed(
          new Response(
            JSON.stringify({
              success: true,
              result: {
                nonce: "abc",
                generation: 1,
                expiresAt: "2026-01-01T00:00:00Z",
              },
            }),
            { status: 200, headers: { "content-type": "application/json" } }
          )
        );
      },
    };
    const challenge = await Effect.runPromise(fetchInvokeChallenge(transport));
    expect(challenge.nonce).toBe("abc");
    expect(challenge.generation).toBe(1);
  });

  it("surfaces transport errors", async () => {
    const transport: InvokeTransport = {
      request() {
        return Effect.fail(new Error("network down"));
      },
    };
    await expect(
      Effect.runPromise(fetchInvokeChallenge(transport))
    ).rejects.toThrow("network down");
  });

  it("rejects malformed serialized challenge payloads", async () => {
    const transport: InvokeTransport = {
      request() {
        return Effect.succeed(
          new Response(
            JSON.stringify({ success: true, result: "{not-json" }),
            { status: 200 }
          )
        );
      },
    };
    await expect(
      Effect.runPromise(fetchInvokeChallenge(transport))
    ).rejects.toThrow("invoke challenge missing nonce");
  });

  it("rejects non-string nonce values", async () => {
    const transport: InvokeTransport = {
      request() {
        return Effect.succeed(
          new Response(
            JSON.stringify({
              success: true,
              result: { nonce: 123, generation: 1, expiresAt: "t" },
            }),
            { status: 200 }
          )
        );
      },
    };
    await expect(
      Effect.runPromise(fetchInvokeChallenge(transport))
    ).rejects.toThrow("invoke challenge missing nonce");
  });
});

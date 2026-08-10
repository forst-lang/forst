import { describe, expect, test } from "bun:test";
import {
  computeInvokeProof,
  invokeProofMessage,
  normalizeHeaders,
  parseInvokeChallengeResult,
  stripReservedHeaders,
} from "./invoke-auth.js";

describe("computeInvokeProof", () => {
  test("matches the cross-language vector shared with Go", () => {
    const token = new TextEncoder().encode("01234567890123456789012345678901");
    const proof = computeInvokeProof(
      token,
      7,
      "nonce-for-cross-language-vector"
    );
    expect(proof).toBe("-Lb6p0ULEFekjdrV0KI-4jv-7HRVsQJknWEGqRfAHT0");
    expect(proof.length).toBeGreaterThan(10);
    expect(invokeProofMessage(7, "nonce-for-cross-language-vector")).toBe(
      "forst-invoke-v1|7|nonce-for-cross-language-vector"
    );
  });

  test("differs when generation changes", () => {
    const token = new TextEncoder().encode("01234567890123456789012345678901");
    const a = computeInvokeProof(token, 1, "nonce");
    const b = computeInvokeProof(token, 2, "nonce");
    expect(a).not.toBe(b);
  });
});

describe("stripReservedHeaders", () => {
  test("drops reserved names case-insensitively", () => {
    const out = stripReservedHeaders({
      Authorization: "Bearer app",
      "X-Forst-Invoke-Proof": "bad",
      "x-forst-invoke-nonce": "n",
    });
    expect(out).toEqual({ Authorization: "Bearer app" });
  });
});

describe("normalizeHeaders", () => {
  test("flattens Headers instances", () => {
    const headers = new Headers({ accept: "application/json" });
    expect(normalizeHeaders(headers)).toEqual({ accept: "application/json" });
  });
});

describe("parseInvokeChallengeResult", () => {
  test("accepts object result", () => {
    const got = parseInvokeChallengeResult({
      success: true,
      result: { nonce: "n1", generation: 1, expiresAt: "t" },
    });
    expect(got?.nonce).toBe("n1");
  });

  test("accepts JSON string result", () => {
    const got = parseInvokeChallengeResult({
      result: JSON.stringify({ nonce: "n2", generation: 2, expiresAt: "t" }),
    });
    expect(got?.nonce).toBe("n2");
  });

  test("returns undefined without nonce", () => {
    expect(
      parseInvokeChallengeResult({ result: { generation: 1 } as never })
    ).toBeUndefined();
  });
});

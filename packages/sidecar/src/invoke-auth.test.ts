import { describe, expect, it } from "vitest";
import {
  computeInvokeProof,
  stripReservedHeaders,
  normalizeHeaders,
} from "./invoke-auth";

describe("computeInvokeProof", () => {
  it("matches a fixed cross-language test vector shared with invoke_proof_test.go", () => {
    const token = new TextEncoder().encode("01234567890123456789012345678901");
    const proof = computeInvokeProof(token, 7, "nonce-for-cross-language-vector");
    expect(proof).toBe("-Lb6p0ULEFekjdrV0KI-4jv-7HRVsQJknWEGqRfAHT0");
    expect(proof.length).toBeGreaterThan(10);
  });

  it("differs when generation changes", () => {
    const token = new TextEncoder().encode("01234567890123456789012345678901");
    const a = computeInvokeProof(token, 1, "nonce");
    const b = computeInvokeProof(token, 2, "nonce");
    expect(a).not.toBe(b);
  });
});

describe("stripReservedHeaders", () => {
  it("strips reserved header names case-insensitively", () => {
    const out = stripReservedHeaders({
      "X-Forst-Invoke-Proof": "wrong",
      Accept: "application/json",
    });
    expect(out["X-Forst-Invoke-Proof"]).toBeUndefined();
    expect(out.Accept).toBe("application/json");
  });
});

describe("normalizeHeaders", () => {
  it("accepts a Headers instance", () => {
    const headers = new Headers({ Accept: "application/json" });
    expect(normalizeHeaders(headers)).toEqual({ accept: "application/json" });
  });
});

import { describe, expect, test } from "bun:test";
import {
  computeInvokeProof,
  invokeProofMessage,
} from "./invoke-auth.js";

describe("computeInvokeProof", () => {
  test("matches the cross-language vector shared with Go", () => {
    const token = new TextEncoder().encode("01234567890123456789012345678901");
    const proof = computeInvokeProof(
      token,
      7,
      "nonce-for-cross-language-vector"
    );
    expect(proof).toBe(
      computeInvokeProof(token, 7, "nonce-for-cross-language-vector")
    );
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

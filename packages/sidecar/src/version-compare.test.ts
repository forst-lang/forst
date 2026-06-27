import { describe, expect, it } from "bun:test";
import {
  contractVersionCompatible,
  versionsEquivalentForSidecar,
} from "./version-compare";
import { FORST_DEV_HTTP_CONTRACT_VERSION } from "./constants";

describe("versionsEquivalentForSidecar", () => {
  it("treats semver-equal strings as equal", () => {
    expect(versionsEquivalentForSidecar("1.0.0", "v1.0.0")).toBe(true);
  });

  it("falls back to exact string when semver cannot coerce", () => {
    expect(versionsEquivalentForSidecar("dev", "dev")).toBe(true);
    expect(versionsEquivalentForSidecar("dev", "prod")).toBe(false);
  });
});

describe("contractVersionCompatible", () => {
  it("matches expected dev HTTP contract constant", () => {
    expect(
      contractVersionCompatible("2", FORST_DEV_HTTP_CONTRACT_VERSION)
    ).toBe(true);
  });

  it("rejects mismatched contract strings", () => {
    expect(contractVersionCompatible("2", "1")).toBe(false);
    expect(contractVersionCompatible("1", FORST_DEV_HTTP_CONTRACT_VERSION)).toBe(
      false
    );
  });
});

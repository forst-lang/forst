import { describe, expect, test } from "bun:test";
import {
  CompilerBinaryChecksumMismatch,
  CompilerBinaryDigestUnavailable,
  CompilerBinaryDownloadFailed,
  CompilerBinaryDownloadHttpFailure,
  CompilerBinaryNotFound,
  UnsupportedArchitecture,
  UnsupportedOperatingSystem,
} from "./errors.js";

describe("CLI error types", () => {
  test("UnsupportedOperatingSystem name", () => {
    const e = new UnsupportedOperatingSystem("bad os");
    expect(e).toBeInstanceOf(Error);
    expect(e.name).toBe("UnsupportedOperatingSystem");
    expect(e.message).toBe("bad os");
  });

  test("UnsupportedArchitecture name", () => {
    const e = new UnsupportedArchitecture("bad arch");
    expect(e.name).toBe("UnsupportedArchitecture");
  });

  test("CompilerBinaryDownloadHttpFailure carries status", () => {
    const e = new CompilerBinaryDownloadHttpFailure(404, "Not Found");
    expect(e.name).toBe("CompilerBinaryDownloadHttpFailure");
    expect(e.status).toBe(404);
    expect(e.statusText).toBe("Not Found");
    expect(e.message).toContain("404");
  });

  test("CompilerBinaryDownloadFailed supports cause", () => {
    const cause = new Error("root");
    const e = new CompilerBinaryDownloadFailed("wrap", { cause });
    expect(e.name).toBe("CompilerBinaryDownloadFailed");
    expect(e.cause).toBe(cause);
  });

  test("CompilerBinaryNotFound supports cause", () => {
    const e = new CompilerBinaryNotFound("missing", { cause: undefined });
    expect(e.name).toBe("CompilerBinaryNotFound");
  });

  test("CompilerBinaryChecksumMismatch exposes hex digests and security warning", () => {
    const e = new CompilerBinaryChecksumMismatch("aa", "bb", {
      version: "0.4.0",
      artifact: "forst-darwin-arm64",
    });
    expect(e.name).toBe("CompilerBinaryChecksumMismatch");
    expect(e.expectedHex).toBe("aa");
    expect(e.actualHex).toBe("bb");
    expect(e.version).toBe("0.4.0");
    expect(e.artifact).toBe("forst-darwin-arm64");
    expect(e.message).toContain("aa");
    expect(e.message).toContain("bb");
    expect(e.message).toContain("supply-chain");
    expect(e.message).toContain("Do not run");
  });

  test("CompilerBinaryDigestUnavailable includes strict mode and reason-specific detail", () => {
    const releaseNotFound = new CompilerBinaryDigestUnavailable(
      "0.0.35",
      "forst-darwin-arm64",
      "release_not_found"
    );
    expect(releaseNotFound.name).toBe("CompilerBinaryDigestUnavailable");
    expect(releaseNotFound.reason).toBe("release_not_found");
    expect(releaseNotFound.message).toContain("FORST_CLI_VERIFY=strict");
    expect(releaseNotFound.message).toContain("v0.0.35");
    expect(releaseNotFound.message).toContain("does not exist");

    const assetNotFound = new CompilerBinaryDigestUnavailable(
      "0.4.0",
      "forst-darwin-arm64",
      "asset_not_found"
    );
    expect(assetNotFound.message).toContain("no asset forst-darwin-arm64");
    expect(assetNotFound.message).toContain("Wrong compiler version");

    const digestMissing = new CompilerBinaryDigestUnavailable(
      "0.4.0",
      "forst-darwin-arm64",
      "digest_missing"
    );
    expect(digestMissing.message).toContain("no sha256");
    expect(digestMissing.message).toContain("FORST_BINARY");
  });
});

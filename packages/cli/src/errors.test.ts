import { describe, expect, test } from "bun:test";
import {
  CompilerBinaryChecksumMismatch,
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

  test("CompilerBinaryChecksumMismatch exposes hex digests", () => {
    const e = new CompilerBinaryChecksumMismatch("aa", "bb");
    expect(e.name).toBe("CompilerBinaryChecksumMismatch");
    expect(e.expectedHex).toBe("aa");
    expect(e.actualHex).toBe("bb");
    expect(e.message).toContain("aa");
    expect(e.message).toContain("bb");
  });
});

import { describe, expect, it } from "bun:test";
import { ForstUtils } from "./utils";

describe("parseForstVersionStdout", () => {
  it("parses forst version line", () => {
    expect(
      ForstUtils.parseForstVersionStdout("forst 1.2.3 abcdef 2024-01-01\n")
    ).toBe("1.2.3");
  });

  it("returns empty on bad input", () => {
    expect(ForstUtils.parseForstVersionStdout("nope")).toBe("");
  });
});

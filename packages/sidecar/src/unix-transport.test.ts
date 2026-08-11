import { describe, expect, it } from "bun:test";
import { isUnixSocketSupported } from "./unix-transport";

describe("isUnixSocketSupported", () => {
  it("returns false on win32", () => {
    const original = process.platform;
    Object.defineProperty(process, "platform", { value: "win32" });
    try {
      expect(isUnixSocketSupported()).toBe(false);
    } finally {
      Object.defineProperty(process, "platform", { value: original });
    }
  });

  it("returns true on unix-like platforms", () => {
    if (process.platform === "win32") {
      return;
    }
    expect(isUnixSocketSupported()).toBe(true);
  });
});

import { describe, expect, test } from "bun:test";
import { readInvokeReadyUrl } from "./invoke-ready.js";

describe("readInvokeReadyUrl", () => {
  test("returns undefined when file missing", () => {
    const got = readInvokeReadyUrl("/no/such/root", {
      existsSync: () => false,
      readFileSync: () => {
        throw new Error("should not read");
      },
    });
    expect(got).toBeUndefined();
  });

  test("returns stripped url from JSON payload", () => {
    const got = readInvokeReadyUrl("/proj", {
      existsSync: (p) => String(p).endsWith("invoke.ready"),
      readFileSync: () =>
        JSON.stringify({
          url: "http://127.0.0.1:6321/",
          runtime: "embedded",
        }),
    });
    expect(got).toBe("http://127.0.0.1:6321");
  });

  test("returns undefined on invalid JSON", () => {
    const got = readInvokeReadyUrl("/proj", {
      existsSync: () => true,
      readFileSync: () => "{not-json",
    });
    expect(got).toBeUndefined();
  });
});

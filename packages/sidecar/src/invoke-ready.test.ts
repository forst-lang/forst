import { mkdirSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import { tmpdir } from "node:os";
import { describe, expect, it } from "vitest";
import { readInvokeReadyUrl } from "./invoke-ready";

describe("readInvokeReadyUrl", () => {
  it("returns url when invoke.ready exists", () => {
    const dir = join(tmpdir(), `forst-invoke-ready-${Date.now()}`);
    mkdirSync(join(dir, ".forst"), { recursive: true });
    writeFileSync(
      join(dir, ".forst", "invoke.ready"),
      JSON.stringify({
        url: "http://127.0.0.1:6323/",
        contractVersion: "1",
        runtime: "embedded",
      })
    );
    expect(readInvokeReadyUrl(dir)).toBe("http://127.0.0.1:6323");
  });

  it("returns undefined when file is missing", () => {
    const dir = join(tmpdir(), `forst-invoke-ready-missing-${Date.now()}`);
    mkdirSync(dir, { recursive: true });
    expect(readInvokeReadyUrl(dir)).toBeUndefined();
  });

  it("returns undefined for malformed json", () => {
    const dir = join(tmpdir(), `forst-invoke-ready-bad-${Date.now()}`);
    mkdirSync(join(dir, ".forst"), { recursive: true });
    writeFileSync(join(dir, ".forst", "invoke.ready"), "not-json");
    expect(readInvokeReadyUrl(dir)).toBeUndefined();
  });
});

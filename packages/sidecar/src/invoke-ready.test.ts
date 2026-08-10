import { mkdirSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import { tmpdir } from "node:os";
import { describe, expect, it } from "bun:test";
import {
  readInvokeReadyAuth,
  readInvokeReadySocketPath,
  readInvokeReadyUrl,
} from "./invoke-ready";

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

  it("returns socketPath when present in invoke.ready", () => {
    const dir = join(tmpdir(), `forst-invoke-socket-${Date.now()}`);
    mkdirSync(join(dir, ".forst"), { recursive: true });
    writeFileSync(
      join(dir, ".forst", "invoke.ready"),
      JSON.stringify({
        url: "http://127.0.0.1:6323",
        socketPath: "/tmp/forst.sock",
        generation: 1,
      })
    );
    expect(readInvokeReadySocketPath(dir)).toBe("/tmp/forst.sock");
  });

  it("returns auth bundle when ready and token file exist", () => {
    const dir = join(tmpdir(), `forst-invoke-auth-${Date.now()}`);
    mkdirSync(join(dir, ".forst"), { recursive: true });
    const token = Buffer.from("secret-token", "utf8");
    writeFileSync(
      join(dir, ".forst", "invoke.ready"),
      JSON.stringify({ url: "http://127.0.0.1:6323", generation: 3 })
    );
    writeFileSync(
      join(dir, ".forst", "invoke.token"),
      token.toString("base64url")
    );
    const auth = readInvokeReadyAuth(dir);
    expect(auth?.generation).toBe(3);
    expect(Buffer.from(auth!.token).toString("utf8")).toBe("secret-token");
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

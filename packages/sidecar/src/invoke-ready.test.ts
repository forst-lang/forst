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

  it("returns undefined when tokenDelivery is env but env token is missing", () => {
    const dir = join(tmpdir(), `forst-invoke-auth-missing-${Date.now()}`);
    mkdirSync(join(dir, ".forst"), { recursive: true });
    const prev = process.env.FORST_INVOKE_TOKEN;
    delete process.env.FORST_INVOKE_TOKEN;
    try {
      writeFileSync(
        join(dir, ".forst", "invoke.ready"),
        JSON.stringify({
          url: "http://127.0.0.1:6323",
          generation: 3,
          tokenDelivery: "env",
        })
      );
      expect(readInvokeReadyAuth(dir)).toBeUndefined();
    } finally {
      if (prev === undefined) {
        delete process.env.FORST_INVOKE_TOKEN;
      } else {
        process.env.FORST_INVOKE_TOKEN = prev;
      }
    }
  });

  it("returns auth bundle when ready and FORST_INVOKE_TOKEN are set", () => {
    const dir = join(tmpdir(), `forst-invoke-auth-${Date.now()}`);
    mkdirSync(join(dir, ".forst"), { recursive: true });
    const token = Buffer.from("secret-token", "utf8");
    const prev = process.env.FORST_INVOKE_TOKEN;
    process.env.FORST_INVOKE_TOKEN = token.toString("base64url");
    try {
      writeFileSync(
        join(dir, ".forst", "invoke.ready"),
        JSON.stringify({
          url: "http://127.0.0.1:6323",
          generation: 3,
          tokenDelivery: "env",
        })
      );
      const auth = readInvokeReadyAuth(dir);
      expect(auth?.generation).toBe(3);
      expect(Buffer.from(auth!.token).toString("utf8")).toBe("secret-token");
    } finally {
      if (prev === undefined) {
        delete process.env.FORST_INVOKE_TOKEN;
      } else {
        process.env.FORST_INVOKE_TOKEN = prev;
      }
    }
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

import { afterEach, describe, expect, test } from "bun:test";
import { mkdtempSync, mkdirSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { PassThrough } from "node:stream";

import {
  consumeHostInvokeAuthStreamForTest,
  envInvokeAuthRecvFd,
  prepareConnectInvokeEnv,
  resetHostInvokeAuthHandoffForTest,
  resolveHostInvokeAuthHandoff,
  startHostInvokeAuthRecvListener,
} from "./host-invoke-auth.js";
import { DEFAULT_EMBEDDED_INVOKE_BASE_URL } from "./constants.js";

afterEach(() => {
  resetHostInvokeAuthHandoffForTest();
  delete process.env[envInvokeAuthRecvFd];
  delete process.env.FORST_SKIP_SPAWN;
  delete process.env.FORST_BOUNDARY_ROOT;
  delete process.env.FORST_BASE_URL;
  delete process.env.FORST_INVOKE_URL;
  delete process.env.FORST_DEV_URL;
});

describe("consumeHostInvokeAuthStreamForTest", () => {
  test("stores valid handoff lines and ignores malformed ones", async () => {
    const stream = new PassThrough();
    const done = consumeHostInvokeAuthStreamForTest(stream);

    stream.write("not-json\n");
    stream.write("null\n");
    stream.write(JSON.stringify({ generation: 0, token: "AQID" }) + "\n");
    const token = Buffer.from([1, 2, 3]).toString("base64url");
    stream.write(JSON.stringify({ generation: 7, token }) + "\n");
    stream.end();
    await done;

    const got = resolveHostInvokeAuthHandoff();
    expect(got?.generation).toBe(7);
    expect(Buffer.from(got!.token)).toEqual(Buffer.from([1, 2, 3]));
  });
});

describe("startHostInvokeAuthRecvListener", () => {
  test("is a no-op without recv fd env", () => {
    startHostInvokeAuthRecvListener();
    startHostInvokeAuthRecvListener();
    expect(resolveHostInvokeAuthHandoff()).toBeUndefined();
  });

  test("is a no-op for invalid recv fd values", () => {
    process.env[envInvokeAuthRecvFd] = "1";
    startHostInvokeAuthRecvListener();
    expect(resolveHostInvokeAuthHandoff()).toBeUndefined();
  });
});

describe("prepareConnectInvokeEnv", () => {
  test("sets skip-spawn and default base url when ready file is absent", () => {
    const root = prepareConnectInvokeEnv("/tmp/forst-boundary");
    expect(root).toBe("/tmp/forst-boundary");
    expect(process.env.FORST_SKIP_SPAWN).toBe("1");
    expect(process.env.FORST_BOUNDARY_ROOT).toBe("/tmp/forst-boundary");
    expect(process.env.FORST_BASE_URL).toBe(DEFAULT_EMBEDDED_INVOKE_BASE_URL);
  });

  test("prefers ready url when present", () => {
    const root = mkdtempSync(join(tmpdir(), "forst-host-auth-"));
    mkdirSync(join(root, ".forst"), { recursive: true });
    writeFileSync(
      join(root, ".forst", "invoke.ready"),
      JSON.stringify({ url: "http://127.0.0.1:9999/" })
    );
    try {
      prepareConnectInvokeEnv(root);
      expect(process.env.FORST_BASE_URL).toBe("http://127.0.0.1:9999");
    } finally {
      rmSync(root, { recursive: true, force: true });
    }
  });
});

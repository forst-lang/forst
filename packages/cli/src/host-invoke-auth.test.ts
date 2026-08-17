import { afterEach, describe, expect, test } from "bun:test";
import { mkdtempSync, mkdirSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { PassThrough } from "node:stream";

import {
  consumeHostInvokeAuthStreamForTest,
  getInvokeAuthHandoff,
  prepareInvokeConnect,
  resetHostInvokeAuthHandoffForTest,
} from "./host-invoke-auth.js";
import { DEFAULT_EMBEDDED_INVOKE_BASE_URL } from "./constants.js";

const recvFdEnv = "FORST_INVOKE_AUTH_RECV_FD";

afterEach(() => {
  resetHostInvokeAuthHandoffForTest();
  delete process.env[recvFdEnv];
  delete process.env.FORST_SKIP_SPAWN;
  delete process.env.FORST_ROOT;
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

    const got = getInvokeAuthHandoff();
    expect(got?.generation).toBe(7);
    expect(Buffer.from(got!.token)).toEqual(Buffer.from([1, 2, 3]));
  });
});

describe("getInvokeAuthHandoff", () => {
  test("returns undefined without recv fd env", () => {
    expect(getInvokeAuthHandoff()).toBeUndefined();
  });

  test("returns undefined for invalid recv fd values", () => {
    process.env[recvFdEnv] = "1";
    expect(getInvokeAuthHandoff()).toBeUndefined();
  });
});

describe("prepareInvokeConnect", () => {
  test("sets skip-spawn and default base url when ready file is absent", () => {
    const root = prepareInvokeConnect("/tmp/forst-boundary");
    expect(root).toBe("/tmp/forst-boundary");
    expect(process.env.FORST_SKIP_SPAWN).toBe("1");
    expect(process.env.FORST_ROOT).toBe("/tmp/forst-boundary");
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
      prepareInvokeConnect(root);
      expect(process.env.FORST_BASE_URL).toBe("http://127.0.0.1:9999");
    } finally {
      rmSync(root, { recursive: true, force: true });
    }
  });
});

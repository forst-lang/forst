import { describe, expect, test } from "bun:test";
import { mkdirSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import { tmpdir } from "node:os";
import {
  readInvokeReadyAuth,
  readInvokeReadyUrl,
  readInvokeTokenFromEnv,
} from "./invoke-ready.js";

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

  test("returns undefined when url field has wrong type", () => {
    const got = readInvokeReadyUrl("/proj", {
      existsSync: () => true,
      readFileSync: () => JSON.stringify({ url: 6321 }),
    });
    expect(got).toBeUndefined();
  });
});

describe("readInvokeTokenFromEnv", () => {
  test("returns decoded token from FORST_INVOKE_TOKEN", () => {
    const prev = process.env.FORST_INVOKE_TOKEN;
    process.env.FORST_INVOKE_TOKEN = Buffer.from("secret-token", "utf8").toString(
      "base64url"
    );
    try {
      const token = readInvokeTokenFromEnv();
      expect(Buffer.from(token!).toString("utf8")).toBe("secret-token");
    } finally {
      if (prev === undefined) {
        delete process.env.FORST_INVOKE_TOKEN;
      } else {
        process.env.FORST_INVOKE_TOKEN = prev;
      }
    }
  });

  test("returns undefined when env is missing", () => {
    const prev = process.env.FORST_INVOKE_TOKEN;
    delete process.env.FORST_INVOKE_TOKEN;
    try {
      expect(readInvokeTokenFromEnv()).toBeUndefined();
    } finally {
      if (prev === undefined) {
        delete process.env.FORST_INVOKE_TOKEN;
      } else {
        process.env.FORST_INVOKE_TOKEN = prev;
      }
    }
  });
});

describe("readInvokeReadyAuth", () => {
  test("returns auth bundle when ready and FORST_INVOKE_TOKEN are set", () => {
    const dir = join(tmpdir(), `forst-cli-invoke-auth-${Date.now()}`);
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

  test("returns undefined when tokenDelivery is env but env token is missing", () => {
    const dir = join(tmpdir(), `forst-cli-invoke-auth-missing-${Date.now()}`);
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

  test("returns undefined when tokenDelivery is handoff", () => {
    const dir = join(tmpdir(), `forst-cli-invoke-handoff-${Date.now()}`);
    mkdirSync(join(dir, ".forst"), { recursive: true });
    const prev = process.env.FORST_INVOKE_TOKEN;
    process.env.FORST_INVOKE_TOKEN = Buffer.from("secret-token", "utf8").toString(
      "base64url"
    );
    try {
      writeFileSync(
        join(dir, ".forst", "invoke.ready"),
        JSON.stringify({
          generation: 3,
          tokenDelivery: "handoff",
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
});

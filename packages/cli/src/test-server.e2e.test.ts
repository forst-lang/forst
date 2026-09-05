/**
 * Opt-in real spawn against examples/in/rfc/embedded-invoke.
 *
 *   FORST_CLI_INVOKE_E2E=1 bun test src/test-server.e2e.test.ts
 *
 * Requires a usable forst binary (FORST_BINARY or repo bin/forst) and a Go toolchain
 * for `forst run` (embedded pays a compile).
 */
import { describe, expect, test } from "bun:test";
import { existsSync, rmSync } from "node:fs";
import { join, resolve } from "node:path";
import { fetchAuthenticatedInvoke } from "./invoke-auth.js";
import { startForstInvokeServer } from "./test-server.js";


const enabled = process.env.FORST_CLI_INVOKE_E2E === "1";
const repoRoot = resolve(import.meta.dir, "../../..");
const exampleRoot = join(repoRoot, "examples/in/rfc/embedded-invoke");

function resolveForstBinary(): string {
  const fromEnv = process.env.FORST_BINARY?.trim();
  if (fromEnv && existsSync(fromEnv)) return fromEnv;
  const fromRepo = join(repoRoot, "bin/forst");
  if (existsSync(fromRepo)) return fromRepo;
  throw new Error(
    "forst binary not found (set FORST_BINARY or run task build)"
  );
}

function processAlive(pid: number): boolean {
  try {
    process.kill(pid, 0);
    return true;
  } catch {
    return false;
  }
}

describe("startForstInvokeServer e2e (opt-in)", () => {
  test.skipIf(!enabled)(
    "spawns embedded-invoke, POST /invoke Echo, then stops with no orphan",
    async () => {
      expect(existsSync(join(exampleRoot, "main.ft"))).toBe(true);
      // Stale ready/token/socket files force the attach path. Remove so this test always spawns.
      rmSync(join(exampleRoot, ".forst", "invoke.ready"), { force: true });
      rmSync(join(exampleRoot, ".forst", "invoke.token"), { force: true });
      rmSync(join(exampleRoot, ".forst", "invoke.sock"), { force: true });
      const bin = resolveForstBinary();
      const cleanEnv: NodeJS.ProcessEnv = {
        PATH: process.env.PATH,
        HOME: process.env.HOME,
        TMPDIR: process.env.TMPDIR,
        FORST_BINARY: bin,
      };

      const handle = await startForstInvokeServer(
        {
          root: exampleRoot,
          mode: "embedded",
          timeoutMs: 180_000,
          logLevel: "error",
          resolveOptions: { allowDownload: false },
        },
        { env: cleanEnv }
      );

      expect(handle.connection).toBe("spawn");
      expect(handle.pid).toBeDefined();
      const pid = handle.pid!;
      expect(processAlive(pid)).toBe(true);

      try {
        const auth = handle.auth;
        expect(auth).toBeDefined();
        const res = await fetchAuthenticatedInvoke(
          {
            baseUrl: handle.baseUrl,
            socketPath: handle.socketPath ?? undefined,
          },
          {
            package: "main",
            function: "Echo",
            args: [{ message: "hello" }],
          },
          auth!
        );
        expect(res.ok).toBe(true);
        const body = (await res.json()) as {
          success?: boolean;
          result?: { echo?: string; timestamp?: number };
        };
        expect(body.success).toBe(true);
        expect(body.result?.echo).toBe("hello");
        expect(body.result?.timestamp).toBe(42);
      } finally {
        await handle.stop();
        await handle.stop();
      }

      // Give the OS a moment to reap; fail if still alive.
      const deadline = Date.now() + 5_000;
      while (Date.now() < deadline && processAlive(pid)) {
        await new Promise((r) => setTimeout(r, 50));
      }
      expect(processAlive(pid)).toBe(false);
    },
    { timeout: 240_000 }
  );
});

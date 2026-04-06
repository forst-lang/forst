/**
 * Smoke-test: merged-package `forst run` on server.ft (same as task example:tictactoe).
 */
import { describe, it, expect } from "bun:test";
import { existsSync } from "node:fs";
import { join } from "node:path";

const exampleRoot = join(import.meta.dir, "..");
const serverFt = join(exampleRoot, "server.ft");

function resolveForstBinary(): string {
  const env = process.env.FORST_BINARY?.trim();
  if (env && existsSync(env)) return env;
  const fromRepo = join(exampleRoot, "..", "..", "bin", "forst");
  if (existsSync(fromRepo)) return fromRepo;
  throw new Error(
    `forst binary not found (set FORST_BINARY or build with task build). Tried: ${fromRepo}`
  );
}

describe("forst run (merged tictactoe)", () => {
  if (process.env.FORST_SKIP_TICTACTOE_E2E === "1") {
    it.skip("skipped (FORST_SKIP_TICTACTOE_E2E=1)", () => {});
    return;
  }

  it(
    "prints server.ft demo output",
    async () => {
      const forst = resolveForstBinary();
      const proc = Bun.spawn(
        [forst, "run", "-root", exampleRoot, "--", serverFt],
        {
          stdout: "pipe",
          stderr: "pipe",
          cwd: join(exampleRoot, "..", "..", "..", "forst"),
        }
      );
      const stdout = await new Response(proc.stdout).text();
      const stderr = await new Response(proc.stderr).text();
      const code = await proc.exited;
      expect(code).toBe(0);
      const out = stdout + stderr;
      expect(out).toContain("tic-tac-toe");
      expect(out).toContain("after X @ (1,2)");
    },
    60_000
  );
});

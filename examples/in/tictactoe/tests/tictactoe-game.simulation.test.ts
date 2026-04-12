/**
 * End-to-end: TypeScript uses **`ForstClient`** from **`client/`** only (no `generated/` imports).
 * Calls **`engine.NewGame`** / **`engine.PlayMove`** via **`forst dev`** → **`POST /invoke`** (same as production).
 *
 * Requires `task example:tictactoe:generate` so `client/types.d.ts` and `client/index.ts` exist.
 * Spawns **`forst dev`** with the example `ftconfig.json` (see `beforeAll`).
 */
import { afterAll, beforeAll, describe, expect, it } from "bun:test";
import { existsSync } from "node:fs";
import { join } from "node:path";
import { DevServerHttpFailure } from "@forst/sidecar";
import { ForstClient } from "../client";
import type { GameState, MoveRequest } from "../client/types";

const exampleRoot = join(import.meta.dir, "..");
const typesFile = join(exampleRoot, "client", "types.d.ts");
const ftconfigPath = join(exampleRoot, "ftconfig.json");

function resolveForstBinary(): string {
  const env = process.env.FORST_BINARY?.trim();
  if (env && existsSync(env)) return env;
  const fromRepo = join(exampleRoot, "..", "..", "..", "bin", "forst");
  if (existsSync(fromRepo)) return fromRepo;
  throw new Error(
    `forst binary not found (set FORST_BINARY or build with task build). Tried: ${fromRepo}`
  );
}

function forstRepoDir(): string {
  return join(exampleRoot, "..", "..", "..", "forst");
}

function devListenPort(): number {
  const raw = process.env.FORST_TICTACTOE_DEV_PORT?.trim();
  if (raw && /^\d+$/.test(raw)) return parseInt(raw, 10);
  return 8099;
}

async function waitForHealth(baseUrl: string, timeoutMs: number): Promise<void> {
  const deadline = Date.now() + timeoutMs;
  const url = `${baseUrl.replace(/\/$/, "")}/health`;
  let lastErr: unknown;
  while (Date.now() < deadline) {
    try {
      const r = await fetch(url);
      if (r.ok) {
        const j = (await r.json()) as { success?: boolean };
        if (j.success === true) return;
      }
    } catch (e) {
      lastErr = e;
    }
    await new Promise((r) => setTimeout(r, 100));
  }
  throw new Error(
    `timed out waiting for ${url} (${lastErr instanceof Error ? lastErr.message : String(lastErr)})`
  );
}

let devProc: ReturnType<typeof Bun.spawn> | null = null;
let client: ForstClient | null = null;
let baseUrl = "";

beforeAll(async () => {
  if (process.env.FORST_SKIP_TICTACTOE_E2E === "1") return;
  if (!existsSync(typesFile)) {
    throw new Error(
      `Missing ${typesFile}. Run: task example:tictactoe:generate (from repo root)`
    );
  }
  if (!existsSync(ftconfigPath)) {
    throw new Error(`Missing ${ftconfigPath}`);
  }

  const forst = resolveForstBinary();
  const port = devListenPort();
  baseUrl = `http://127.0.0.1:${port}`;

  devProc = Bun.spawn(
    [
      forst,
      "dev",
      "-root",
      exampleRoot,
      "-config",
      ftconfigPath,
      "-port",
      String(port),
      "-log-level",
      "error",
    ],
    {
      cwd: forstRepoDir(),
      stdout: "pipe",
      stderr: "pipe",
    }
  );

  await waitForHealth(baseUrl, 60_000);

  client = new ForstClient({
    baseUrl,
    retries: 0,
    timeout: 60_000,
  });
});

afterAll(() => {
  if (devProc) {
    try {
      devProc.kill();
    } catch {
      /* ignore */
    }
    devProc = null;
  }
  client = null;
});

describe("tictactoe game (ForstClient + forst dev)", () => {
  if (process.env.FORST_SKIP_TICTACTOE_E2E === "1") {
    it.skip("skipped (FORST_SKIP_TICTACTOE_E2E=1)", () => { });
    return;
  }

  it("runs NewGame then one move and rejects duplicate square via server", async () => {
    expect(client).not.toBeNull();
    const c = client!;

    let state = await c.engine.NewGame();
    expect(state.cells).toHaveLength(9);
    expect(state.nextPlayer).toBe("X");

    const req: MoveRequest = { state, row: 1, col: 2 };
    const r1 = await c.engine.PlayMove(req);
    expect(r1.state.nextPlayer).toBe("O");
    expect(r1.state.cells[5]).toBe("X");

    try {
      await c.engine.PlayMove({ state: r1.state, row: 1, col: 2 });
      expect.unreachable("expected duplicate move to fail");
    } catch (e) {
      expect(e).toBeInstanceOf(DevServerHttpFailure);
      const http = e as DevServerHttpFailure;
      const blob = `${http.serverErrorFromBody ?? ""} ${http.responseText}`;
      // Duplicate square fails `ensure` in engine.ft (runtime reports Bool.True / assertion; message text may vary by emit).
      expect(blob).toMatch(/cell already taken|Bool\.True\(\)|assertion failed/);
    }
  });

  it("plays five moves so X wins the top row (server-side rules)", async () => {
    expect(client).not.toBeNull();
    const c = client!;

    let state: GameState = await c.engine.NewGame();
    const moves: [number, number][] = [
      [0, 0],
      [1, 0],
      [0, 1],
      [1, 1],
      [0, 2],
    ];
    for (const [row, col] of moves) {
      const r = await c.engine.PlayMove({ state, row, col });
      state = r.state;
    }
    expect(state.cells[0]).toBe("X");
    expect(state.cells[1]).toBe("X");
    expect(state.cells[2]).toBe("X");
  });
});

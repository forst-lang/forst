/**
 * End-to-end: @forst/tictactoe $pkg namespace against a real invoke server.
 * Lifecycle comes from startForstTestServer (@forst/cli/invoke peer).
 *
 * Requires `task example:tictactoe:generate` so node_modules/@forst/tictactoe exists.
 */
import { afterAll, beforeAll, describe, expect, it } from "bun:test";
import { existsSync, rmSync } from "node:fs";
import { join } from "node:path";
import type { $GameState, $MoveRequest } from "@forst/tictactoe";
import { $main } from "@forst/tictactoe/main";
import { $CellTaken } from "@forst/tictactoe/main/errors";
import {
  startForstTestServer,
  type ForstTestServer,
} from "@forst/tictactoe/$testing";

const exampleRoot = join(import.meta.dir, "..");
const typesFile = join(exampleRoot, ".forst", "client", "dist", "types.d.ts");

function resolveForstBinary(): string {
  const env = process.env.FORST_BINARY?.trim();
  if (env && existsSync(env)) return env;
  const fromRepo = join(exampleRoot, "..", "..", "..", "bin", "forst");
  if (existsSync(fromRepo)) return fromRepo;
  throw new Error(
    `forst binary not found (set FORST_BINARY or build with task build). Tried: ${fromRepo}`
  );
}

let server: ForstTestServer | null = null;

beforeAll(async () => {
  if (process.env.FORST_SKIP_TICTACTOE_E2E === "1") return;
  if (!existsSync(typesFile)) {
    throw new Error(
      `Missing ${typesFile}. Run: task example:tictactoe:generate (from repo root)`
    );
  }

  // Stale ready files force attach. This suite always wants a fresh spawn.
  rmSync(join(exampleRoot, ".forst", "invoke.ready"), { force: true });

  const bin = resolveForstBinary();
  server = await startForstTestServer({
    root: exampleRoot,
    mode: "dev",
    timeoutMs: 60_000,
    logLevel: "error",
    env: { FORST_BINARY: bin },
  });
}, 90_000);

afterAll(async () => {
  if (server) {
    await server.stop();
    server = null;
  }
});

describe("tictactoe game (startForstTestServer + $main namespace)", () => {
  if (process.env.FORST_SKIP_TICTACTOE_E2E === "1") {
    it.skip("skipped (FORST_SKIP_TICTACTOE_E2E=1)", () => {});
    return;
  }

  it("runs NewGame then one move and rejects duplicate square via server", async () => {
    expect(server).not.toBeNull();

    let state = await $main.NewGame();
    expect(state.cells).toHaveLength(9);
    expect(state.nextPlayer).toBe("X");

    const req: $MoveRequest = { state, row: 1, col: 2 };
    const r1 = await $main.PlayMove(req);
    expect(r1.state.nextPlayer).toBe("O");
    expect(r1.state.cells[5]).toBe("X");

    try {
      await $main.PlayMove({ state: r1.state, row: 1, col: 2 });
      expect.unreachable("expected duplicate move to fail");
    } catch (e) {
      expect(e).toBeInstanceOf($CellTaken);
      const taken = e as $CellTaken;
      expect(taken.row).toBe(1);
      expect(taken.col).toBe(2);
    }
  });

  it("plays five moves so X wins the top row (server-side rules)", async () => {
    expect(server).not.toBeNull();

    let state: $GameState = await $main.NewGame();
    const moves: [number, number][] = [
      [0, 0],
      [1, 0],
      [0, 1],
      [1, 1],
      [0, 2],
    ];
    for (const [row, col] of moves) {
      const r = await $main.PlayMove({ state, row, col });
      state = r.state;
    }
    expect(state.cells[0]).toBe("X");
    expect(state.cells[1]).toBe("X");
    expect(state.cells[2]).toBe("X");
    expect(state.status).toBe("x_won");
  });
});

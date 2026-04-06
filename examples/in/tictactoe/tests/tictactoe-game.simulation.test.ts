/**
 * Full game in TypeScript using types from the **`client/`** tree (`forst generate` copies merged `types.d.ts` there).
 * Move rules mirror `engine.ft` (ApplyMove / invalid moves).
 * Requires `task example:tictactoe:generate` so `client/types.d.ts` exists.
 */
import { describe, it, expect } from "bun:test";
import { existsSync } from "node:fs";
import { join } from "node:path";
import type { GameState, MoveRequest, MoveResponse } from "../client/types";

const exampleRoot = join(import.meta.dir, "..");
const typesFile = join(exampleRoot, "client", "types.d.ts");

function invalidMove(msg: string): Error {
  return new Error(msg);
}

/** Mirrors `ApplyMove` in engine.ft for valid/invalid moves. */
function applyMove(req: MoveRequest): { ok: MoveResponse; err?: undefined } | { ok?: undefined; err: Error } {
  const row = req.row;
  if (row <= -1) return { err: invalidMove("row must be >= 0") };
  if (row >= 3) return { err: invalidMove("row must be <= 2") };
  const col = req.col;
  if (col <= -1) return { err: invalidMove("col must be >= 0") };
  if (col >= 3) return { err: invalidMove("col must be <= 2") };
  const idx = row * 3 + col;
  if (req.state.cells[idx] !== "") {
    return { err: invalidMove("cell already taken") };
  }
  const next = [...req.state.cells];
  next[idx] = req.state.nextPlayer;
  const np = req.state.nextPlayer === "X" ? "O" : "X";
  return {
    ok: {
      ok: true,
      message: "ok",
      state: {
        cells: next,
        nextPlayer: np,
        status: req.state.status,
      },
    },
  };
}

function newGame(): GameState {
  return {
    cells: Array(9).fill(""),
    nextPlayer: "X",
    status: "playing",
  };
}

describe("tictactoe game (TS simulation + client types)", () => {
  if (!existsSync(typesFile)) {
    throw new Error(
      `Missing ${typesFile}. Run: task example:tictactoe:generate (from repo root)`
    );
  }

  if (process.env.FORST_SKIP_TICTACTOE_E2E === "1") {
    it.skip("skipped (FORST_SKIP_TICTACTOE_E2E=1)", () => { });
    return;
  }

  it("runs NewGame then one move and rejects duplicate square", () => {
    let state = newGame();
    expect(state.cells).toHaveLength(9);
    expect(state.nextPlayer).toBe("X");

    const r1 = applyMove({ state, row: 1, col: 2 });
    if (!r1.ok) throw r1.err;
    expect(r1.ok.ok).toBe(true);
    expect(r1.ok.state.nextPlayer).toBe("O");
    expect(r1.ok.state.cells[5]).toBe("X");

    const bad = applyMove({ state: r1.ok.state, row: 1, col: 2 });
    expect(bad.err).toBeDefined();
    expect(bad.err?.message).toContain("cell already taken");
  });

  it("plays five moves so X wins the top row", () => {
    let state = newGame();
    const moves: [number, number][] = [
      [0, 0],
      [1, 0],
      [0, 1],
      [1, 1],
      [0, 2],
    ];
    for (const [row, col] of moves) {
      const r = applyMove({ state, row, col });
      if (!r.ok) throw r.err;
      expect(r.ok.ok).toBe(true);
      state = r.ok.state;
    }
    expect(state.cells[0]).toBe("X");
    expect(state.cells[1]).toBe("X");
    expect(state.cells[2]).toBe("X");
  });
});

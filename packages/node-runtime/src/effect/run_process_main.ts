import { Effect } from "effect";

/** True when running under Deno (npm compat or native). */
export function isDeno(): boolean {
  return typeof (globalThis as { Deno?: unknown }).Deno !== "undefined";
}

/** Runs an Effect program as the process main entry (Node/Bun via NodeRuntime, Deno via runPromise). */
export async function runProcessMain(
  program: Effect.Effect<void, unknown, never>
): Promise<void> {
  if (isDeno()) {
    await Effect.runPromise(program);
    return;
  }
  const { NodeRuntime } = await import("@effect/platform-node");
  NodeRuntime.runMain(program, { disablePrettyLogger: true });
}

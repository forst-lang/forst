import { Effect } from "effect";
import { makeForstNodeRuntimeLayer } from "../../src/effect/layer.js";

/** Runs an Effect in tests with the same runtime layer as production code. */
export const runTestEffect = <A, E>(eff: Effect.Effect<A, E, never>) =>
  Effect.runPromise(eff.pipe(Effect.provide(makeForstNodeRuntimeLayer())));

import { Effect } from "effect";
import { makeForstNodeRuntimeLayer } from "../../src/effect/layer.js";

export const runTestEffect = <A, E>(eff: Effect.Effect<A, E, never>) =>
  Effect.runPromise(eff.pipe(Effect.provide(makeForstNodeRuntimeLayer())));

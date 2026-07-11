import { Layer, ManagedRuntime } from "effect";
import { ForstNodeRuntimeLayer } from "./layer.js";

export type ForstNodeRuntime = ManagedRuntime.ManagedRuntime<never, never>;

/** Build a managed runtime from any Effect layer (logging, tracing, services). */
export function makeForstNodeRuntime(
  layer: Layer.Layer<never>
): ForstNodeRuntime {
  return ManagedRuntime.make(layer);
}

/** Default runtime using stderr pretty logging and `FORST_NODE_LOG_LEVEL`. */
export const forstNodeRuntime = makeForstNodeRuntime(ForstNodeRuntimeLayer);

export interface NodeRuntimeSetup {
  layer: Layer.Layer<never>;
  runtime: ForstNodeRuntime;
}

/**
 * Bundle a layer with its managed runtime for boundary `Effect.provide` and
 * async sites (proto loop, host RPC forks) that cannot inherit caller context.
 */
export function createNodeRuntimeSetup(
  layer: Layer.Layer<never> = ForstNodeRuntimeLayer
): NodeRuntimeSetup {
  return { layer, runtime: makeForstNodeRuntime(layer) };
}

export const defaultNodeRuntimeSetup = createNodeRuntimeSetup(
  ForstNodeRuntimeLayer
);

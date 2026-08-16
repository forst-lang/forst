import { Effect, Layer } from "effect";
import { makeForstNodeRuntimeLayer } from "./layer.js";

/** Env gate duplicated here so the main bundle never statically imports OTEL. */
function openTelemetryRequestedFromEnv(): boolean {
  if (process.env.FORST_NODE_OTEL?.trim() === "1") {
    return true;
  }
  const endpoint = process.env.OTEL_EXPORTER_OTLP_ENDPOINT?.trim();
  return endpoint !== undefined && endpoint.length > 0;
}

/**
 * Resolves the runtime layer for bootstrap and other boundaries.
 * Merges OpenTelemetry when env is set and optional peers are installed.
 */
export function resolveForstNodeRuntimeLayer(): Effect.Effect<
  Layer.Layer<never>,
  never,
  never
> {
  return Effect.gen(function* () {
    if (!openTelemetryRequestedFromEnv()) {
      return makeForstNodeRuntimeLayer();
    }

    const otelModule = yield* Effect.promise(() =>
      import("./opentelemetry.js").catch(() => null)
    );

    if (otelModule === null) {
      yield* Effect.logWarning(
        "OpenTelemetry requested (FORST_NODE_OTEL or OTEL_EXPORTER_OTLP_ENDPOINT) but @forst/node-runtime/opentelemetry peers are not installed; continuing with stderr spans only"
      ).pipe(Effect.annotateLogs({ event: "otel_peers_missing" }));
      return makeForstNodeRuntimeLayer();
    }

    return otelModule.mergeForstNodeRuntimeWithOpenTelemetry(
      otelModule.openTelemetryLayerFromEnv()
    );
  });
}

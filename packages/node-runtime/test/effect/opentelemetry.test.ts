import { afterEach, describe, expect, test } from "bun:test";
import { Effect } from "effect";
import type { ReadableSpan, SpanExporter } from "@opentelemetry/sdk-trace-base";
import { SimpleSpanProcessor } from "@opentelemetry/sdk-trace-base";
import {
  envOpenTelemetry,
  envOtelExporterOtlpEndpoint,
  mergeForstNodeRuntimeWithOpenTelemetry,
  makeForstNodeOpenTelemetryLayer,
  openTelemetryEnabledFromEnv,
} from "../../src/effect/opentelemetry.js";
import { resolveForstNodeRuntimeLayer } from "../../src/effect/resolve_layer.js";
import {
  envLogFormat,
  envLogLevel,
  setWriteStderrForTest,
} from "../../src/effect/layer.js";

function capturingSpanExporter(): {
  exporter: SpanExporter;
  spans: ReadableSpan[];
} {
  const spans: ReadableSpan[] = [];
  const exporter: SpanExporter = {
    export(items, resultCallback) {
      spans.push(...items);
      resultCallback({ code: 0 });
    },
    shutdown: () => Promise.resolve(),
    forceFlush: () => Promise.resolve(),
  };
  return { exporter, spans };
}

describe("openTelemetryEnabledFromEnv", () => {
  const prevOtel = process.env[envOpenTelemetry];
  const prevEndpoint = process.env[envOtelExporterOtlpEndpoint];

  afterEach(() => {
    if (prevOtel === undefined) {
      delete process.env[envOpenTelemetry];
    } else {
      process.env[envOpenTelemetry] = prevOtel;
    }
    if (prevEndpoint === undefined) {
      delete process.env[envOtelExporterOtlpEndpoint];
    } else {
      process.env[envOtelExporterOtlpEndpoint] = prevEndpoint;
    }
  });

  test("is false by default", () => {
    delete process.env[envOpenTelemetry];
    delete process.env[envOtelExporterOtlpEndpoint];
    expect(openTelemetryEnabledFromEnv()).toBe(false);
  });

  test("is true when FORST_NODE_OTEL=1", () => {
    process.env[envOpenTelemetry] = "1";
    delete process.env[envOtelExporterOtlpEndpoint];
    expect(openTelemetryEnabledFromEnv()).toBe(true);
  });

  test("is true when OTEL_EXPORTER_OTLP_ENDPOINT is set", () => {
    delete process.env[envOpenTelemetry];
    process.env[envOtelExporterOtlpEndpoint] =
      "http://localhost:4318/v1/traces";
    expect(openTelemetryEnabledFromEnv()).toBe(true);
  });
});

describe("mergeForstNodeRuntimeWithOpenTelemetry", () => {
  test("exports Effect.fn spans through a capturing OTEL exporter", async () => {
    const { exporter, spans } = capturingSpanExporter();
    const otelLayer = makeForstNodeOpenTelemetryLayer({
      serviceName: "test-node-runtime",
      spanProcessor: new SimpleSpanProcessor(exporter),
    });
    const layer = mergeForstNodeRuntimeWithOpenTelemetry(otelLayer);

    const runSpan = Effect.fn("Rpc.dispatch")(function* () {
      yield* Effect.annotateCurrentSpan("rpc_method", "forst.node/ping");
      yield* Effect.logInfo("done");
    });

    await Effect.runPromise(runSpan().pipe(Effect.provide(layer), Effect.scoped));

    expect(spans.length).toBeGreaterThan(0);
    expect(spans.some((span) => span.name === "Rpc.dispatch")).toBe(true);
  });
});

describe("resolveForstNodeRuntimeLayer", () => {
  const prevOtel = process.env[envOpenTelemetry];
  const prevEndpoint = process.env[envOtelExporterOtlpEndpoint];
  const prevLevel = process.env[envLogLevel];
  const prevFormat = process.env[envLogFormat];

  afterEach(() => {
    if (prevOtel === undefined) {
      delete process.env[envOpenTelemetry];
    } else {
      process.env[envOpenTelemetry] = prevOtel;
    }
    if (prevEndpoint === undefined) {
      delete process.env[envOtelExporterOtlpEndpoint];
    } else {
      process.env[envOtelExporterOtlpEndpoint] = prevEndpoint;
    }
    if (prevLevel === undefined) {
      delete process.env[envLogLevel];
    } else {
      process.env[envLogLevel] = prevLevel;
    }
    if (prevFormat === undefined) {
      delete process.env[envLogFormat];
    } else {
      process.env[envLogFormat] = prevFormat;
    }
  });

  test("returns stderr-only layer when OTEL env is unset", async () => {
    delete process.env[envOpenTelemetry];
    delete process.env[envOtelExporterOtlpEndpoint];

    const lines: string[] = [];
    const restore = setWriteStderrForTest((text) => {
      lines.push(text);
    });

    try {
      const layer = await Effect.runPromise(resolveForstNodeRuntimeLayer());
      await Effect.runPromise(
        Effect.logInfo("resolve smoke").pipe(Effect.provide(layer))
      );
      expect(lines.join("")).toContain("resolve smoke");
    } finally {
      restore();
    }
  });

  test("merges OTEL layer when FORST_NODE_OTEL=1 and peers are installed", async () => {
    process.env[envOpenTelemetry] = "1";
    delete process.env[envOtelExporterOtlpEndpoint];

    const layer = await Effect.runPromise(resolveForstNodeRuntimeLayer());
    const runSpan = Effect.fn("Runtime.handleSyncCall")(function* () {
      yield* Effect.logInfo("otel resolve");
    });

    await Effect.runPromise(runSpan().pipe(Effect.provide(layer)));
  });
});

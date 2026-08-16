import * as NodeSdk from "@effect/opentelemetry/NodeSdk";
import { OTLPTraceExporter } from "@opentelemetry/exporter-trace-otlp-http";
import type { SpanProcessor } from "@opentelemetry/sdk-trace-base";
import {
  BatchSpanProcessor,
  ConsoleSpanExporter,
  SimpleSpanProcessor,
} from "@opentelemetry/sdk-trace-base";
import { Layer } from "effect";
import {
  makeForstNodeRuntimeLayer,
  type ForstNodeRuntimeLayerOptions,
} from "./layer.js";

/** Environment variable to opt in to OpenTelemetry when no OTLP endpoint is set. */
export const envOpenTelemetry = "FORST_NODE_OTEL";

/** Standard OTEL service name env var. */
export const envOtelServiceName = "OTEL_SERVICE_NAME";

/** Standard OTEL OTLP traces endpoint env var. */
export const envOtelExporterOtlpEndpoint = "OTEL_EXPORTER_OTLP_ENDPOINT";

/** Standard OTEL OTLP exporter headers env var (`key=value,key2=value2`). */
export const envOtelExporterOtlpHeaders = "OTEL_EXPORTER_OTLP_HEADERS";

const defaultServiceName = "forst-node-runtime";

/** Options for Forst OpenTelemetry layer helpers. */
export interface ForstNodeOpenTelemetryOptions {
  serviceName?: string;
  serviceVersion?: string;
  spanProcessor?: SpanProcessor | ReadonlyArray<SpanProcessor>;
}

/** Returns true when OTEL should activate (`FORST_NODE_OTEL=1` or OTLP endpoint set). */
export function openTelemetryEnabledFromEnv(): boolean {
  if (process.env[envOpenTelemetry]?.trim() === "1") {
    return true;
  }
  const endpoint = process.env[envOtelExporterOtlpEndpoint]?.trim();
  return endpoint !== undefined && endpoint.length > 0;
}

function resolveServiceName(options?: ForstNodeOpenTelemetryOptions): string {
  return (
    options?.serviceName?.trim() ||
    process.env[envOtelServiceName]?.trim() ||
    defaultServiceName
  );
}

function parseOtelExporterHeaders(
  raw: string | undefined
): Record<string, string> | undefined {
  if (!raw?.trim()) {
    return undefined;
  }
  const headers: Record<string, string> = {};
  for (const part of raw.split(",")) {
    const idx = part.indexOf("=");
    if (idx <= 0) {
      continue;
    }
    const key = part.slice(0, idx).trim();
    const value = part.slice(idx + 1).trim();
    if (key.length > 0) {
      headers[key] = value;
    }
  }
  return Object.keys(headers).length > 0 ? headers : undefined;
}

function defaultSpanProcessorFromEnv(): SpanProcessor {
  const endpoint = process.env[envOtelExporterOtlpEndpoint]?.trim();
  if (endpoint) {
    const headers = parseOtelExporterHeaders(
      process.env[envOtelExporterOtlpHeaders]
    );
    return new BatchSpanProcessor(
      new OTLPTraceExporter({
        url: endpoint,
        ...(headers !== undefined ? { headers } : {}),
      })
    );
  }
  return new SimpleSpanProcessor(new ConsoleSpanExporter());
}

function normalizeSpanProcessors(
  spanProcessor: SpanProcessor | ReadonlyArray<SpanProcessor>
): SpanProcessor | ReadonlyArray<SpanProcessor> {
  return spanProcessor;
}

/** Builds a NodeSdk layer with explicit or env-derived span processors. */
export function makeForstNodeOpenTelemetryLayer(
  options: ForstNodeOpenTelemetryOptions = {}
): Layer.Layer<unknown> {
  const serviceName = resolveServiceName(options);
  const spanProcessor =
    options.spanProcessor !== undefined
      ? normalizeSpanProcessors(options.spanProcessor)
      : defaultSpanProcessorFromEnv();

  return NodeSdk.layer(() => ({
    resource: {
      serviceName,
      ...(options.serviceVersion !== undefined
        ? { serviceVersion: options.serviceVersion }
        : {}),
    },
    spanProcessor,
  })) as Layer.Layer<unknown>;
}

/** NodeSdk layer using OTLP when `OTEL_EXPORTER_OTLP_ENDPOINT` is set, else console export. */
export function openTelemetryLayerFromEnv(
  options?: ForstNodeOpenTelemetryOptions
): Layer.Layer<unknown> {
  return makeForstNodeOpenTelemetryLayer(options);
}

/** Merge Forst stderr logging/tracing with an OpenTelemetry export layer. */
export function mergeForstNodeRuntimeWithOpenTelemetry(
  otelLayer: Layer.Layer<unknown>,
  forstOptions?: ForstNodeRuntimeLayerOptions
): Layer.Layer<never> {
  return Layer.mergeAll(
    makeForstNodeRuntimeLayer(forstOptions),
    otelLayer
  ) as Layer.Layer<never>;
}

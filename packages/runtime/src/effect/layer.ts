import * as fs from "node:fs";
import { FiberId, HashMap, Layer, Logger, LogLevel } from "effect";

/** Structured log annotation fields written by forst-runtime loggers. */
export type LogFields = Record<
  string,
  string | number | boolean | null | undefined
>;

/** Options for {@link makeForstRuntimeLayer}. */
export interface ForstRuntimeLayerOptions {
  /** When false, skip Logger.replace so a parent layer's logger wins on merge. Default true. */
  replaceLogger?: boolean;
}

/** Environment variable name for minimum log level. */
export const envLogLevel = "FORST_BRIDGE_LOG_LEVEL";
/** Environment variable name for log output format (`json` or pretty). */
export const envLogFormat = "FORST_BRIDGE_LOG_FORMAT";

/** Parses `FORST_BRIDGE_LOG_LEVEL` into an Effect log level. */
export function parseEnvLevel(raw: string | undefined): LogLevel.LogLevel {
  switch (raw?.trim().toLowerCase()) {
    case "trace":
      return LogLevel.Trace;
    case "debug":
      return LogLevel.Debug;
    case "warn":
    case "warning":
      return LogLevel.Warning;
    case "error":
      return LogLevel.Error;
    default:
      return LogLevel.Info;
  }
}

function annotationValue(value: unknown): string | number | boolean | null {
  if (
    value === null ||
    typeof value === "string" ||
    typeof value === "number" ||
    typeof value === "boolean"
  ) {
    return value;
  }
  return String(value);
}

function formatTimestamp(date: Date): string {
  const hours = `${date.getHours()}`.padStart(2, "0");
  const minutes = `${date.getMinutes()}`.padStart(2, "0");
  const seconds = `${date.getSeconds()}`.padStart(2, "0");
  const millis = `${date.getMilliseconds()}`.padStart(3, "0");
  return `${hours}:${minutes}:${seconds}.${millis}`;
}

function formatPrettyLogLine(
  logLevel: LogLevel.LogLevel,
  message: unknown,
  annotations: HashMap.HashMap<string, unknown>,
  date: Date,
  fiberId?: FiberId.FiberId
): string {
  const parts = Array.isArray(message) ? message : [message];
  const headline = parts.map(String).join(" ");
  const fiberSuffix =
    fiberId !== undefined ? ` (${FiberId.threadName(fiberId)})` : "";
  const lines = [
    `[${formatTimestamp(date)}] ${logLevel.label}${fiberSuffix}: ${headline}`,
  ];

  for (const [key, value] of HashMap.entries(annotations)) {
    lines.push(`  ${key}: ${annotationValue(value)}`);
  }

  return `${lines.join("\n")}\n`;
}

function writeStderrImpl(text: string): void {
  fs.writeSync(2, text);
}

let writeStderrHook: (text: string) => void = writeStderrImpl;

/** Test-only hook to capture stderr output without patching fs. */
export function setWriteStderrForTest(writer: (text: string) => void): () => void {
  const prev = writeStderrHook;
  writeStderrHook = writer;
  return () => {
    writeStderrHook = prev;
  };
}

function writeStderr(text: string): void {
  writeStderrHook(text);
}

/**
 * Human-readable stderr logger (default).
 * Writes to fd 2 directly.
 */
export const stderrPrettyLogger: Logger.Logger<unknown, void> = Logger.make(
  ({ logLevel, message, annotations, date, fiberId }) => {
    writeStderr(
      formatPrettyLogLine(logLevel, message, annotations, date, fiberId)
    );
  }
);

/** Structured JSON stderr logger; set `FORST_BRIDGE_LOG_FORMAT=json` or compose manually. */
export const stderrJsonLogger: Logger.Logger<unknown, void> = Logger.make(
  ({ logLevel, message, annotations, date }) => {
    const fields: LogFields = { component: "forst-runtime" };

    for (const [key, value] of HashMap.entries(annotations)) {
      fields[key] = annotationValue(value);
    }

    if (fields.event === undefined) {
      const parts = Array.isArray(message) ? message : [message];
      fields.event = parts.map(String).join(" ");
    }

    fields.level = logLevel.label;
    fields.ts = date.getTime();

    writeStderr(`${JSON.stringify(fields)}\n`);
  }
);

function resolveBaseLogger(): Logger.Logger<unknown, void> {
  if (process.env[envLogFormat]?.trim().toLowerCase() === "json") {
    return stderrJsonLogger;
  }
  return stderrPrettyLogger;
}

function buildRuntimeLogger(
  minLevel: LogLevel.LogLevel
): Logger.Logger<unknown, void> {
  const withSpans = Logger.withSpanAnnotations(resolveBaseLogger());
  if (LogLevel.lessThanEqual(minLevel, LogLevel.Debug)) {
    return Logger.zip(withSpans, Logger.tracerLogger);
  }
  return withSpans;
}

/** Builds the default Effect layer for forst-runtime logging and tracing. */
export function makeForstRuntimeLayer(
  options: ForstRuntimeLayerOptions = {}
): Layer.Layer<never> {
  const replaceLogger = options.replaceLogger ?? true;
  const minLevel = parseEnvLevel(process.env[envLogLevel]);
  const tracing = Layer.mergeAll(
    Layer.setTracerEnabled(true),
    Logger.minimumLogLevel(minLevel),
    Logger.remove(Logger.tracerLogger)
  );

  if (!replaceLogger) {
    return tracing;
  }

  return Layer.mergeAll(
    tracing,
    Logger.replace(Logger.defaultLogger, buildRuntimeLogger(minLevel))
  );
}

/** Default Effect layer: stderr logging, tracing, and `FORST_BRIDGE_LOG_LEVEL`. */
export const ForstRuntimeLayer: Layer.Layer<never, never, never> =
  makeForstRuntimeLayer();

/** @deprecated Use {@link ForstRuntimeLayer}. */
export const ForstNodeRuntimeLayer = ForstRuntimeLayer;

/** @deprecated Use {@link makeForstRuntimeLayer}. */
export const makeForstNodeRuntimeLayer = makeForstRuntimeLayer;

/** @deprecated Use {@link ForstRuntimeLayerOptions}. */
export type ForstNodeRuntimeLayerOptions = ForstRuntimeLayerOptions;

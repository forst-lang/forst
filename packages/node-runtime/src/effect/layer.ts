import * as fs from "node:fs";
import { HashMap, Layer, Logger, LogLevel } from "effect";

export type LogFields = Record<
  string,
  string | number | boolean | null | undefined
>;

export const envLogLevel = "FORST_NODE_LOG_LEVEL";
export const envLogFormat = "FORST_NODE_LOG_FORMAT";

export function parseEnvLevel(raw: string | undefined): LogLevel.LogLevel {
  switch (raw?.trim().toLowerCase()) {
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
  date: Date
): string {
  const parts = Array.isArray(message) ? message : [message];
  const headline = parts.map(String).join(" ");
  const lines = [`[${formatTimestamp(date)}] ${logLevel.label}: ${headline}`];

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
 * Writes to fd 2 directly so stdout stays reserved for RPC frames.
 */
export const stderrPrettyLogger = Logger.make(
  ({ logLevel, message, annotations, date }) => {
    writeStderr(formatPrettyLogLine(logLevel, message, annotations, date));
  }
);

/** Structured JSON stderr logger; set `FORST_NODE_LOG_FORMAT=json` or compose manually. */
export const stderrJsonLogger = Logger.make(
  ({ logLevel, message, annotations, date }) => {
    const fields: LogFields = { component: "node-runtime" };

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

function resolveDefaultLogger(): Logger.Logger<unknown, void> {
  if (process.env[envLogFormat]?.trim().toLowerCase() === "json") {
    return stderrJsonLogger;
  }
  return stderrPrettyLogger;
}

export function makeForstNodeRuntimeLayer(): Layer.Layer<never> {
  return Layer.mergeAll(
    Logger.replace(Logger.defaultLogger, resolveDefaultLogger()),
    Logger.minimumLogLevel(parseEnvLevel(process.env[envLogLevel]))
  );
}

export const ForstNodeRuntimeLayer = makeForstNodeRuntimeLayer();

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

/** Human-readable stderr logger (default). */
export const stderrPrettyLogger = Logger.prettyLogger({ stderr: true });

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

    process.stderr.write(`${JSON.stringify(fields)}\n`);
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

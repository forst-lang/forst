import { afterEach, describe, expect, test } from "bun:test";
import { Effect, LogLevel } from "effect";
import {
  envLogFormat,
  envLogLevel,
  parseEnvLevel,
  setWriteStderrForTest,
} from "../../src/effect/layer.js";
import { runTestEffect } from "../helpers/run-effect.js";

function captureStderr(run: () => void | Promise<void>): Promise<string> {
  const lines: string[] = [];
  const restore = setWriteStderrForTest((text) => {
    lines.push(text);
  });
  let outcome: void | Promise<void>;
  try {
    outcome = run();
  } catch (err) {
    restore();
    throw err;
  }
  return Promise.resolve(outcome)
    .finally(restore)
    .then(() => lines.join(""));
}

describe("parseEnvLevel", () => {
  test("maps trace to LogLevel.Trace", () => {
    expect(parseEnvLevel("trace")).toEqual(LogLevel.Trace);
    expect(parseEnvLevel("TRACE")).toEqual(LogLevel.Trace);
  });
});

describe("ForstNodeRuntimeLayer logging", () => {
  const prevLevel = process.env[envLogLevel];
  const prevFormat = process.env[envLogFormat];

  afterEach(() => {
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

  test("Effect.logDebug is silent at default info level", async () => {
    delete process.env[envLogLevel];
    delete process.env[envLogFormat];
    const output = await captureStderr(() =>
      runTestEffect(
        Effect.logDebug("ignored").pipe(
          Effect.annotateLogs({ event: "ignored", rpc_method: "forst.node/call" })
        )
      )
    );
    expect(output).toBe("");
  });

  test("Effect.logInfo emits pretty output with fiber id at default info level", async () => {
    delete process.env[envLogLevel];
    delete process.env[envLogFormat];
    const output = await captureStderr(() =>
      runTestEffect(
        Effect.logInfo("spawn").pipe(
          Effect.annotateLogs({ event: "spawn", pid: 42 })
        )
      )
    );
    expect(output).toContain("INFO (#");
    expect(output).toContain("spawn");
    expect(output).toContain("event: spawn");
    expect(output).toContain("pid: 42");
  });

  test("Effect.logDebug emits pretty output when FORST_NODE_LOG_LEVEL=debug", async () => {
    process.env[envLogLevel] = "debug";
    delete process.env[envLogFormat];
    const output = await captureStderr(() =>
      runTestEffect(
        Effect.logDebug("diagnostic").pipe(
          Effect.annotateLogs({
            event: "diagnostic",
            rpc_method: "forst.node/call",
          })
        )
      )
    );
    expect(output).toContain("DEBUG");
    expect(output).toContain("diagnostic");
    expect(output).toContain("rpc_method: forst.node/call");
  });

  test("Effect.fn span annotations appear in logs at debug level", async () => {
    process.env[envLogLevel] = "debug";
    delete process.env[envLogFormat];

    const runSpan = Effect.fn("Rpc.dispatch")(function* () {
      yield* Effect.annotateCurrentSpan("rpc_method", "forst.node/call");
      yield* Effect.annotateCurrentSpan("rpc_id", 7);
      yield* Effect.logInfo("dispatch complete");
    });

    const output = await captureStderr(() => runTestEffect(runSpan()));
    expect(output).not.toContain("rpc_recv");
    expect(output).toContain("dispatch complete");
    expect(output).toContain("effect.spanName: Rpc.dispatch");
    expect(output).toContain("effect.spanId:");
    expect(output).toContain("effect.traceId:");
  });

  test("Effect.fn span annotations appear in logs at trace level", async () => {
    process.env[envLogLevel] = "trace";
    delete process.env[envLogFormat];

    const runSpan = Effect.fn("Runtime.handleSyncCall")(function* () {
      yield* Effect.annotateCurrentSpan("module_id", "legacy/payment");
      yield* Effect.logInfo("call complete");
    });

    const output = await captureStderr(() => runTestEffect(runSpan()));
    expect(output).toContain("call complete");
    expect(output).toContain("effect.spanName: Runtime.handleSyncCall");
    expect(output).toContain("effect.spanId:");
  });

  test("Effect.logError emits at default info level", async () => {
    delete process.env[envLogLevel];
    delete process.env[envLogFormat];
    const output = await captureStderr(() =>
      runTestEffect(
        Effect.logError("fatal").pipe(
          Effect.annotateLogs({ event: "fatal", message: "boom" })
        )
      )
    );
    expect(output).toContain("ERROR");
    expect(output).toContain("fatal");
    expect(output).toContain("event: fatal");
  });

  test("FORST_NODE_LOG_FORMAT=json emits structured JSON with span annotations", async () => {
    process.env[envLogLevel] = "debug";
    process.env[envLogFormat] = "json";

    const runSpan = Effect.fn("Rpc.dispatch")(function* () {
      yield* Effect.annotateCurrentSpan("rpc_method", "forst.node/ping");
      yield* Effect.logInfo("done");
    });

    const output = await captureStderr(() => runTestEffect(runSpan()));
    const payload = JSON.parse(output.trim());
    expect(payload.component).toBe("node-runtime");
    expect(payload.event).toBe("done");
    expect(payload.level).toBe("INFO");
    expect(payload["effect.spanName"]).toBe("Rpc.dispatch");
    expect(payload["effect.spanId"]).toBeDefined();
    expect(payload["effect.traceId"]).toBeDefined();
  });

  test("pretty logs write to stderr only", async () => {
    delete process.env[envLogLevel];
    delete process.env[envLogFormat];
    const stdoutLines: string[] = [];
    const originalStdoutWrite = process.stdout.write.bind(process.stdout);
    process.stdout.write = ((chunk: string | Uint8Array) => {
      stdoutLines.push(
        typeof chunk === "string" ? chunk : Buffer.from(chunk).toString("utf8")
      );
      return true;
    }) as typeof process.stdout.write;

    try {
      await captureStderr(() =>
        runTestEffect(
          Effect.logInfo("spawn").pipe(
            Effect.annotateLogs({ event: "spawn", pid: 42 })
          )
        )
      );
    } finally {
      process.stdout.write = originalStdoutWrite;
    }

    expect(stdoutLines.join("")).toBe("");
  });
});

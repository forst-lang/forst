import { afterEach, describe, expect, test } from "bun:test";
import { Effect } from "effect";
import {
  envLogFormat,
  envLogLevel,
  setWriteStderrForTest,
} from "../../src/effect/layer.js";
import { runTestEffect } from "../helpers/run-effect.js";

function captureStderr(run: () => void | Promise<void>): Promise<string> {
  const lines: string[] = [];
  const restore = setWriteStderrForTest((text) => {
    lines.push(text);
  });
  return Promise.resolve(run())
    .finally(restore)
    .then(() => lines.join(""));
}

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
        Effect.logDebug("rpc_recv").pipe(
          Effect.annotateLogs({
            event: "rpc_recv",
            rpc_method: "forst.node/call",
            rpc_id: 1,
          })
        )
      )
    );
    expect(output).toBe("");
  });

  test("Effect.logInfo emits pretty output at default info level", async () => {
    delete process.env[envLogLevel];
    delete process.env[envLogFormat];
    const output = await captureStderr(() =>
      runTestEffect(
        Effect.logInfo("spawn").pipe(
          Effect.annotateLogs({ event: "spawn", pid: 42 })
        )
      )
    );
    expect(output).toContain("INFO");
    expect(output).toContain("spawn");
    expect(output).toContain("event: spawn");
    expect(output).toContain("pid: 42");
  });

  test("Effect.logDebug emits pretty output when FORST_NODE_LOG_LEVEL=debug", async () => {
    process.env[envLogLevel] = "debug";
    delete process.env[envLogFormat];
    const output = await captureStderr(() =>
      runTestEffect(
        Effect.logDebug("rpc_recv").pipe(
          Effect.annotateLogs({
            event: "rpc_recv",
            rpc_method: "forst.node/call",
            rpc_id: 7,
          })
        )
      )
    );
    expect(output).toContain("DEBUG");
    expect(output).toContain("rpc_recv");
    expect(output).toContain("rpc_method: forst.node/call");
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

  test("FORST_NODE_LOG_FORMAT=json emits structured JSON", async () => {
    delete process.env[envLogLevel];
    process.env[envLogFormat] = "json";
    const output = await captureStderr(() =>
      runTestEffect(
        Effect.logInfo("spawn").pipe(
          Effect.annotateLogs({ event: "spawn", pid: 42 })
        )
      )
    );
    const payload = JSON.parse(output.trim());
    expect(payload.component).toBe("node-runtime");
    expect(payload.event).toBe("spawn");
    expect(payload.pid).toBe(42);
    expect(payload.level).toBe("INFO");
  });

  test("pretty logs write to stderr only (stdout reserved for RPC)", async () => {
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

#!/usr/bin/env node

import { NodeRuntime } from "@effect/platform-node";
import { Effect } from "effect";
import { ForstNodeRuntimeLayer } from "../effect/layer.js";
import * as CliErrors from "./errors.js";
import { emitForstIndexV1Json } from "./emit-forst-index-v1.js";

export interface CliOptions {
  root: string;
  format: string;
  files: string[];
}

export function parseCliArgs(argv: string[]): CliOptions {
  let root = ".";
  let format = "forst-index-v1";
  let files: string[] = [];

  for (let i = 0; i < argv.length; i++) {
    const arg = argv[i];
    if (arg === "--root") {
      const value = argv[++i];
      if (!value) {
        throw CliErrors.cliMissingRootValue();
      }
      root = value;
      continue;
    }
    if (arg === "--format") {
      const value = argv[++i];
      if (!value) {
        throw CliErrors.cliMissingFormatValue();
      }
      format = value;
      continue;
    }
    if (arg === "--files") {
      const value = argv[++i];
      if (!value) {
        throw CliErrors.cliMissingFilesValue();
      }
      files = value
        .split(",")
        .map((file) => file.trim())
        .filter((file) => file.length > 0);
      continue;
    }
    if (arg === "--help" || arg === "-h") {
      printHelp();
      process.exit(0);
    }
    throw CliErrors.cliUnknownArgument(arg);
  }

  if (files.length === 0) {
    throw CliErrors.cliFilesRequired();
  }

  return { root, format, files };
}

function printHelp(): void {
  process.stdout.write(`Usage: forst-node-index --root DIR --format forst-index-v1 --files file1.ts,file2.ts

Options:
  --root DIR          Project boundary root (default: .)
  --format FORMAT     Index format (only forst-index-v1 supported)
  --files LIST        Comma-separated project-relative TS file paths
  -h, --help          Show this help
`);
}

export const runCliEffect = Effect.fn("Indexer.runCli")(function* (argv: string[]) {
  try {
    const options = parseCliArgs(argv);

    if (options.format !== "forst-index-v1") {
      throw CliErrors.cliUnsupportedFormat(options.format);
    }

    yield* Effect.annotateCurrentSpan("root", options.root);
    yield* Effect.annotateCurrentSpan("file_count", options.files.length);

    const json = yield* Effect.sync(() =>
      emitForstIndexV1Json({
        root: options.root,
        files: options.files,
      })
    );
    process.stdout.write(`${json}\n`);
    return 0;
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    yield* Effect.annotateCurrentSpan("message", message);
    yield* Effect.logError("cli_error").pipe(
      Effect.annotateLogs({ event: "cli_error", message })
    );
    process.stderr.write(`forst-node-index: ${message}\n`);
    return 1;
  }
});

export function runCli(argv: string[]): number {
  return Effect.runSync(
    runCliEffect(argv).pipe(Effect.provide(ForstNodeRuntimeLayer))
  );
}

/** Async wrapper for programmatic callers. */
export async function runIndexerCli(argv: string[]): Promise<number> {
  return Effect.runPromise(
    runCliEffect(argv).pipe(Effect.provide(ForstNodeRuntimeLayer))
  );
}

const isDirectExecution =
  typeof process.argv[1] === "string" &&
  (process.argv[1].endsWith("/indexer/cli.js") ||
    process.argv[1].endsWith("/indexer/cli.ts"));

if (isDirectExecution) {
  NodeRuntime.runMain(
    runCliEffect(process.argv.slice(2)).pipe(
      Effect.flatMap((code) =>
        Effect.sync(() => {
          process.exit(code);
        })
      ),
      Effect.provide(ForstNodeRuntimeLayer)
    ),
    { disablePrettyLogger: true }
  );
}

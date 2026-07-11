#!/usr/bin/env node

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
        throw new Error("missing value for --root");
      }
      root = value;
      continue;
    }
    if (arg === "--format") {
      const value = argv[++i];
      if (!value) {
        throw new Error("missing value for --format");
      }
      format = value;
      continue;
    }
    if (arg === "--files") {
      const value = argv[++i];
      if (!value) {
        throw new Error("missing value for --files");
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
    throw new Error(`unknown argument: ${arg}`);
  }

  if (files.length === 0) {
    throw new Error("at least one file is required via --files");
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

export function runCli(argv: string[]): number {
  try {
    const options = parseCliArgs(argv);

    if (options.format !== "forst-index-v1") {
      throw new Error(`unsupported format: ${options.format}`);
    }

    const json = emitForstIndexV1Json({
      root: options.root,
      files: options.files,
    });
    process.stdout.write(`${json}\n`);
    return 0;
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    process.stderr.write(`forst-node-index: ${message}\n`);
    return 1;
  }
}

/** Async wrapper for programmatic callers. */
export async function runIndexerCli(argv: string[]): Promise<number> {
  return runCli(argv);
}

const isDirectExecution =
  typeof process.argv[1] === "string" &&
  (process.argv[1].endsWith("/indexer/cli.js") ||
    process.argv[1].endsWith("/indexer/cli.ts"));

if (isDirectExecution) {
  process.exit(runCli(process.argv.slice(2)));
}

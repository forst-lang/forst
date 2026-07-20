#!/usr/bin/env node
/**
 * CLI shim that forwards argv to the resolved native `forst` binary.
 *
 * Handles `--info`, `--version`, and `--go-buildinfo` locally; all other flags
 * are passed through after downloading or locating the platform artifact.
 *
 * @module forst_cli_bin
 */
import { spawn } from "node:child_process";
import {
  printForstCliInfo,
  printForstCliVersion,
  printForstGoBuildInfo,
} from "./cli-info.js";
import {
  FORST_CLI_INFO_FLAG,
  FORST_CLI_GO_BUILDINFO_FLAG,
  FORST_CLI_VERSION_FLAGS,
} from "./constants.js";
import { buildForstSpawnEnv } from "./spawn-env.js";

const argv = process.argv.slice(2);

/** Spawns the resolved native `forst` binary with the current argv, forwarding SIGINT/SIGTERM and exit status. */
export async function runForstCli(): Promise<number> {
  const { bin, env } = await buildForstSpawnEnv();
  const child = spawn(bin, process.argv.slice(2), { stdio: "inherit", env });
  const forward = (sig: NodeJS.Signals) => {
    child.kill(sig);
  };
  process.on("SIGINT", forward);
  process.on("SIGTERM", forward);
  return new Promise<number>((resolve, reject) => {
    child.on("error", reject);
    child.on("close", (code, signal) => {
      process.removeListener("SIGINT", forward);
      process.removeListener("SIGTERM", forward);
      if (signal) {
        resolve(1);
        return;
      }
      resolve(code ?? 0);
    });
  });
}

const versionFlags = FORST_CLI_VERSION_FLAGS as readonly string[];

if (argv.length === 1 && argv[0] === FORST_CLI_INFO_FLAG) {
  printForstCliInfo()
    .then(() => process.exit(0))
    .catch((err) => {
      console.error(err);
      process.exit(1);
    });
} else if (argv.length === 1 && versionFlags.includes(argv[0])) {
  printForstCliVersion()
    .then(() => process.exit(0))
    .catch((err) => {
      console.error(err);
      process.exit(1);
    });
} else if (argv.length === 1 && argv[0] === FORST_CLI_GO_BUILDINFO_FLAG) {
  printForstGoBuildInfo()
    .then(() => process.exit(0))
    .catch((err) => {
      console.error(err);
      process.exit(1);
    });
} else {
  runForstCli()
    .then((code) => process.exit(code))
    .catch((err) => {
      console.error(err);
      process.exit(1);
    });
}

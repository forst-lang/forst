#!/usr/bin/env node
import { spawn } from "node:child_process";
import { printForstCliInfo, printForstCliVersion } from "./cli-info.js";
import { FORST_CLI_INFO_FLAG, FORST_CLI_VERSION_FLAGS } from "./constants.js";
import { resolveForstBinary } from "./resolve.js";

const argv = process.argv.slice(2);

export async function runForstCli(): Promise<number> {
  const bin = await resolveForstBinary();
  const child = spawn(bin, process.argv.slice(2), { stdio: "inherit" });
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
} else {
  runForstCli()
    .then((code) => process.exit(code))
    .catch((err) => {
      console.error(err);
      process.exit(1);
    });
}

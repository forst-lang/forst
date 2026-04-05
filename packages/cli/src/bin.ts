#!/usr/bin/env node
import { spawn } from "node:child_process";
import { resolveForstBinary } from "./resolve.js";

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

runForstCli()
  .then((code) => process.exit(code))
  .catch((err) => {
    console.error(err);
    process.exit(1);
  });

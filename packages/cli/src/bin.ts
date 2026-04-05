#!/usr/bin/env node
import { spawn } from "node:child_process";
import { resolveForstBinary } from "./resolve.js";

export async function runForstCli(): Promise<void> {
  const bin = await resolveForstBinary();
  const child = spawn(bin, process.argv.slice(2), { stdio: "inherit" });
  const forward = (sig: NodeJS.Signals) => {
    child.kill(sig);
  };
  process.on("SIGINT", forward);
  process.on("SIGTERM", forward);
  await new Promise<void>((resolve, reject) => {
    child.on("error", reject);
    child.on("close", (code, signal) => {
      process.removeListener("SIGINT", forward);
      process.removeListener("SIGTERM", forward);
      if (signal) {
        process.exit(1);
      }
      process.exit(code ?? 0);
      resolve();
    });
  });
}

runForstCli().catch((err) => {
  console.error(err);
  process.exit(1);
});

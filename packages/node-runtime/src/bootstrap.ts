#!/usr/bin/env node
import { log } from "./logging/logger.js";
import { startRpcServer } from "./rpc/server.js";

export async function main(): Promise<void> {
  log("spawn", { pid: process.pid });
  await startRpcServer(process.stdin, process.stdout, {
    exitProcessOnShutdown: true,
  });
}

const isDirectExecution =
  typeof process.argv[1] === "string" &&
  (process.argv[1].endsWith("/bootstrap.js") ||
    process.argv[1].endsWith("/bootstrap.ts"));

if (isDirectExecution) {
  main().catch((err) => {
    log("fatal", { message: err instanceof Error ? err.message : String(err) });
    process.exit(1);
  });
}

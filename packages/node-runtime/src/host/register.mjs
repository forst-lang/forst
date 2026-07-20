#!/usr/bin/env node
/* @ts-self-types="./register.d.ts" */
import { Effect } from "effect";
import { resolve } from "node:path";
import { pathToFileURL } from "node:url";
import { ForstNodeRuntimeLayer } from "../effect/layer.js";
import { signalForstAppReady, startForstNodeHost } from "../host.js";

const appReadyModule = process.env.FORST_NODE_APP_READY_MODULE?.trim();

await Effect.runPromise(
  startForstNodeHost({ deferAppReady: Boolean(appReadyModule) }).pipe(
    Effect.provide(ForstNodeRuntimeLayer)
  )
);

if (appReadyModule) {
  const url = pathToFileURL(resolve(appReadyModule)).href;
  await import(url);
  await Effect.runPromise(
    signalForstAppReady().pipe(Effect.provide(ForstNodeRuntimeLayer))
  );
}

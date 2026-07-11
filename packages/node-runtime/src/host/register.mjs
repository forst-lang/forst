import { resolve } from "node:path";
import { pathToFileURL } from "node:url";
import { signalForstAppReady, startForstNodeHost } from "../host.js";

const appReadyModule = process.env.FORST_NODE_APP_READY_MODULE?.trim();

await startForstNodeHost({ deferAppReady: Boolean(appReadyModule) });

if (appReadyModule) {
  const url = pathToFileURL(resolve(appReadyModule)).href;
  await import(url);
  await signalForstAppReady();
}

"use strict";

// Deprecated: use register.mjs with NODE_OPTIONS --import instead.
process.stderr.write(
  `${JSON.stringify({
    component: "node-runtime",
    event: "host_register_deprecated",
    message:
      "register.cjs is deprecated; use @forst/node-runtime/host/register.mjs with NODE_OPTIONS --import",
  })}\n`
);

void import("../host.js")
  .then(async ({ startForstNodeHost, signalForstAppReady }) => {
    const appReadyModule = process.env.FORST_NODE_APP_READY_MODULE?.trim();
    await startForstNodeHost({ deferAppReady: true });
    if (appReadyModule) {
      const { resolve } = await import("node:path");
      const { pathToFileURL } = await import("node:url");
      await import(pathToFileURL(resolve(appReadyModule)).href);
      await signalForstAppReady();
    }
  })
  .catch((err) => {
    process.stderr.write(
      `${JSON.stringify({
        component: "node-runtime",
        event: "host_register_fatal",
        message: err instanceof Error ? err.message : String(err),
      })}\n`
    );
  });

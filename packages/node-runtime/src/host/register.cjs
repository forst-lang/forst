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

void Promise.all([import("../host.js"), import("effect"), import("../effect/layer.js")])
  .then(async ([hostMod, effectMod, layerMod]) => {
    const { startForstNodeHost, signalForstAppReady } = hostMod;
    const { Effect } = effectMod;
    const { ForstNodeRuntimeLayer } = layerMod;
    const appReadyModule = process.env.FORST_NODE_APP_READY_MODULE?.trim();

    await Effect.runPromise(
      startForstNodeHost({ deferAppReady: Boolean(appReadyModule) }).pipe(
        Effect.provide(ForstNodeRuntimeLayer)
      )
    );

    if (appReadyModule) {
      const { resolve } = await import("node:path");
      const { pathToFileURL } = await import("node:url");
      await import(pathToFileURL(resolve(appReadyModule)).href);
      await Effect.runPromise(
        signalForstAppReady().pipe(Effect.provide(ForstNodeRuntimeLayer))
      );
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

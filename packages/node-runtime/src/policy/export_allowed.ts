import { Effect } from "effect";
import type { ForstNodeExportKind, ForstNodeManifestExportV1 } from "../manifest/schema.js";
import { JsonRpcError } from "../rpc/errors.js";
import type { ManifestIndex } from "./manifest.js";
import { assertExportAllowed } from "./manifest.js";

/**
 * Policy check as Effect. Error normalization preserves `JsonRpcError` instances;
 * all other failures become `JsonRpcError(-32000, String(err))`.
 */
export const assertExportAllowedEffect = Effect.fn("Policy.assertExportAllowed")(
  function* (
    index: ManifestIndex,
    moduleId: string,
    exportName: string,
    expectedKind?: ForstNodeExportKind
  ) {
    yield* Effect.annotateCurrentSpan("module_id", moduleId);
    yield* Effect.annotateCurrentSpan("export_name", exportName);
    if (expectedKind !== undefined) {
      yield* Effect.annotateCurrentSpan("expected_kind", expectedKind);
    }
    return yield* Effect.try({
      try: () => assertExportAllowed(index, moduleId, exportName, expectedKind),
      catch: (err) =>
        err instanceof JsonRpcError
          ? err
          : new JsonRpcError(-32000, String(err)),
    });
  }
);
import {
  ForstNodeSchemaValidationError,
  parseForstNodeManifestV1,
  type ForstNodeExportKind,
  type ForstNodeManifestExportV1,
  type ForstNodeManifestV1,
} from "../manifest/schema.js";
import * as Errors from "../rpc/errors.js";

export interface ManifestIndex {
  boundaryRoot: string;
  exports: ForstNodeManifestExportV1[];
  byKey: Map<string, ForstNodeManifestExportV1>;
}

function exportKey(moduleId: string, name: string): string {
  return `${moduleId}\0${name}`;
}

export function validateManifest(manifest: unknown): ForstNodeManifestV1 {
  try {
    return parseForstNodeManifestV1(manifest);
  } catch (err) {
    if (err instanceof ForstNodeSchemaValidationError) {
      throw Errors.invalidParams(err.message);
    }
    throw err;
  }
}

export function buildManifestIndex(manifest: ForstNodeManifestV1): ManifestIndex {
  const byKey = new Map<string, ForstNodeManifestExportV1>();
  for (const entry of manifest.exports) {
    byKey.set(exportKey(entry.moduleId, entry.name), entry);
  }
  return {
    boundaryRoot: manifest.boundaryRoot,
    exports: manifest.exports,
    byKey,
  };
}

export function assertExportAllowed(
  index: ManifestIndex,
  moduleId: string,
  exportName: string,
  expectedKind?: ForstNodeExportKind
): ForstNodeManifestExportV1 {
  const entry = index.byKey.get(exportKey(moduleId, exportName));
  if (!entry) {
    throw Errors.forbidden("export not in manifest allowlist", {
      moduleId,
      exportName,
    });
  }

  if (expectedKind !== undefined && entry.kind !== expectedKind) {
    throw Errors.forbidden("export kind mismatch for RPC method", {
      moduleId,
      exportName,
      expectedKind,
      actualKind: entry.kind,
    });
  }

  return entry;
}

/** Compile-time execution allowlist embedded in Go binaries. */
export const FORST_NODE_MANIFEST_V1_VERSION = 1 as const;

/** TypeScript index format consumed by the Forst compiler. */
export const FORST_INDEX_V1_FORMAT = "forst-index-v1" as const;

export type ForstNodeExportKind =
  | "function"
  | "asyncFunction"
  | "generator"
  | "asyncGenerator";

export const FORST_NODE_EXPORT_KINDS: readonly ForstNodeExportKind[] = [
  "function",
  "asyncFunction",
  "generator",
  "asyncGenerator",
] as const;

export interface ForstNodeManifestExportV1 {
  moduleId: string;
  name: string;
  kind: ForstNodeExportKind;
}

export interface ForstNodeManifestV1 {
  version: typeof FORST_NODE_MANIFEST_V1_VERSION;
  boundaryRoot: string;
  exports: ForstNodeManifestExportV1[];
}

export type ForstIndexTypeKind =
  | "string"
  | "number"
  | "boolean"
  | "void"
  | "bytes"
  | "object"
  | "array"
  | "union"
  | "unknown";

export interface ForstIndexTypeNode {
  kind?: ForstIndexTypeKind;
  fields?: Record<string, ForstIndexTypeNode>;
  element?: ForstIndexTypeNode;
  members?: ForstIndexTypeNode[];
  /** When true, `element` describes the binary element type (e.g. bytes). */
  $binary?: boolean;
}

export interface ForstIndexParameterV1 {
  name: string;
  type: ForstIndexTypeNode;
}

/** Source span for go-to-definition (1-based line, 0-based column). */
export interface ForstIndexSourceLocationV1 {
  /** Project-relative POSIX path; omitted when same as the indexed module. */
  file?: string;
  line: number;
  column: number;
  endLine?: number;
  endColumn?: number;
}

export interface ForstIndexExportV1 {
  name: string;
  kind: ForstNodeExportKind;
  parameters: ForstIndexParameterV1[];
  returnType?: ForstIndexTypeNode;
  yieldType?: ForstIndexTypeNode;
  definition?: ForstIndexSourceLocationV1;
}

export interface ForstIndexModuleV1 {
  moduleId: string;
  exports: ForstIndexExportV1[];
}

export class ForstNodeSchemaValidationError extends Error {
  readonly path: string;

  constructor(path: string, message: string) {
    super(`${path}: ${message}`);
    this.name = "ForstNodeSchemaValidationError";
    this.path = path;
  }
}

export function isForstNodeExportKind(value: unknown): value is ForstNodeExportKind {
  return (
    typeof value === "string" &&
    (FORST_NODE_EXPORT_KINDS as readonly string[]).includes(value)
  );
}

export function isValidModuleId(moduleId: string): boolean {
  if (moduleId.length === 0) {
    return false;
  }
  if (moduleId.startsWith("/") || /^[A-Za-z]:[\\/]/.test(moduleId)) {
    return false;
  }
  if (moduleId.includes("..")) {
    return false;
  }
  if (moduleId.includes("\\")) {
    return false;
  }
  return true;
}

function assertRecord(value: unknown, path: string): Record<string, unknown> {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    throw new ForstNodeSchemaValidationError(path, "expected object");
  }
  return value as Record<string, unknown>;
}

function assertString(value: unknown, path: string): string {
  if (typeof value !== "string" || value.length === 0) {
    throw new ForstNodeSchemaValidationError(path, "expected non-empty string");
  }
  return value;
}

function parseExportKind(value: unknown, path: string): ForstNodeExportKind {
  const kind = assertString(value, path);
  if (!isForstNodeExportKind(kind)) {
    throw new ForstNodeSchemaValidationError(
      path,
      `expected one of ${FORST_NODE_EXPORT_KINDS.join(", ")}`,
    );
  }
  return kind;
}

function parseManifestExport(
  value: unknown,
  path: string,
): ForstNodeManifestExportV1 {
  const record = assertRecord(value, path);
  const moduleId = assertString(record.moduleId, `${path}.moduleId`);
  if (!isValidModuleId(moduleId)) {
    throw new ForstNodeSchemaValidationError(
      `${path}.moduleId`,
      "must be a project-relative POSIX path without .. or absolute segments",
    );
  }
  return {
    moduleId,
    name: assertString(record.name, `${path}.name`),
    kind: parseExportKind(record.kind, `${path}.kind`),
  };
}

export function parseForstNodeManifestV1(value: unknown): ForstNodeManifestV1 {
  const record = assertRecord(value, "manifest");
  const version = record.version;
  if (version !== FORST_NODE_MANIFEST_V1_VERSION) {
    throw new ForstNodeSchemaValidationError(
      "manifest.version",
      `expected ${FORST_NODE_MANIFEST_V1_VERSION}`,
    );
  }
  const boundaryRoot = assertString(record.boundaryRoot, "manifest.boundaryRoot");
  if (!Array.isArray(record.exports)) {
    throw new ForstNodeSchemaValidationError(
      "manifest.exports",
      "expected array",
    );
  }
  const exportEntries = record.exports.map((entry, index) =>
    parseManifestExport(entry, `manifest.exports[${index}]`),
  );
  return {
    version: FORST_NODE_MANIFEST_V1_VERSION,
    boundaryRoot,
    exports: exportEntries,
  };
}

function parseIndexTypeNode(value: unknown, path: string): ForstIndexTypeNode {
  const record = assertRecord(value, path);

  if (record.$binary === true) {
    const node: ForstIndexTypeNode = { kind: "object", $binary: true };
    if (record.element !== undefined) {
      node.element = parseIndexTypeNode(record.element, `${path}.element`);
    }
    return node;
  }

  const kind = assertString(record.kind, `${path}.kind`) as ForstIndexTypeKind;
  const node: ForstIndexTypeNode = { kind };

  if (record.$binary !== undefined) {
    if (typeof record.$binary !== "boolean") {
      throw new ForstNodeSchemaValidationError(
        `${path}.$binary`,
        "expected boolean",
      );
    }
    node.$binary = record.$binary;
  }

  if (record.element !== undefined) {
    node.element = parseIndexTypeNode(record.element, `${path}.element`);
  }

  if (record.fields !== undefined) {
    const fieldsRecord = assertRecord(record.fields, `${path}.fields`);
    node.fields = {};
    for (const [fieldName, fieldType] of Object.entries(fieldsRecord)) {
      node.fields[fieldName] = parseIndexTypeNode(
        fieldType,
        `${path}.fields.${fieldName}`,
      );
    }
  }

  if (record.members !== undefined) {
    if (!Array.isArray(record.members)) {
      throw new ForstNodeSchemaValidationError(
        `${path}.members`,
        "expected array",
      );
    }
    node.members = record.members.map((member, index) =>
      parseIndexTypeNode(member, `${path}.members[${index}]`),
    );
  }

  return node;
}

function parseIndexParameter(value: unknown, path: string): ForstIndexParameterV1 {
  const record = assertRecord(value, path);
  return {
    name: assertString(record.name, `${path}.name`),
    type: parseIndexTypeNode(record.type, `${path}.type`),
  };
}

function assertNonNegativeInt(value: unknown, path: string): number {
  if (typeof value !== "number" || !Number.isInteger(value) || value < 0) {
    throw new ForstNodeSchemaValidationError(path, "expected non-negative integer");
  }
  return value;
}

function assertPositiveInt(value: unknown, path: string): number {
  if (typeof value !== "number" || !Number.isInteger(value) || value < 1) {
    throw new ForstNodeSchemaValidationError(path, "expected positive integer");
  }
  return value;
}

function parseIndexSourceLocation(
  value: unknown,
  path: string,
): ForstIndexSourceLocationV1 {
  const record = assertRecord(value, path);
  const parsed: ForstIndexSourceLocationV1 = {
    line: assertPositiveInt(record.line, `${path}.line`),
    column: assertNonNegativeInt(record.column, `${path}.column`),
  };
  if (record.file !== undefined) {
    const file = assertString(record.file, `${path}.file`);
    if (!isValidModuleId(file)) {
      throw new ForstNodeSchemaValidationError(
        `${path}.file`,
        "must be a project-relative POSIX path without .. or absolute segments",
      );
    }
    parsed.file = file;
  }
  if (record.endLine !== undefined) {
    parsed.endLine = assertPositiveInt(record.endLine, `${path}.endLine`);
  }
  if (record.endColumn !== undefined) {
    parsed.endColumn = assertNonNegativeInt(record.endColumn, `${path}.endColumn`);
  }
  return parsed;
}

function parseIndexExport(value: unknown, path: string): ForstIndexExportV1 {
  const record = assertRecord(value, path);
  if (!Array.isArray(record.parameters)) {
    throw new ForstNodeSchemaValidationError(
      `${path}.parameters`,
      "expected array",
    );
  }
  const parameters = record.parameters.map((parameter, index) =>
    parseIndexParameter(parameter, `${path}.parameters[${index}]`),
  );
  const parsed: ForstIndexExportV1 = {
    name: assertString(record.name, `${path}.name`),
    kind: parseExportKind(record.kind, `${path}.kind`),
    parameters,
  };
  if (record.returnType !== undefined) {
    parsed.returnType = parseIndexTypeNode(
      record.returnType,
      `${path}.returnType`,
    );
  }
  if (record.yieldType !== undefined) {
    parsed.yieldType = parseIndexTypeNode(record.yieldType, `${path}.yieldType`);
  }
  if (record.definition !== undefined) {
    parsed.definition = parseIndexSourceLocation(record.definition, `${path}.definition`);
  }
  return parsed;
}

export function parseForstIndexModuleV1(value: unknown): ForstIndexModuleV1 {
  const record = assertRecord(value, "index");
  const moduleId = assertString(record.moduleId, "index.moduleId");
  if (!isValidModuleId(moduleId)) {
    throw new ForstNodeSchemaValidationError(
      "index.moduleId",
      "must be a project-relative POSIX path without .. or absolute segments",
    );
  }
  if (!Array.isArray(record.exports)) {
    throw new ForstNodeSchemaValidationError("index.exports", "expected array");
  }
  const exportEntries = record.exports.map((entry, index) =>
    parseIndexExport(entry, `index.exports[${index}]`),
  );
  return { moduleId, exports: exportEntries };
}

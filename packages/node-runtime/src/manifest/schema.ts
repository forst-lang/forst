/** Compile-time execution allowlist embedded in Go binaries. */
export const FORST_NODE_MANIFEST_V1_VERSION = 1 as const;

/** TypeScript index format consumed by the Forst compiler. */
export const FORST_INDEX_V1_FORMAT = "forst-index-v1" as const;

/** Callable export kind recorded in manifest and index artifacts. */
export type ForstNodeExportKind =
  | "function"
  | "asyncFunction"
  | "generator"
  | "asyncGenerator";

/** All supported {@link ForstNodeExportKind} values. */
export const FORST_NODE_EXPORT_KINDS: readonly ForstNodeExportKind[] = [
  "function",
  "asyncFunction",
  "generator",
  "asyncGenerator",
] as const;

/** Single export entry in a Forst Node manifest v1 allowlist. */
export interface ForstNodeManifestExportV1 {
  /** Project-relative POSIX module path. */
  moduleId: string;
  /** Exported symbol name within the module. */
  name: string;
  /** Callable kind of the export. */
  kind: ForstNodeExportKind;
}

/** Runtime execution allowlist shipped from the Go sidecar. */
export interface ForstNodeManifestV1 {
  /** Manifest schema version; must be {@link FORST_NODE_MANIFEST_V1_VERSION}. */
  version: typeof FORST_NODE_MANIFEST_V1_VERSION;
  /** Absolute project boundary root used for module resolution. */
  boundaryRoot: string;
  /** Callable exports permitted at runtime. */
  exports: ForstNodeManifestExportV1[];
}

/** Primitive and composite kinds in a Forst index type tree. */
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

/** Recursive type node in a Forst index export signature. */
export interface ForstIndexTypeNode {
  /** Type kind discriminator when not inferred from shape. */
  kind?: ForstIndexTypeKind;
  /** Object field types keyed by name. */
  fields?: Record<string, ForstIndexTypeNode>;
  /** Element type for arrays and binary buffers. */
  element?: ForstIndexTypeNode;
  /** Union member types. */
  members?: ForstIndexTypeNode[];
  /** When true, `element` describes the binary element type (e.g. bytes). */
  $binary?: boolean;
}

/** Function parameter in a Forst index export signature. */
export interface ForstIndexParameterV1 {
  /** Parameter name. */
  name: string;
  /** Parameter type tree. */
  type: ForstIndexTypeNode;
}

/** Source span for go-to-definition (1-based line, 0-based column). */
export interface ForstIndexSourceLocationV1 {
  /** Project-relative POSIX path; omitted when same as the indexed module. */
  file?: string;
  /** 1-based start line. */
  line: number;
  /** 0-based start column. */
  column: number;
  /** 1-based end line. */
  endLine?: number;
  /** 0-based end column. */
  endColumn?: number;
}

/** Indexed callable export within a module. */
export interface ForstIndexExportV1 {
  /** Export name. */
  name: string;
  /** Callable kind. */
  kind: ForstNodeExportKind;
  /** Function parameters. */
  parameters: ForstIndexParameterV1[];
  /** Return type for functions and async functions. */
  returnType?: ForstIndexTypeNode;
  /** Yielded element type for generators. */
  yieldType?: ForstIndexTypeNode;
  /** Source span for go-to-definition (1-based line, 0-based column). */
  definition?: ForstIndexSourceLocationV1;
}

/** Indexed module section in a Forst index v1 document. */
export interface ForstIndexModuleV1 {
  /** Project-relative POSIX module path. */
  moduleId: string;
  /** Callable exports discovered in the module. */
  exports: ForstIndexExportV1[];
}

/** Thrown when manifest or index JSON fails schema validation. */
export class ForstNodeSchemaValidationError extends Error {
  /** JSON path to the invalid value. */
  readonly path: string;

  /**
   * Creates a schema validation error.
   * @param path JSON path to the invalid value.
   * @param message Validation error message.
   */
  constructor(path: string, message: string) {
    super(`${path}: ${message}`);
    this.name = "ForstNodeSchemaValidationError";
    this.path = path;
  }
}

/** Type guard for {@link ForstNodeExportKind}. */
export function isForstNodeExportKind(value: unknown): value is ForstNodeExportKind {
  return (
    typeof value === "string" &&
    (FORST_NODE_EXPORT_KINDS as readonly string[]).includes(value)
  );
}

/** Returns true when `moduleId` is a safe project-relative POSIX path. */
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

/** Parses and validates a Forst Node manifest v1 JSON value. */
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

/** Parses and validates a single Forst index module v1 JSON value. */
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

import path from "node:path";
import { Node, type FunctionDeclaration, type SourceFile, type Type } from "ts-morph";
import type {
  ForstIndexExportV1,
  ForstIndexModuleV1,
  ForstIndexParameterV1,
  ForstIndexTypeNode,
  ForstNodeExportKind,
} from "../manifest/schema.js";
import { FORST_INDEX_V1_FORMAT } from "../manifest/schema.js";
import {
  addSourceFiles,
  createIndexerProject,
  toPosixModuleId,
} from "./project.js";

export interface ForstIndexV1 {
  format: typeof FORST_INDEX_V1_FORMAT;
  modules: ForstIndexModuleV1[];
}

export interface EmitForstIndexV1Options {
  root: string;
  files: string[];
}

const GENERATOR_LIKE = new Set([
  "Generator",
  "AsyncGenerator",
  "IterableIterator",
  "AsyncIterableIterator",
]);

const BYTES_LIKE = new Set(["Buffer", "Uint8Array", "ArrayBuffer"]);

function typeName(type: Type): string {
  const symbol = type.getAliasSymbol() ?? type.getSymbol();
  return symbol?.getName() ?? type.getText();
}

function isPromiseType(type: Type | undefined): type is Type {
  if (!type) {
    return false;
  }
  const name = typeName(type);
  if (name === "Promise") {
    return true;
  }
  const target = type.getTargetType();
  return target !== undefined && target !== type && isPromiseType(target);
}

function unwrapPromise(type: Type): Type {
  if (!isPromiseType(type)) {
    return type;
  }
  const args = type.getTypeArguments();
  if (args.length > 0 && args[0]) {
    return args[0];
  }
  const target = type.getTargetType();
  if (target && target !== type) {
    return unwrapPromise(target);
  }
  return type;
}

function isGeneratorLikeType(type: Type): boolean {
  const name = typeName(type);
  if (GENERATOR_LIKE.has(name)) {
    return true;
  }
  const target = type.getTargetType();
  return target !== undefined && target !== type && isGeneratorLikeType(target);
}

function generatorYieldType(type: Type): Type | undefined {
  if (!isGeneratorLikeType(type)) {
    return undefined;
  }
  const args = type.getTypeArguments();
  if (args.length > 0) {
    return args[0];
  }
  const target = type.getTargetType();
  if (target !== undefined && target !== type) {
    return generatorYieldType(target);
  }
  return undefined;
}

function generatorReturnType(type: Type): Type | undefined {
  if (!isGeneratorLikeType(type)) {
    return undefined;
  }
  const args = type.getTypeArguments();
  if (args.length >= 2) {
    return args[1];
  }
  const target = type.getTargetType();
  if (target !== undefined && target !== type) {
    return generatorReturnType(target);
  }
  return undefined;
}

function isBytesType(type: Type): boolean {
  return BYTES_LIKE.has(typeName(type));
}

function mapType(type: Type): ForstIndexTypeNode {
  if (type.isString() || type.isStringLiteral()) {
    return { kind: "string" };
  }
  if (type.isNumber() || type.isNumberLiteral()) {
    return { kind: "number" };
  }
  if (type.isBoolean() || type.isBooleanLiteral()) {
    return { kind: "boolean" };
  }
  if (type.isUndefined()) {
    return { kind: "void" };
  }
  if (type.isNull()) {
    return { kind: "unknown" };
  }
  if (type.isVoid()) {
    return { kind: "void" };
  }
  if (isBytesType(type)) {
    return { kind: "bytes" };
  }
  if (typeName(type) === "Date") {
    return { kind: "string" };
  }

  if (type.isArray()) {
    const element = type.getArrayElementType();
    return {
      kind: "array",
      element: element ? mapType(element) : { kind: "unknown" },
    };
  }

  if (type.isUnion()) {
    const members = type.getUnionTypes().map(mapType);
    return { kind: "union", members };
  }

  if (type.isObject()) {
    const fields: Record<string, ForstIndexTypeNode> = {};
    for (const prop of type.getProperties()) {
      const propName = prop.getName();
      if (propName.startsWith("__")) {
        continue;
      }
      const declarations = prop.getDeclarations();
      if (declarations.length === 0) {
        continue;
      }
      const propType = prop.getTypeAtLocation(declarations[0]);
      fields[propName] = mapType(propType);
    }

    if (Object.keys(fields).length > 0) {
      return { kind: "object", fields };
    }
  }

  return { kind: "unknown" };
}

function mapBinaryYieldType(type: Type): ForstIndexTypeNode {
  if (isBytesType(type)) {
    return {
      $binary: true,
      element: { kind: "bytes" },
    } as ForstIndexTypeNode;
  }
  return mapType(type);
}

function getExportKind(isAsync: boolean, isGenerator: boolean): ForstNodeExportKind {
  if (isAsync && isGenerator) {
    return "asyncGenerator";
  }
  if (isGenerator) {
    return "generator";
  }
  if (isAsync) {
    return "asyncFunction";
  }
  return "function";
}

function isCallableExport(node: Node): node is FunctionDeclaration {
  if (Node.isFunctionDeclaration(node)) {
    return true;
  }
  if (Node.isVariableDeclaration(node)) {
    const init = node.getInitializer();
    return (
      init !== undefined &&
      (Node.isFunctionExpression(init) || Node.isArrowFunction(init))
    );
  }
  return false;
}

function getFunctionLikeFromExport(node: Node):
  | {
      isAsync: boolean;
      isGenerator: boolean;
      parameters: import("ts-morph").ParameterDeclaration[];
      returnTypeNode: Type;
    }
  | undefined {
  if (Node.isFunctionDeclaration(node)) {
    return {
      isAsync: node.isAsync(),
      isGenerator: node.isGenerator(),
      parameters: node.getParameters(),
      returnTypeNode: node.getReturnType(),
    };
  }

  if (Node.isVariableDeclaration(node)) {
    const init = node.getInitializer();
    if (init === undefined) {
      return undefined;
    }
    if (Node.isFunctionExpression(init) || Node.isArrowFunction(init)) {
      return {
        isAsync: init.isAsync(),
        isGenerator: Node.isFunctionExpression(init) ? init.isGenerator() : false,
        parameters: init.getParameters(),
        returnTypeNode: init.getReturnType(),
      };
    }
  }

  return undefined;
}

function indexCallableExport(name: string, node: Node): ForstIndexExportV1 | undefined {
  if (!isCallableExport(node)) {
    return undefined;
  }

  const fn = getFunctionLikeFromExport(node);
  if (!fn) {
    return undefined;
  }

  const kind = getExportKind(fn.isAsync, fn.isGenerator);
  const parameters: ForstIndexParameterV1[] = fn.parameters.map((param) => ({
    name: param.getName(),
    type: mapType(param.getType()),
  }));

  const exportEntry: ForstIndexExportV1 = {
    name,
    kind,
    parameters,
  };

  if (kind === "generator" || kind === "asyncGenerator") {
    const yieldType = generatorYieldType(fn.returnTypeNode);
    if (yieldType) {
      exportEntry.yieldType =
        kind === "generator" ? mapBinaryYieldType(yieldType) : mapType(yieldType);
    }

    const genReturn = generatorReturnType(fn.returnTypeNode);
    if (genReturn) {
      const mapped = mapType(genReturn);
      exportEntry.returnType =
        mapped.kind === "unknown" ? { kind: "void" } : mapped;
    } else {
      exportEntry.returnType = { kind: "void" };
    }
    return exportEntry;
  }

  exportEntry.returnType = mapType(unwrapPromise(fn.returnTypeNode));
  return exportEntry;
}

function resolveCallableExportDeclaration(declaration: Node): Node {
  if (Node.isExportSpecifier(declaration)) {
    const symbol = declaration.getSymbol();
    const aliased = symbol?.getAliasedSymbol();
    if (aliased) {
      const decls = aliased.getDeclarations();
      if (decls.length > 0) {
        return decls[0];
      }
    }

    const exportDecl = declaration.getExportDeclaration();
    const moduleSpecifier = exportDecl?.getModuleSpecifierSourceFile();
    if (moduleSpecifier) {
      const exported = moduleSpecifier.getExportedDeclarations().get(declaration.getName());
      if (exported && exported.length > 0) {
        return resolveCallableExportDeclaration(exported[0]);
      }
    }
  }

  return declaration;
}

function indexSourceFile(sourceFile: SourceFile, root: string): ForstIndexModuleV1 {
  const moduleId = toPosixModuleId(root, sourceFile.getFilePath());
  const exports: ForstIndexExportV1[] = [];

  for (const [name, declarations] of sourceFile.getExportedDeclarations()) {
    for (const declaration of declarations) {
      const target = resolveCallableExportDeclaration(declaration);
      const indexed = indexCallableExport(name, target);
      if (indexed) {
        exports.push(indexed);
        break;
      }
    }
  }

  exports.sort((a, b) => a.name.localeCompare(b.name));

  return { moduleId, exports };
}

export function emitForstIndexV1(options: EmitForstIndexV1Options): ForstIndexV1 {
  const root = path.resolve(options.root);
  const project = createIndexerProject({ root });
  addSourceFiles(project, root, options.files);

  const modules = options.files.map((relativeFile) => {
    const absolutePath = path.resolve(root, relativeFile);
    const sourceFile = project.getSourceFileOrThrow(absolutePath);
    return indexSourceFile(sourceFile, root);
  });

  return {
    format: FORST_INDEX_V1_FORMAT,
    modules,
  };
}

export function emitForstIndexV1Json(options: EmitForstIndexV1Options): string {
  return JSON.stringify(emitForstIndexV1(options), null, 2);
}

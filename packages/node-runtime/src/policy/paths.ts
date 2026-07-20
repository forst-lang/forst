import fs from "node:fs/promises";
import path from "node:path";
import { pathToFileURL } from "node:url";
import type { URL } from "node:url";
import * as Errors from "../rpc/errors.js";

const ALLOWED_EXTENSIONS = new Set([".ts", ".tsx", ".js"]);

function loadFilesExcludeFromEnv(): string[] {
  const raw = process.env.FORST_FILES_EXCLUDE;
  if (!raw) {
    return [];
  }
  try {
    const parsed: unknown = JSON.parse(raw);
    if (
      Array.isArray(parsed) &&
      parsed.every((entry) => typeof entry === "string")
    ) {
      return parsed;
    }
  } catch {
    // ignore malformed env JSON
  }
  return [];
}

const envFilesExclude = loadFilesExcludeFromEnv();
let activeFilesExclude: string[] = envFilesExclude;

/** Set exclude patterns from initialize params (falls back to FORST_FILES_EXCLUDE env). */
export function setFilesExcludePatterns(patterns: string[] | undefined): void {
  if (patterns && patterns.length > 0) {
    activeFilesExclude = patterns;
    return;
  }
  activeFilesExclude = envFilesExclude;
}

export function getFilesExcludePatterns(): readonly string[] {
  return activeFilesExclude;
}

function globPatternToRegExp(pattern: string): RegExp {
  let re = "^";
  for (let i = 0; i < pattern.length; ) {
    if (pattern.startsWith("**/", i)) {
      re += "(?:.*/)?";
      i += 3;
      continue;
    }
    if (pattern.startsWith("/**", i)) {
      re += "/.*";
      i += 3;
      continue;
    }
    if (pattern.startsWith("**", i)) {
      re += ".*";
      i += 2;
      continue;
    }
    const ch = pattern[i];
    if (ch === "*") {
      re += "[^/]*";
      i += 1;
      continue;
    }
    if (ch === "?") {
      re += "[^/]";
      i += 1;
      continue;
    }
    if (".+^${}()|[]\\".includes(ch)) {
      re += "\\" + ch;
    } else {
      re += ch;
    }
    i += 1;
  }
  re += "$";
  return new RegExp(re);
}

function matchesGlobPattern(pattern: string, candidate: string): boolean {
  const normalizedPattern = pattern.replace(/\\/g, "/");
  const normalizedCandidate = candidate.replace(/\\/g, "/");
  return globPatternToRegExp(normalizedPattern).test(normalizedCandidate);
}

function matchesExcludePatterns(
  moduleId: string,
  patterns: readonly string[]
): boolean {
  if (patterns.length === 0) {
    return false;
  }
  const posixPath = moduleId.replace(/\\/g, "/");
  return patterns.some((pattern) => matchesGlobPattern(pattern, posixPath));
}

/** Reject path traversal, absolute paths, and file:// URLs in RPC payloads. */
export function validateModuleIdSyntax(moduleId: string): void {
  if (typeof moduleId !== "string" || moduleId.trim() === "") {
    throw Errors.invalidParams("moduleId must be a non-empty string");
  }

  if (
    path.isAbsolute(moduleId) ||
    moduleId.startsWith("file://") ||
    moduleId.includes("\\")
  ) {
    throw Errors.forbidden("moduleId must be a project-relative POSIX path", {
      moduleId,
    });
  }

  if (moduleId.split("/").some((segment) => segment === "..")) {
    throw Errors.forbidden("moduleId must not contain .. segments", { moduleId });
  }

  const ext = path.posix.extname(moduleId);
  if (!ALLOWED_EXTENSIONS.has(ext)) {
    throw Errors.forbidden("moduleId extension not allowed", { moduleId, ext });
  }

  if (moduleId.includes("node_modules/")) {
    throw Errors.forbidden("moduleId must not reference node_modules", { moduleId });
  }

  if (matchesExcludePatterns(moduleId, getFilesExcludePatterns())) {
    throw Errors.forbidden("moduleId matches files.exclude pattern", { moduleId });
  }
}

function isUnderRoot(root: string, candidate: string): boolean {
  const relative = path.relative(root, candidate);
  return (
    relative === "" || (!relative.startsWith("..") && !path.isAbsolute(relative))
  );
}

/** Resolve moduleId under boundaryRoot with realpath hardening against symlink escape. */
export async function resolveModulePath(
  boundaryRoot: string,
  moduleId: string
): Promise<string> {
  validateModuleIdSyntax(moduleId);

  let resolvedRoot: string;
  try {
    resolvedRoot = await fs.realpath(boundaryRoot);
  } catch {
    throw Errors.invalidParams("boundaryRoot does not exist", { boundaryRoot });
  }

  const candidate = path.resolve(resolvedRoot, moduleId);

  let realCandidate: string;
  try {
    realCandidate = await fs.realpath(candidate);
  } catch {
    throw Errors.forbidden("moduleId does not resolve to an existing file", {
      moduleId,
    });
  }

  if (!isUnderRoot(resolvedRoot, realCandidate)) {
    throw Errors.forbidden("moduleId escapes boundaryRoot", {
      moduleId,
      boundaryRoot: resolvedRoot,
    });
  }

  const stat = await fs.stat(realCandidate);
  if (!stat.isFile()) {
    throw Errors.forbidden("moduleId must refer to a regular file", { moduleId });
  }

  return realCandidate;
}

export function modulePathToFileUrl(absPath: string): URL {
  return pathToFileURL(absPath);
}

export { matchesGlobPattern, matchesExcludePatterns };

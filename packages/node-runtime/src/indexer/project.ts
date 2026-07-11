import fs from "node:fs";
import path from "node:path";
import { Project, type ProjectOptions } from "ts-morph";

const DEFAULT_COMPILER_OPTIONS: ProjectOptions["compilerOptions"] = {
  target: 99,
  module: 99,
  strict: true,
  esModuleInterop: true,
  skipLibCheck: true,
  moduleResolution: 2,
};

function findTsConfigFile(root: string): string | undefined {
  let current = path.resolve(root);

  while (true) {
    const candidate = path.join(current, "tsconfig.json");
    if (fs.existsSync(candidate)) {
      return candidate;
    }

    const parent = path.dirname(current);
    if (parent === current) {
      return undefined;
    }
    current = parent;
  }
}

export interface CreateIndexerProjectOptions {
  root: string;
}

export function createIndexerProject(options: CreateIndexerProjectOptions): Project {
  const root = path.resolve(options.root);
  const tsConfigFilePath = findTsConfigFile(root);

  if (tsConfigFilePath) {
    return new Project({
      tsConfigFilePath,
      skipAddingFilesFromTsConfig: true,
    });
  }

  return new Project({
    compilerOptions: DEFAULT_COMPILER_OPTIONS,
  });
}

export function addSourceFiles(
  project: Project,
  root: string,
  relativeFiles: string[],
): void {
  const resolvedRoot = path.resolve(root);

  for (const relativeFile of relativeFiles) {
    const absolutePath = path.resolve(resolvedRoot, relativeFile);
    project.addSourceFileAtPath(absolutePath);
  }
}

export function toPosixModuleId(root: string, absolutePath: string): string {
  const resolvedRoot = path.resolve(root);
  const resolvedPath = path.resolve(absolutePath);
  const relative = path.relative(resolvedRoot, resolvedPath);
  return relative.split(path.sep).join("/");
}

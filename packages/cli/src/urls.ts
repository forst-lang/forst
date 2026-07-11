import { COMPILER_RELEASES_BASE } from "./constants.js";

/** `https://github.com/.../download/v{version}/{artifactName}` */
export function buildCompilerArtifactDownloadUrl(
  version: string,
  artifactName: string
): string {
  return `${COMPILER_RELEASES_BASE}/download/v${version}/${artifactName}`;
}

/** Download URL for the compiler Go module source tarball shipped with releases. */
export function buildCompilerModuleDownloadUrl(version: string): string {
  return buildCompilerArtifactDownloadUrl(version, `forst-module-${version}.tar.gz`);
}

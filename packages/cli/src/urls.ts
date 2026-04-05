import { COMPILER_RELEASES_BASE } from "./constants.js";

/** `https://github.com/.../download/v{version}/{artifactName}` */
export function buildCompilerArtifactDownloadUrl(
  version: string,
  artifactName: string
): string {
  return `${COMPILER_RELEASES_BASE}/download/v${version}/${artifactName}`;
}

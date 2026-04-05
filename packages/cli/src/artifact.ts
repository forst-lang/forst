import {
  UnsupportedArchitecture,
  UnsupportedOperatingSystem,
} from "./errors.js";

/**
 * Release asset filename for the given Node-style platform/arch,
 * e.g. `forst-darwin-arm64` or `forst-windows-amd64.exe`.
 */
export function getCompilerArtifactName(
  nodePlatform: NodeJS.Platform,
  nodeArch: string
): string {
  if (!["darwin", "linux", "win32"].includes(nodePlatform)) {
    throw new UnsupportedOperatingSystem(
      `Unsupported OS for prebuilt Forst: ${nodePlatform}`
    );
  }
  if (!["arm64", "x64"].includes(nodeArch)) {
    throw new UnsupportedArchitecture(
      `Unsupported architecture for prebuilt Forst: ${nodeArch}`
    );
  }

  const platformName = nodePlatform === "win32" ? "windows" : nodePlatform;
  const ext = nodePlatform === "win32" ? ".exe" : "";
  const archString = nodeArch === "x64" ? "amd64" : nodeArch;

  return `forst-${platformName}-${archString}${ext}`;
}

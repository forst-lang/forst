import { describe, expect, test } from "bun:test";
import { getCompilerModuleArtifactName } from "./compiler-module.js";
import { buildCompilerModuleDownloadUrl } from "./urls.js";

describe("compiler module release artifact", () => {
  test("artifact name includes version", () => {
    expect(getCompilerModuleArtifactName("1.2.3")).toBe(
      "forst-module-1.2.3.tar.gz"
    );
  });

  test("download url points at GitHub release asset", () => {
    expect(buildCompilerModuleDownloadUrl("0.0.37")).toBe(
      "https://github.com/forst-lang/forst/releases/download/v0.0.37/forst-module-0.0.37.tar.gz"
    );
  });
});

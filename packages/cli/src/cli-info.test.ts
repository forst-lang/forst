import { chmodSync, mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { describe, expect, test } from "bun:test";
import { printForstCliInfo } from "./cli-info.js";

describe("printForstCliInfo", () => {
  test.skipIf(process.platform === "win32")(
    "prints npm line, binary path, and forst version from fake FORST_BINARY",
    async () => {
    const dir = mkdtempSync(join(tmpdir(), "forst-cli-info-"));
    const fake = join(dir, "forst");
    const script = `#!/bin/sh
if [ "$1" = "version" ]; then
  echo "forst 1.2.3 fakecommit fake-date"
fi
`;
    writeFileSync(fake, script);
    chmodSync(fake, 0o755);

    const lines: string[] = [];
    const origLog = console.log;
    console.log = (...a: unknown[]) => {
      lines.push(a.map(String).join(" "));
    };
    try {
      await printForstCliInfo({
        env: { ...process.env, FORST_BINARY: fake },
      });
    } finally {
      console.log = origLog;
      rmSync(dir, { recursive: true, force: true });
    }

    const flat = lines.join("\n");
    expect(flat).toContain("@forst/cli (npm):");
    expect(flat).toContain(fake);
    expect(flat).toContain("1.2.3");
  }
  );
});

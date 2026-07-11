import { describe, expect, test } from "bun:test";
import { spawnSync } from "node:child_process";
import * as fs from "node:fs";
import * as os from "node:os";
import * as path from "node:path";

const packageRoot = path.resolve(__dirname, "..", "..");

describe("host/register.mjs", () => {
  test("blocks main until app ready module loads and signals readiness", () => {
    if (process.platform === "win32") {
      return;
    }

    const dir = fs.mkdtempSync(path.join(os.tmpdir(), "forst-register-test-"));
    const socketPath = path.join(dir, "node.sock");
    const readyPath = path.join(dir, "node.sock.ready");
    const appReadyModule = path.join(dir, "seed.mjs");
    const mainScript = path.join(dir, "main.mjs");
    const registerPath = path.join(packageRoot, "dist", "host", "register.mjs");

    fs.writeFileSync(
      appReadyModule,
      `globalThis.__forstRegisterTest = { seeded: true };\n`,
      "utf8"
    );
    fs.writeFileSync(
      mainScript,
      `import * as fs from "node:fs";
if (!globalThis.__forstRegisterTest?.seeded) {
  process.exit(42);
}
    const ready = JSON.parse(fs.readFileSync(${JSON.stringify(readyPath)}, "utf8"));
if (ready.phase !== "app") {
  process.exit(43);
}
process.exit(0);
`,
      "utf8"
    );

    const result = spawnSync(
      process.execPath,
      ["--import", registerPath, mainScript],
      {
        env: {
          ...process.env,
          FORST_NODE_HOST: "1",
          FORST_NODE_SOCKET: socketPath,
          FORST_NODE_HOST_READY: readyPath,
          FORST_NODE_APP_READY_MODULE: appReadyModule,
          NODE_OPTIONS: "",
        },
        encoding: "utf8",
      }
    );

    if (result.status !== 0) {
      throw new Error(
        `register ordering failed status=${result.status} stderr=${result.stderr} stdout=${result.stdout}`
      );
    }

    expect(result.status).toBe(0);
    expect(fs.existsSync(readyPath)).toBe(true);
  });

  test("signals readiness without app ready module", () => {
    if (process.platform === "win32") {
      return;
    }

    const dir = fs.mkdtempSync(path.join(os.tmpdir(), "forst-register-test-"));
    const socketPath = path.join(dir, "node.sock");
    const readyPath = path.join(dir, "node.sock.ready");
    const mainScript = path.join(dir, "main.mjs");
    const registerPath = path.join(packageRoot, "dist", "host", "register.mjs");

    fs.writeFileSync(
      mainScript,
      `import * as fs from "node:fs";
const ready = JSON.parse(fs.readFileSync(${JSON.stringify(readyPath)}, "utf8"));
if (ready.phase !== "app") {
  process.exit(44);
}
process.exit(0);
`,
      "utf8"
    );

    const result = spawnSync(
      process.execPath,
      ["--import", registerPath, mainScript],
      {
        env: {
          ...process.env,
          FORST_NODE_HOST: "1",
          FORST_NODE_SOCKET: socketPath,
          FORST_NODE_HOST_READY: readyPath,
          NODE_OPTIONS: "",
        },
        encoding: "utf8",
      }
    );

    if (result.status !== 0) {
      throw new Error(
        `register default readiness failed status=${result.status} stderr=${result.stderr} stdout=${result.stdout}`
      );
    }

    expect(result.status).toBe(0);
    expect(fs.existsSync(readyPath)).toBe(true);
  });
});

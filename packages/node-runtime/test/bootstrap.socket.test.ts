import { describe, expect, test, afterEach } from "bun:test";
import * as fs from "node:fs";
import * as net from "node:net";
import * as os from "node:os";
import * as path from "node:path";
import { spawn } from "node:child_process";
import { fileURLToPath } from "node:url";
import {
  newRequestFrame,
  ProtoFrameReader,
  writeProtoFrame,
} from "../src/rpc/frame.js";
import {
  METHOD_INITIALIZE,
  METHOD_PING,
  PROTOCOL_VERSION,
  WIRE_PROTOCOL_PROTO_V1,
} from "../src/rpc/protocol.js";
import {
  envReadyPath,
  envSocketPath,
} from "../src/rpc/socket_server.js";

const packageRoot = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  ".."
);
const bootstrapJs = path.join(packageRoot, "dist", "bootstrap.js");

function tempDir(): string {
  return fs.mkdtempSync(path.join(os.tmpdir(), "forst-bootstrap-socket-"));
}

describe("bootstrap socket integration", () => {
  const children: Array<ReturnType<typeof spawn>> = [];

  afterEach(() => {
    for (const child of children) {
      if (!child.killed) {
        child.kill("SIGTERM");
      }
    }
    children.length = 0;
  });

  test("dist/bootstrap.js listens and responds over socket", async () => {
    if (process.platform === "win32") {
      return;
    }
    if (!fs.existsSync(bootstrapJs)) {
      throw new Error(`build bootstrap first: missing ${bootstrapJs}`);
    }

    const dir = tempDir();
    const socketPath = path.join(dir, "node-bootstrap.sock");
    const readyPath = path.join(dir, "node-bootstrap.sock.ready");

    const child = spawn(process.execPath, [bootstrapJs], {
      env: {
        ...process.env,
        [envSocketPath]: socketPath,
        [envReadyPath]: readyPath,
        FORST_NODE_LOG_LEVEL: "error",
      },
      stdio: ["ignore", "pipe", "pipe"],
    });
    children.push(child);

    const deadline = Date.now() + 15000;
    while (Date.now() < deadline) {
      const marker = fs.existsSync(readyPath)
        ? JSON.parse(fs.readFileSync(readyPath, "utf8"))
        : null;
      if (marker?.phase === "app") {
        break;
      }
      await new Promise((r) => setTimeout(r, 50));
    }
    expect(fs.existsSync(readyPath)).toBe(true);

    const conn = net.createConnection(socketPath);
    await new Promise<void>((resolve, reject) => {
      conn.once("connect", () => resolve());
      conn.once("error", reject);
    });

    const reader = new ProtoFrameReader();
    const responses: unknown[] = [];
    conn.on("data", (chunk) => {
      reader.append(Buffer.from(chunk));
      for (;;) {
        const frame = reader.tryReadFrame();
        if (frame === null) {
          break;
        }
        if (frame.response?.okJson !== undefined) {
          responses.push(
            JSON.parse(new TextDecoder().decode(frame.response.okJson))
          );
        }
      }
    });

    writeProtoFrame(
      conn,
      newRequestFrame(1, METHOD_INITIALIZE, {
        protocolVersion: PROTOCOL_VERSION,
        boundaryRoot: dir,
        manifest: {
          version: 1,
          boundaryRoot: dir,
          exports: [],
        },
        supportedProtocols: [WIRE_PROTOCOL_PROTO_V1],
      })
    );
    writeProtoFrame(conn, newRequestFrame(2, METHOD_PING, {}));

    await new Promise<void>((resolve, reject) => {
      const pingDeadline = Date.now() + 5000;
      const tick = (): void => {
        if (responses.length >= 2) {
          resolve();
          return;
        }
        if (Date.now() > pingDeadline) {
          reject(new Error("timeout waiting for bootstrap ping"));
          return;
        }
        setTimeout(tick, 20);
      };
      tick();
    });

    conn.destroy();
    child.kill("SIGTERM");
    expect(responses.length).toBeGreaterThanOrEqual(2);
  });
});

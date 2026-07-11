import { describe, expect, test, beforeEach, afterEach } from "bun:test";
import * as fs from "node:fs";
import * as net from "node:net";
import * as os from "node:os";
import * as path from "node:path";
import {
  startForstNodeHost,
  signalForstAppReady,
  resetHostForTest,
} from "./host.js";

function tempDir(): string {
  return fs.mkdtempSync(path.join(os.tmpdir(), "forst-host-test-"));
}

describe("startForstNodeHost", () => {
  const prevHost = process.env.FORST_NODE_HOST;
  const prevSocket = process.env.FORST_NODE_SOCKET;
  const prevReady = process.env.FORST_NODE_HOST_READY;

  beforeEach(() => {
    process.env.FORST_NODE_HOST = "1";
  });

  afterEach(() => {
    void resetHostForTest();
    if (prevHost === undefined) {
      delete process.env.FORST_NODE_HOST;
    } else {
      process.env.FORST_NODE_HOST = prevHost;
    }
    if (prevSocket === undefined) {
      delete process.env.FORST_NODE_SOCKET;
    } else {
      process.env.FORST_NODE_SOCKET = prevSocket;
    }
    if (prevReady === undefined) {
      delete process.env.FORST_NODE_HOST_READY;
    } else {
      process.env.FORST_NODE_HOST_READY = prevReady;
    }
  });

  test("noop when FORST_NODE_HOST unset", async () => {
    delete process.env.FORST_NODE_HOST;
    const handle = await startForstNodeHost();
    expect(handle.socketPath).toBe("");
    await handle.close();
  });

  test("idempotent start returns same promise", async () => {
    if (process.platform === "win32") {
      return;
    }
    const dir = tempDir();
    const socketPath = path.join(dir, "node.sock");
    const readyPath = path.join(dir, "node.sock.ready");
    process.env.FORST_NODE_SOCKET = socketPath;
    process.env.FORST_NODE_HOST_READY = readyPath;

    const a = await startForstNodeHost();
    const b = await startForstNodeHost();
    expect(a.socketPath).toBe(socketPath);
    expect(b.socketPath).toBe(socketPath);
    expect(fs.existsSync(readyPath)).toBe(true);
    const ready = JSON.parse(fs.readFileSync(readyPath, "utf8")) as {
      pid: number;
      socket: string;
      phase: string;
    };
    expect(ready.pid).toBe(process.pid);
    expect(ready.socket).toBe(socketPath);
    expect(ready.phase).toBe("app");

    await a.close();
  });

  test("deferAppReady delays ready file until signalForstAppReady", async () => {
    if (process.platform === "win32") {
      return;
    }
    const dir = tempDir();
    const socketPath = path.join(dir, "node.sock");
    const readyPath = path.join(dir, "node.sock.ready");
    process.env.FORST_NODE_SOCKET = socketPath;
    process.env.FORST_NODE_HOST_READY = readyPath;

    await startForstNodeHost({ deferAppReady: true });
    expect(fs.existsSync(readyPath)).toBe(false);

    await signalForstAppReady();
    expect(fs.existsSync(readyPath)).toBe(true);
    const ready = JSON.parse(fs.readFileSync(readyPath, "utf8")) as {
      phase: string;
      socket: string;
    };
    expect(ready.phase).toBe("app");
    expect(ready.socket).toBe(socketPath);

    await signalForstAppReady();
    const readyAgain = JSON.parse(fs.readFileSync(readyPath, "utf8")) as {
      phase: string;
    };
    expect(readyAgain.phase).toBe("app");
  });

  test("rejects second client connection", async () => {
    if (process.platform === "win32") {
      return;
    }
    const dir = tempDir();
    const socketPath = path.join(dir, "node.sock");
    process.env.FORST_NODE_SOCKET = socketPath;

    await startForstNodeHost();

    const first = net.createConnection(socketPath);
    await new Promise<void>((resolve, reject) => {
      first.once("connect", () => resolve());
      first.once("error", reject);
    });

    const second = net.createConnection(socketPath);
    const destroyed = await new Promise<boolean>((resolve) => {
      second.once("close", () => resolve(second.destroyed));
      second.once("error", () => resolve(true));
      setTimeout(() => resolve(second.destroyed), 200);
    });
    expect(destroyed).toBe(true);

    first.destroy();
  });
});

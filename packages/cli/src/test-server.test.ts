import { describe, expect, test } from "bun:test";
import { EventEmitter } from "node:events";
import type { ChildProcess } from "node:child_process";
import {
  ForstInvokeServerExitedEarly,
  ForstInvokeServerStartTimeout,
  ForstInvokeServerUnreachable,
} from "./errors.js";
import {
  startForstInvokeServer,
  __test__,
  type SpawnFn,
} from "./test-server.js";

class FakeChild extends EventEmitter {
  pid = 4242;
  exitCode: number | null = null;
  signalCode: NodeJS.Signals | null = null;
  stdout = new EventEmitter();
  stderr = new EventEmitter();
  killed = false;
  killSignal: NodeJS.Signals | null = null;

  kill(signal?: NodeJS.Signals): boolean {
    this.killed = true;
    this.killSignal = signal ?? "SIGTERM";
    if (signal === "SIGKILL" || signal === "SIGTERM") {
      this.exitCode = signal === "SIGKILL" ? null : 0;
      this.signalCode = signal;
      queueMicrotask(() => this.emit("exit", this.exitCode, signal));
    }
    return true;
  }

  exit(code: number): void {
    this.exitCode = code;
    this.emit("exit", code, null);
  }
}

function okHealthFetch(calls: { n: number }, failUntil = 0): typeof fetch {
  return (async () => {
    calls.n += 1;
    if (calls.n <= failUntil) {
      throw new Error("ECONNREFUSED");
    }
    return new Response(JSON.stringify({ success: true }), { status: 200 });
  }) as typeof fetch;
}

describe("detectMode", () => {
  test("auto → embedded when server.embedded", () => {
    const mode = __test__.detectMode("/r", "auto", {
      existsSync: () => true,
      readFileSync: () => JSON.stringify({ server: { embedded: true } }),
    });
    expect(mode).toBe("embedded");
  });

  test("auto → embedded when node.hostMode", () => {
    const mode = __test__.detectMode("/r", "auto", {
      existsSync: () => true,
      readFileSync: () => JSON.stringify({ node: { hostMode: true } }),
    });
    expect(mode).toBe("embedded");
  });

  test("auto → dev otherwise", () => {
    const mode = __test__.detectMode("/r", "auto", {
      existsSync: () => true,
      readFileSync: () => JSON.stringify({}),
    });
    expect(mode).toBe("dev");
  });

  test("explicit mode wins", () => {
    expect(
      __test__.detectMode("/r", "dev", {
        existsSync: () => true,
        readFileSync: () => JSON.stringify({ server: { embedded: true } }),
      })
    ).toBe("dev");
  });
});

describe("resolveAttachTarget", () => {
  const fs = {
    existsSync: () => false,
    readFileSync: () => "",
  };

  test("options.baseUrl wins", () => {
    const got = __test__.resolveAttachTarget(
      { baseUrl: "http://127.0.0.1:9/" },
      "/r",
      { ...fs, env: {} }
    );
    expect(got).toEqual({ kind: "attach", url: "http://127.0.0.1:9" });
  });

  test("FORST_SKIP_SPAWN attaches via env URL", () => {
    const got = __test__.resolveAttachTarget({}, "/r", {
      ...fs,
      env: { FORST_SKIP_SPAWN: "1", FORST_BASE_URL: "http://127.0.0.1:6321" },
    });
    expect(got).toEqual({ kind: "attach", url: "http://127.0.0.1:6321" });
  });

  test("FORST_SKIP_SPAWN without URL throws", () => {
    expect(() =>
      __test__.resolveAttachTarget({}, "/r", { ...fs, env: { FORST_SKIP_SPAWN: "1" } })
    ).toThrow(ForstInvokeServerUnreachable);
  });

  test("invoke.ready attaches", () => {
    const got = __test__.resolveAttachTarget({}, "/r", {
      existsSync: (p) => String(p).endsWith("invoke.ready"),
      readFileSync: () => JSON.stringify({ url: "http://127.0.0.1:7000" }),
      env: {},
    });
    expect(got).toEqual({ kind: "attach", url: "http://127.0.0.1:7000" });
  });

  test("env URL attaches without skip", () => {
    const got = __test__.resolveAttachTarget({}, "/r", {
      ...fs,
      env: { FORST_INVOKE_URL: "http://127.0.0.1:8000" },
    });
    expect(got).toEqual({ kind: "attach", url: "http://127.0.0.1:8000" });
  });

  test("spawn when nothing points at a server", () => {
    const got = __test__.resolveAttachTarget({}, "/r", { ...fs, env: {} });
    expect(got).toEqual({ kind: "spawn" });
  });
});

describe("buildSpawnArgs", () => {
  test("dev argv", () => {
    expect(__test__.buildSpawnArgs("dev", { logLevel: "error" }, "/proj", 6320)).toEqual([
      "dev",
      "-port",
      "6320",
      "-root",
      "/proj",
      "-log-level",
      "error",
    ]);
  });

  test("embedded argv", () => {
    const args = __test__.buildSpawnArgs(
      "embedded",
      { entry: "cmd/main.ft" },
      "/proj",
      6321
    );
    expect(args.slice(0, 5)).toEqual([
      "run",
      "-export-struct-fields",
      "-root",
      "/proj",
      "--",
    ]);
    expect(args[5]).toContain("cmd/main.ft");
  });
});

describe("startForstInvokeServer", () => {
  test("attach path never spawns and stop is a no-op", async () => {
    let spawned = false;
    const calls = { n: 0 };
    const handle = await startForstInvokeServer(
      { baseUrl: "http://127.0.0.1:9" },
      {
        spawn: (() => {
          spawned = true;
          throw new Error("must not spawn");
        }) as SpawnFn,
        fetch: okHealthFetch(calls),
        env: {},
        sleep: async () => {},
      }
    );
    expect(spawned).toBe(false);
    expect(handle.connection).toBe("connect");
    expect(handle.pid).toBeUndefined();
    expect(handle.baseUrl).toBe("http://127.0.0.1:9");
    await handle.stop();
    await handle.stop();
  });

  test("attach unreachable becomes ForstInvokeServerUnreachable", async () => {
    await expect(
      startForstInvokeServer(
        { baseUrl: "http://127.0.0.1:9", timeoutMs: 50 },
        {
          fetch: (async () => {
            throw new Error("ECONNREFUSED");
          }) as typeof fetch,
          env: {},
          now: (() => {
            let t = 0;
            return () => {
              t += 100;
              return t;
            };
          })(),
          sleep: async () => {},
        }
      )
    ).rejects.toBeInstanceOf(ForstInvokeServerUnreachable);
  });

  test("spawn waits for health then returns handle", async () => {
    const child = new FakeChild();
    const calls = { n: 0 };
    const spawnArgs: string[][] = [];
    const handle = await startForstInvokeServer(
      { mode: "dev", port: 19001, root: "/proj", timeoutMs: 5_000 },
      {
        spawn: ((bin, args) => {
          spawnArgs.push([bin, ...args]);
          return child as unknown as ChildProcess;
        }) as SpawnFn,
        fetch: okHealthFetch(calls, 2),
        buildSpawnEnv: async () => ({ bin: "/bin/forst", env: {} }),
        env: {},
        sleep: async () => {},
        existsSync: () => false,
        readFileSync: () => "",
      }
    );
    expect(handle.connection).toBe("spawn");
    expect(handle.port).toBe(19001);
    expect(handle.pid).toBe(4242);
    expect(spawnArgs[0]?.[0]).toBe("/bin/forst");
    expect(spawnArgs[0]).toContain("dev");
    expect(calls.n).toBeGreaterThan(2);
    await handle.stop();
    expect(child.killed).toBe(true);
    expect(child.killSignal).toBe("SIGTERM");
  });

  test("child exit before ready throws ForstInvokeServerExitedEarly", async () => {
    const child = new FakeChild();
    await expect(
      startForstInvokeServer(
        { mode: "dev", port: 19002, timeoutMs: 5_000 },
        {
          spawn: (() => {
            child.exit(1);
            return child as unknown as ChildProcess;
          }) as SpawnFn,
          fetch: (async () => {
            throw new Error("ECONNREFUSED");
          }) as typeof fetch,
          buildSpawnEnv: async () => ({ bin: "/bin/forst", env: {} }),
          env: {},
          sleep: async () => {},
          existsSync: () => false,
          readFileSync: () => "",
          stopGraceMs: 10,
        }
      )
    ).rejects.toBeInstanceOf(ForstInvokeServerExitedEarly);
  });

  test("timeout throws ForstInvokeServerStartTimeout", async () => {
    const child = new FakeChild();
    let t = 0;
    await expect(
      startForstInvokeServer(
        { mode: "dev", port: 19003, timeoutMs: 250 },
        {
          spawn: (() => child as unknown as ChildProcess) as SpawnFn,
          fetch: (async () => {
            throw new Error("ECONNREFUSED");
          }) as typeof fetch,
          buildSpawnEnv: async () => ({ bin: "/bin/forst", env: {} }),
          env: {},
          now: () => {
            t += 100;
            return t;
          },
          sleep: async () => {},
          existsSync: () => false,
          readFileSync: () => "",
        }
      )
    ).rejects.toBeInstanceOf(ForstInvokeServerStartTimeout);
  });

  test("bind failure retries with a new port once", async () => {
    const children = [new FakeChild(), new FakeChild()];
    let spawnCount = 0;
    const ports: number[] = [];
    let freePort = 19100;
    const handle = await startForstInvokeServer(
      { mode: "dev", timeoutMs: 5_000 },
      {
        spawn: ((_bin, args) => {
          const portIdx = args.indexOf("-port");
          ports.push(Number(args[portIdx + 1]));
          const child = children[spawnCount]!;
          spawnCount += 1;
          if (spawnCount === 1) {
            // Emit after spawn returns so captureStream/exit listeners are attached.
            queueMicrotask(() => {
              child.stderr.emit(
                "data",
                "listen tcp: bind: address already in use\n"
              );
              child.exit(1);
            });
          }
          return child as unknown as ChildProcess;
        }) as SpawnFn,
        fetch: (async (input) => {
          const url = String(input);
          if (ports[0] && url.includes(`:${ports[0]}/`)) {
            throw new Error("ECONNREFUSED");
          }
          return new Response("{}", { status: 200 });
        }) as typeof fetch,
        buildSpawnEnv: async () => ({ bin: "/bin/forst", env: {} }),
        createServer: (() => {
          const ee = new EventEmitter();
          const port = ++freePort;
          (ee as unknown as { listen: Function }).listen = (
            _p: number,
            _h: string,
            cb: () => void
          ) => {
            queueMicrotask(cb);
            return ee;
          };
          (ee as unknown as { address: Function }).address = () => ({
            port,
            family: "IPv4",
            address: "127.0.0.1",
          });
          (ee as unknown as { close: Function }).close = (
            cb?: (err?: Error) => void
          ) => {
            cb?.();
            return ee;
          };
          (ee as unknown as { on: Function }).on = (
            _ev: string,
            _fn: (...a: unknown[]) => void
          ) => ee;
          return ee;
        }) as typeof import("node:net").createServer,
        env: {},
        sleep: async () => {},
        existsSync: () => false,
        readFileSync: () => "",
        stopGraceMs: 10,
      }
    );
    expect(spawnCount).toBe(2);
    expect(ports[0]).not.toBe(ports[1]);
    expect(handle.port).toBe(ports[1]);
    await handle.stop();
  });

  test("stop escalates to SIGKILL when SIGTERM is ignored", async () => {
    const child = new FakeChild();
    child.kill = (signal?: NodeJS.Signals) => {
      child.killSignal = signal ?? "SIGTERM";
      child.killed = true;
      if (signal === "SIGKILL") {
        child.signalCode = "SIGKILL";
        queueMicrotask(() => child.emit("exit", null, "SIGKILL"));
      }
      // ignore SIGTERM
      return true;
    };
    const handle = await startForstInvokeServer(
      { mode: "dev", port: 19004, timeoutMs: 5_000 },
      {
        spawn: (() => child as unknown as ChildProcess) as SpawnFn,
        fetch: okHealthFetch({ n: 0 }),
        buildSpawnEnv: async () => ({ bin: "/bin/forst", env: {} }),
        env: {},
        sleep: async () => {},
        existsSync: () => false,
        readFileSync: () => "",
        stopGraceMs: 20,
      }
    );
    await handle.stop();
    await handle.stop();
    expect(child.killSignal).toBe("SIGKILL");
  });

  test("Symbol.asyncDispose stops the handle", async () => {
    const child = new FakeChild();
    const handle = await startForstInvokeServer(
      { mode: "dev", port: 19005 },
      {
        spawn: (() => child as unknown as ChildProcess) as SpawnFn,
        fetch: okHealthFetch({ n: 0 }),
        buildSpawnEnv: async () => ({ bin: "/bin/forst", env: {} }),
        env: {},
        sleep: async () => {},
        existsSync: () => false,
        readFileSync: () => "",
      }
    );
    await handle[Symbol.asyncDispose]();
    expect(child.killed).toBe(true);
  });
});

import { Effect, Layer, Exit } from "effect";
import { Echo, Main } from "@forst/gen/main";
import { ForstTransport } from "@forst/gen/$effect";
import { ForstTestLayer } from "@forst/gen/$testing";

function assert(cond, msg) {
  if (!cond) {
    throw new Error(msg);
  }
}

// Unstubbed method under Layer.mock fails with UnimplementedError.
const partial = Layer.mock(Main, {});

const unstubbed = await Effect.runPromiseExit(
  Echo({ message: "x" }).pipe(Effect.provide(partial))
);
assert(Exit.isFailure(unstubbed), "unstubbed call must fail");
const pretty = String(unstubbed.cause);
assert(
  pretty.includes("UnimplementedError") || pretty.includes("Not implemented"),
  "expected UnimplementedError defect, got: " + pretty
);

// Interrupted fiber aborts in-flight HTTP via tryPromise signal.
let aborted = false;
const slowTransport = {
  async invokeFunction(_pkg, _fn, _args, options) {
    return await new Promise((_resolve, reject) => {
      const onAbort = () => {
        aborted = true;
        reject(Object.assign(new Error("aborted"), { name: "AbortError" }));
      };
      if (options?.signal?.aborted) {
        onAbort();
        return;
      }
      options?.signal?.addEventListener("abort", onAbort, { once: true });
    });
  },
  async *invokeStream() {},
};

const transportLayer = Layer.succeed(ForstTransport, {
  client: slowTransport,
});

const program = Echo({ message: "slow" }).pipe(
  Effect.provide(Main.DefaultWithoutDependencies),
  Effect.provide(transportLayer),
  Effect.timeout("50 millis")
);

const timed = await Effect.runPromiseExit(program);
assert(Exit.isFailure(timed), "timeout must fail the effect");
assert(aborted, "interruption must abort the in-flight request");

// ForstTestLayer value handler works without transport / base URL.
const testLayer = ForstTestLayer({
  packages: {
    main: {
      Echo: () => ({ echo: "from-test", timestamp: 42 }),
    },
  },
});
const result = await Effect.runPromise(
  Echo({ message: "n" }).pipe(Effect.provide(testLayer))
);
assert(result.echo === "from-test", "ForstTestLayer handler must run");

console.log("effect-runtime-ok");

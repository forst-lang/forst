import { Effect, Layer, Exit } from "effect";
import { $main } from "@forst/gen/main";
import { ForstTransport } from "@forst/gen/$transport";
import { ForstTestLayer } from "@forst/gen/$testing";

function assert(cond, msg) {
  if (!cond) {
    throw new Error(msg);
  }
}

// Unstubbed method under Layer.mock fails with UnimplementedError.
const partial = Layer.mock($main, {});

const unstubbed = await Effect.runPromiseExit(
  $main.Echo({ message: "x" }).pipe(Effect.provide(partial))
);
assert(Exit.isFailure(unstubbed), "unstubbed call must fail");
const pretty = String(unstubbed.cause);
assert(
  pretty.includes("UnimplementedError") || pretty.includes("Not implemented"),
  "expected UnimplementedError defect, got: " + pretty
);

// Timeout fails the in-flight invoke effect.
const slowTransport = {
  async invokeFunction(_pkg, _fn, _args, _options) {
    return await new Promise(() => {});
  },
  async *invokeStream() {},
};

const transportLayer = Layer.succeed(ForstTransport, {
  client: slowTransport,
});

const program = $main.Echo({ message: "slow" }).pipe(
  Effect.provide($main.DefaultWithoutDependencies),
  Effect.provide(transportLayer),
  Effect.timeout("200 millis")
);

const timed = await Effect.runPromiseExit(program);
assert(Exit.isFailure(timed), "timeout must fail the effect");

// ForstTestLayer value handler works without transport / base URL.
const testLayer = ForstTestLayer({
  packages: {
    main: {
      Echo: () => ({ echo: "from-test", timestamp: 42 }),
    },
  },
});
const result = await Effect.runPromise(
  $main.Echo({ message: "n" }).pipe(Effect.provide(testLayer))
);
assert(result.echo === "from-test", "ForstTestLayer handler must run");

console.log("effect-runtime-ok");

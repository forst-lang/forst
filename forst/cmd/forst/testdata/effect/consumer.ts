/**
 * Effect mode type-level fixture. @ts-expect-error lines must stay errors;
 * if signatures widen to any, these directives fail the build.
 */
import { Effect, Layer } from "effect";
import { ForstClientLive } from "@forst/gen";
import { $main } from "@forst/gen/main";
import { ForstTransport } from "@forst/gen/$transport";
import { ForstTestLayer } from "@forst/gen/$testing";

// Happy path: catchTag + retry + provide compiles.
const happy = $main.Echo({ message: "hi" }).pipe(
  Effect.catchTag("@forst/errors/InvokeTimedOut", () =>
    Effect.succeed({ echo: "x", timestamp: 0 })
  ),
  Effect.retry({ times: 2 }),
  Effect.provide(ForstClientLive)
);
void happy;

// @ts-expect-error runPromise without providing the package service
Effect.runPromise($main.Echo({ message: "hi" }));

// @ts-expect-error Echo input must be $EchoRequest
void $main.Echo({ notMessage: true });

// DefaultWithoutDependencies requires ForstTransport and accepts a mock of it.
const transportMock = Layer.mock(ForstTransport, {
  client: {
    invokeFunction: async <T>(_p: string, _f: string, _a?: unknown[]) =>
      ({
        success: true as const,
        result: { echo: "ok", timestamp: 1 } as T,
      }),
    async *invokeStream() {},
  },
});
const withFakeTransport = $main.Echo({ message: "hi" }).pipe(
  Effect.provide($main.DefaultWithoutDependencies),
  Effect.provide(transportMock)
);
void withFakeTransport;

// ForstTestLayer override shape
void ForstTestLayer({
  packages: {
    main: {
      Echo: () => ({ echo: "stub", timestamp: 1 }),
    },
  },
});

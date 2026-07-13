import { Effect } from "effect";
import * as Deferred from "effect/Deferred";
import * as Fiber from "effect/Fiber";
import * as Queue from "effect/Queue";
import * as Ref from "effect/Ref";
import type { InvokeTransport } from "./transport";

export interface DevServerHealthSnapshot {
  success?: boolean;
  reloading?: boolean;
  generation?: number;
  result?: {
    reloading?: boolean;
    generation?: number;
    nativeWatch?: boolean;
  };
}

export interface ReloadAwareTransportConfig {
  /** Poll interval cap while waiting for reload to finish (ms). */
  maxPollIntervalMs?: number;
  /** Re-resolve invoke base URL each health poll (e.g. from invoke.ready). */
  resolveBaseUrl?: () => string | undefined;
  fetchHealth?: (transport: InvokeTransport) => Effect.Effect<DevServerHealthSnapshot, Error>;
}

type ParkedRequest = {
  readonly endpoint: string;
  readonly init: RequestInit | undefined;
  readonly deferred: Deferred.Deferred<Response, Error>;
};

const defaultMaxPollIntervalMs = 30_000;

function isReloadHttpStatus(status: number): boolean {
  return status === 503;
}

function parseRetryAfterMs(response: Response): number | undefined {
  const header = response.headers.get("Retry-After");
  if (!header) {
    return undefined;
  }
  const seconds = Number(header);
  if (Number.isFinite(seconds) && seconds >= 0) {
    return Math.min(seconds * 1000, defaultMaxPollIntervalMs);
  }
  return undefined;
}

function healthIndicatesReloading(snapshot: DevServerHealthSnapshot): boolean {
  if (snapshot.reloading === true) {
    return true;
  }
  return snapshot.result?.reloading === true;
}

async function readReloadingFromBody(response: Response): Promise<boolean> {
  if (!isReloadHttpStatus(response.status)) {
    return false;
  }
  try {
    const text = await response.text();
    const parsed = JSON.parse(text) as {
      error?: string;
      reloading?: boolean;
    };
    return (
      parsed.reloading === true ||
      parsed.error === "reloading" ||
      /reloading/i.test(parsed.error ?? "")
    );
  } catch {
    return true;
  }
}

function isConnectionError(error: unknown): boolean {
  if (!(error instanceof Error)) {
    return false;
  }
  const msg = error.message.toLowerCase();
  return (
    msg.includes("econnrefused") ||
    msg.includes("fetch failed") ||
    msg.includes("network") ||
    msg.includes("failed to fetch")
  );
}

async function defaultFetchHealth(
  transport: InvokeTransport
): Promise<DevServerHealthSnapshot> {
  try {
    const response = await Effect.runPromise(
      transport.request("/health", { method: "GET" })
    );
    if (!response.ok) {
      return { success: false, reloading: true };
    }
    return (await response.json()) as DevServerHealthSnapshot;
  } catch (error) {
    if (isConnectionError(error)) {
      return { success: false, reloading: true };
    }
    throw error instanceof Error ? error : new Error(String(error));
  }
}

function defaultFetchHealthEffect(
  transport: InvokeTransport
): Effect.Effect<DevServerHealthSnapshot, Error> {
  return Effect.tryPromise({
    try: () => defaultFetchHealth(transport),
    catch: (error) =>
      error instanceof Error ? error : new Error(String(error)),
  });
}

/** Wraps an {@link InvokeTransport}; parks requests on 503/reloading and replays when ready. */
export function createReloadAwareTransport(
  inner: InvokeTransport,
  config: ReloadAwareTransportConfig = {}
): InvokeTransport {
  const maxPollIntervalMs = config.maxPollIntervalMs ?? defaultMaxPollIntervalMs;
  const fetchHealth = config.fetchHealth ?? defaultFetchHealthEffect;

  const runtime = Effect.gen(function* () {
    const parkQueue = yield* Queue.unbounded<ParkedRequest>();
    const drainFiber = yield* Ref.make<Fiber.RuntimeFiber<void, never> | null>(
      null
    );
    return { parkQueue, drainFiber };
  });

  let runtimePromise: Promise<{
    parkQueue: Queue.Queue<ParkedRequest>;
    drainFiber: Ref.Ref<Fiber.RuntimeFiber<void, never> | null>;
  }> | null = null;

  const ensureRuntime = () => {
    if (!runtimePromise) {
      runtimePromise = Effect.runPromise(runtime);
    }
    return runtimePromise;
  };

  const waitUntilReady = Effect.gen(function* () {
    let attempt = 0;
    while (true) {
      const snapshot = yield* fetchHealth(inner);
      if (!healthIndicatesReloading(snapshot)) {
        return;
      }
      const delay = Math.min(250 * 2 ** attempt, maxPollIntervalMs);
      yield* Effect.sleep(delay);
      attempt += 1;
    }
  });

  const replayParked = (parked: ParkedRequest) =>
    Effect.gen(function* () {
      while (true) {
        yield* waitUntilReady;
        while (true) {
          const response = yield* inner.request(parked.endpoint, parked.init).pipe(
            Effect.catchAll((error) => {
              if (!isConnectionError(error)) {
                return Effect.fail(error);
              }
              return Effect.succeed(null as Response | null);
            })
          );
          if (response === null) {
            break;
          }
          if (!isReloadHttpStatus(response.status)) {
            yield* Deferred.succeed(parked.deferred, response);
            return;
          }
          const reloading = yield* Effect.tryPromise({
            try: () => readReloadingFromBody(response),
            catch: (error) =>
              error instanceof Error ? error : new Error(String(error)),
          });
          if (!reloading) {
            yield* Deferred.succeed(parked.deferred, response);
            return;
          }
          const retryAfterMs = parseRetryAfterMs(response);
          if (retryAfterMs !== undefined) {
            yield* Effect.sleep(retryAfterMs);
          }
        }
      }
    });

  const drainLoop = (parkQueue: Queue.Queue<ParkedRequest>) =>
    Effect.forever(
      Effect.gen(function* () {
        const parked = yield* Queue.take(parkQueue);
        yield* replayParked(parked).pipe(
          Effect.catchAll((error) => Deferred.fail(parked.deferred, error))
        );
      })
    );

  const ensureDrainFiber = (
    parkQueue: Queue.Queue<ParkedRequest>,
    drainFiber: Ref.Ref<Fiber.RuntimeFiber<void, never> | null>
  ) =>
    Effect.gen(function* () {
      const existing = yield* Ref.get(drainFiber);
      if (existing !== null) {
        return;
      }
      const fiber = yield* Effect.fork(drainLoop(parkQueue));
      yield* Ref.set(drainFiber, fiber);
    });

  const parkRequest = (
    endpoint: string,
    init: RequestInit | undefined,
    parkQueue: Queue.Queue<ParkedRequest>,
    drainFiber: Ref.Ref<Fiber.RuntimeFiber<void, never> | null>
  ) =>
    Effect.gen(function* () {
      const deferred = yield* Deferred.make<Response, Error>();
      yield* Queue.offer(parkQueue, { endpoint, init, deferred });
      yield* ensureDrainFiber(parkQueue, drainFiber);
      return yield* Deferred.await(deferred);
    });

  return {
    request(endpoint, init) {
      return Effect.gen(function* () {
        const response = yield* inner.request(endpoint, init).pipe(
          Effect.catchAll((error) => {
            if (!isConnectionError(error)) {
              return Effect.fail(error);
            }
            return Effect.gen(function* () {
              const { parkQueue, drainFiber } = yield* Effect.promise(() =>
                ensureRuntime()
              );
              return yield* parkRequest(endpoint, init, parkQueue, drainFiber);
            });
          })
        );
        if (!isReloadHttpStatus(response.status)) {
          return response;
        }

        const reloading = yield* Effect.tryPromise({
          try: () => readReloadingFromBody(response),
          catch: (error) =>
            error instanceof Error ? error : new Error(String(error)),
        });
        if (!reloading) {
          return response;
        }

        const { parkQueue, drainFiber } = yield* Effect.promise(() =>
          ensureRuntime()
        );
        return yield* parkRequest(endpoint, init, parkQueue, drainFiber);
      });
    },
  };
}

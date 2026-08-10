import { Effect } from "effect";
import type { InvokeTransport } from "./transport";
import type { InvokeChallenge } from "./invoke-auth";

export function fetchInvokeChallenge(
  transport: InvokeTransport
): Effect.Effect<InvokeChallenge, Error> {
  return Effect.gen(function* () {
    const response = yield* transport.request("/invoke/challenge", {
      method: "GET",
    });
    if (!response.ok) {
      return yield* Effect.fail(
        new Error(`invoke challenge failed: HTTP ${response.status}`)
      );
    }
    const payload = (yield* Effect.tryPromise({
      try: () => response.json(),
      catch: (error) =>
        error instanceof Error ? error : new Error(String(error)),
    })) as {
      success?: boolean;
      result?: InvokeChallenge | string;
    };
    const raw = payload.result;
    const parsed =
      typeof raw === "string"
        ? (JSON.parse(raw) as InvokeChallenge)
        : (raw as InvokeChallenge | undefined);
    if (!parsed?.nonce) {
      return yield* Effect.fail(new Error("invoke challenge missing nonce"));
    }
    return parsed;
  });
}

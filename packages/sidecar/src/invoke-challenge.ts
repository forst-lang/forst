import { Effect } from "effect";
import {
  parseInvokeChallengeResult,
  type InvokeChallenge,
} from "@forst/cli/invoke";
import type { InvokeTransport } from "./transport";

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
    const parsed = parseInvokeChallengeResult(payload);
    if (!parsed) {
      return yield* Effect.fail(new Error("invoke challenge missing nonce"));
    }
    return parsed;
  });
}

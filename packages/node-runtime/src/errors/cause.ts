import { Data } from "effect";

/** Wraps non-`Error` rejection values so Effect catch paths and RPC error serialization always see an `Error`. */
export class UnknownCauseError extends Data.TaggedError("UnknownCauseError")<{
  readonly value: unknown;
  readonly message: string;
}> {}

/** Normalizes unknown catch/rejection values into an `Error` for consistent logging and RPC error mapping. */
export function causeToError(cause: unknown): Error {
  return cause instanceof Error
    ? cause
    : new UnknownCauseError({ value: cause, message: String(cause) });
}

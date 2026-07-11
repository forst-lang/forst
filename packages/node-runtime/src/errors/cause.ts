import { Data } from "effect";

export class UnknownCauseError extends Data.TaggedError("UnknownCauseError")<{
  readonly value: unknown;
  readonly message: string;
}> {}

export function causeToError(cause: unknown): Error {
  return cause instanceof Error
    ? cause
    : new UnknownCauseError({ value: cause, message: String(cause) });
}

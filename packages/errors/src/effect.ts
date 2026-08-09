import { Data } from "effect";
import { errorsTag } from "./tags.js";

export { ERRORS_TAG_PREFIX, errorsTag } from "./tags.js";

/** Effect TaggedError constructor shape used for JSR-friendly superclass extraction. */
type EffectTaggedBase<A extends Record<string, unknown>> = new (
  args: A,
) => Error & { readonly _tag: string } & Readonly<A>;

const InvokeRejectedBase: EffectTaggedBase<{
  readonly packageName: string;
  readonly functionName: string;
  readonly serverError?: string;
  readonly message?: string;
}> = Data.TaggedError(errorsTag("InvokeRejected"))<{
  readonly packageName: string;
  readonly functionName: string;
  readonly serverError?: string;
  readonly message?: string;
}>;
export class InvokeRejected extends InvokeRejectedBase {}

const InvokeHttpFailureBase: EffectTaggedBase<{
  readonly packageName: string;
  readonly functionName: string;
  readonly status: number;
  readonly responseText: string;
  readonly message?: string;
}> = Data.TaggedError(errorsTag("InvokeHttpFailure"))<{
  readonly packageName: string;
  readonly functionName: string;
  readonly status: number;
  readonly responseText: string;
  readonly message?: string;
}>;
export class InvokeHttpFailure extends InvokeHttpFailureBase {}

const InvokeTimedOutBase: EffectTaggedBase<{
  readonly packageName: string;
  readonly functionName: string;
  readonly timeoutMs?: number;
  readonly message?: string;
}> = Data.TaggedError(errorsTag("InvokeTimedOut"))<{
  readonly packageName: string;
  readonly functionName: string;
  readonly timeoutMs?: number;
  readonly message?: string;
}>;
export class InvokeTimedOut extends InvokeTimedOutBase {}

const InvokeUnreachableBase: EffectTaggedBase<{
  readonly packageName: string;
  readonly functionName: string;
  readonly baseUrl: string;
  readonly message?: string;
}> = Data.TaggedError(errorsTag("InvokeUnreachable"))<{
  readonly packageName: string;
  readonly functionName: string;
  readonly baseUrl: string;
  readonly message?: string;
}>;
export class InvokeUnreachable extends InvokeUnreachableBase {}

const InvokeBaseUrlMissingBase: EffectTaggedBase<{
  readonly envVar: string;
  readonly nodeEnv: string;
  readonly message?: string;
}> = Data.TaggedError(errorsTag("InvokeBaseUrlMissing"))<{
  readonly envVar: string;
  readonly nodeEnv: string;
  readonly message?: string;
}>;
export class InvokeBaseUrlMissing extends InvokeBaseUrlMissingBase {}

const InvokeStreamAbortedBase: EffectTaggedBase<{
  readonly packageName: string;
  readonly functionName: string;
  readonly rowIndex: number;
  readonly message?: string;
}> = Data.TaggedError(errorsTag("InvokeStreamAborted"))<{
  readonly packageName: string;
  readonly functionName: string;
  readonly rowIndex: number;
  readonly message?: string;
}>;
export class InvokeStreamAborted extends InvokeStreamAbortedBase {}

const ContractVersionMismatchBase: EffectTaggedBase<{
  readonly expectedContractVersion: string;
  readonly serverContractVersion: string;
  readonly message?: string;
}> = Data.TaggedError(errorsTag("ContractVersionMismatch"))<{
  readonly expectedContractVersion: string;
  readonly serverContractVersion: string;
  readonly message?: string;
}>;
export class ContractVersionMismatch extends ContractVersionMismatchBase {}

const ForstTestServerFailedBase: EffectTaggedBase<{
  readonly reason: "cli_missing" | "spawn_failed" | "ready_timeout" | "unreachable";
  readonly installCommand?: string;
  readonly causeMessage?: string;
  readonly message?: string;
}> = Data.TaggedError(errorsTag("ForstTestServerFailed"))<{
  readonly reason: "cli_missing" | "spawn_failed" | "ready_timeout" | "unreachable";
  readonly installCommand?: string;
  readonly causeMessage?: string;
  readonly message?: string;
}>;
export class ForstTestServerFailed extends ForstTestServerFailedBase {}

const ForstUnknownFailureBase: EffectTaggedBase<{
  readonly message: string;
  readonly serverError?: string;
  readonly tag?: string;
  readonly packageName?: string;
  readonly functionName?: string;
}> = Data.TaggedError(errorsTag("ForstUnknownFailure"))<{
  readonly message: string;
  readonly serverError?: string;
  readonly tag?: string;
  readonly packageName?: string;
  readonly functionName?: string;
}>;
export class ForstUnknownFailure extends ForstUnknownFailureBase {}

export type InvokeFailure =
  | InvokeRejected
  | InvokeHttpFailure
  | InvokeTimedOut
  | InvokeUnreachable
  | InvokeBaseUrlMissing
  | InvokeStreamAborted
  | ContractVersionMismatch;

const INVOKE_FAILURE_TAGS = new Set<string>([
  errorsTag("InvokeRejected"),
  errorsTag("InvokeHttpFailure"),
  errorsTag("InvokeTimedOut"),
  errorsTag("InvokeUnreachable"),
  errorsTag("InvokeBaseUrlMissing"),
  errorsTag("InvokeStreamAborted"),
  errorsTag("ContractVersionMismatch"),
]);

export const isInvokeFailure = (u: unknown): u is InvokeFailure =>
  u instanceof Error && INVOKE_FAILURE_TAGS.has((u as { _tag?: string })._tag ?? "");

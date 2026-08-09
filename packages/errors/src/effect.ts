import { Data } from "effect";
import { errorsTag } from "./tags.js";

export { ERRORS_TAG_PREFIX, errorsTag } from "./tags.js";

export class InvokeRejected extends Data.TaggedError(errorsTag("InvokeRejected"))<{
  readonly packageName: string;
  readonly functionName: string;
  readonly serverError?: string;
  readonly message?: string;
}> {}

export class InvokeHttpFailure extends Data.TaggedError(errorsTag("InvokeHttpFailure"))<{
  readonly packageName: string;
  readonly functionName: string;
  readonly status: number;
  readonly responseText: string;
  readonly message?: string;
}> {}

export class InvokeTimedOut extends Data.TaggedError(errorsTag("InvokeTimedOut"))<{
  readonly packageName: string;
  readonly functionName: string;
  readonly timeoutMs?: number;
  readonly message?: string;
}> {}

export class InvokeUnreachable extends Data.TaggedError(errorsTag("InvokeUnreachable"))<{
  readonly packageName: string;
  readonly functionName: string;
  readonly baseUrl: string;
  readonly message?: string;
}> {}

export class InvokeBaseUrlMissing extends Data.TaggedError(errorsTag("InvokeBaseUrlMissing"))<{
  readonly envVar: string;
  readonly nodeEnv: string;
  readonly message?: string;
}> {}

export class InvokeStreamAborted extends Data.TaggedError(errorsTag("InvokeStreamAborted"))<{
  readonly packageName: string;
  readonly functionName: string;
  readonly rowIndex: number;
  readonly message?: string;
}> {}

export class ContractVersionMismatch extends Data.TaggedError(errorsTag("ContractVersionMismatch"))<{
  readonly expectedContractVersion: string;
  readonly serverContractVersion: string;
  readonly message?: string;
}> {}

export class ForstTestServerFailed extends Data.TaggedError(errorsTag("ForstTestServerFailed"))<{
  readonly reason: "cli_missing" | "spawn_failed" | "ready_timeout" | "unreachable";
  readonly installCommand?: string;
  readonly causeMessage?: string;
  readonly message?: string;
}> {}

export class ForstUnknownFailure extends Data.TaggedError(errorsTag("ForstUnknownFailure"))<{
  readonly message: string;
  readonly serverError?: string;
  readonly tag?: string;
  readonly packageName?: string;
  readonly functionName?: string;
}> {}

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

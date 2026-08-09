import { ERRORS_TAG_PREFIX, errorsTag } from "./tags.js";

export { ERRORS_TAG_PREFIX, errorsTag };

/** Constructor returned by {@link tagged}. */
export type TaggedBaseConstructor = new (
  props?: { message?: string } & Record<string, unknown>,
) => Error;

/** Promise-mode tagged error helper used by generated Forst clients. */
export const tagged = (tag: string): TaggedBaseConstructor =>
  class extends Error {
    constructor(props?: { message?: string } & Record<string, unknown>) {
      super(props?.message ?? tag);
      Object.assign(this, props);
      Object.defineProperty(this, "_tag", {
        value: tag,
        enumerable: true,
        writable: false,
      });
      this.name = tag;
      Object.setPrototypeOf(this, new.target.prototype);
      const Err = Error as ErrorConstructor & {
        captureStackTrace?: (target: object, ctor: Function) => void;
      };
      if (Err.captureStackTrace) Err.captureStackTrace(this, new.target);
    }
  };

export type TaggedError<
  Tag extends string,
  A extends Record<string, unknown> = Record<string, unknown>,
> = Error & { readonly _tag: Tag } & Readonly<A>;

const InvokeRejectedBase: TaggedBaseConstructor = tagged(errorsTag("InvokeRejected"));
export class InvokeRejected extends InvokeRejectedBase {
  declare readonly _tag: "@forst/errors/InvokeRejected";
  declare readonly packageName: string;
  declare readonly functionName: string;
  declare readonly serverError?: string;
}

const InvokeHttpFailureBase: TaggedBaseConstructor = tagged(errorsTag("InvokeHttpFailure"));
export class InvokeHttpFailure extends InvokeHttpFailureBase {
  declare readonly _tag: "@forst/errors/InvokeHttpFailure";
  declare readonly packageName: string;
  declare readonly functionName: string;
  declare readonly status: number;
  declare readonly responseText: string;
}

const InvokeTimedOutBase: TaggedBaseConstructor = tagged(errorsTag("InvokeTimedOut"));
export class InvokeTimedOut extends InvokeTimedOutBase {
  declare readonly _tag: "@forst/errors/InvokeTimedOut";
  declare readonly packageName: string;
  declare readonly functionName: string;
  declare readonly timeoutMs?: number;
}

const InvokeUnreachableBase: TaggedBaseConstructor = tagged(errorsTag("InvokeUnreachable"));
export class InvokeUnreachable extends InvokeUnreachableBase {
  declare readonly _tag: "@forst/errors/InvokeUnreachable";
  declare readonly packageName: string;
  declare readonly functionName: string;
  declare readonly baseUrl: string;
}

const InvokeBaseUrlMissingBase: TaggedBaseConstructor = tagged(errorsTag("InvokeBaseUrlMissing"));
export class InvokeBaseUrlMissing extends InvokeBaseUrlMissingBase {
  declare readonly _tag: "@forst/errors/InvokeBaseUrlMissing";
  declare readonly envVar: string;
  declare readonly nodeEnv: string;
}

const InvokeStreamAbortedBase: TaggedBaseConstructor = tagged(errorsTag("InvokeStreamAborted"));
export class InvokeStreamAborted extends InvokeStreamAbortedBase {
  declare readonly _tag: "@forst/errors/InvokeStreamAborted";
  declare readonly packageName: string;
  declare readonly functionName: string;
  declare readonly rowIndex: number;
}

const ContractVersionMismatchBase: TaggedBaseConstructor = tagged(
  errorsTag("ContractVersionMismatch"),
);
export class ContractVersionMismatch extends ContractVersionMismatchBase {
  declare readonly _tag: "@forst/errors/ContractVersionMismatch";
  declare readonly expectedContractVersion: string;
  declare readonly serverContractVersion: string;
}

const ForstTestServerFailedBase: TaggedBaseConstructor = tagged(errorsTag("ForstTestServerFailed"));
export class ForstTestServerFailed extends ForstTestServerFailedBase {
  declare readonly _tag: "@forst/errors/ForstTestServerFailed";
  declare readonly reason: "cli_missing" | "spawn_failed" | "ready_timeout" | "unreachable";
  declare readonly installCommand?: string;
  declare readonly causeMessage?: string;
}

const ForstUnknownFailureBase: TaggedBaseConstructor = tagged(errorsTag("ForstUnknownFailure"));
export class ForstUnknownFailure extends ForstUnknownFailureBase {
  declare readonly _tag: "@forst/errors/ForstUnknownFailure";
  declare readonly message: string;
  declare readonly serverError?: string;
  declare readonly tag?: string;
  declare readonly packageName?: string;
  declare readonly functionName?: string;
}

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

/** Narrow a thrown value to an invoke transport failure. Prefer `_tag` matching in Effect code. */
export const isInvokeFailure = (u: unknown): u is InvokeFailure =>
  u instanceof Error && INVOKE_FAILURE_TAGS.has((u as { _tag?: string })._tag ?? "");

import { ERRORS_TAG_PREFIX, errorsTag } from "./tags.js";

export { ERRORS_TAG_PREFIX, errorsTag };

/** Promise-mode tagged error helper used by generated Forst clients. */
export const tagged = (tag: string) =>
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

export class InvokeRejected extends tagged(errorsTag("InvokeRejected")) {
  declare readonly _tag: "@forst/errors/InvokeRejected";
  declare readonly packageName: string;
  declare readonly functionName: string;
  declare readonly serverError?: string;
}

export class InvokeHttpFailure extends tagged(errorsTag("InvokeHttpFailure")) {
  declare readonly _tag: "@forst/errors/InvokeHttpFailure";
  declare readonly packageName: string;
  declare readonly functionName: string;
  declare readonly status: number;
  declare readonly responseText: string;
}

export class InvokeTimedOut extends tagged(errorsTag("InvokeTimedOut")) {
  declare readonly _tag: "@forst/errors/InvokeTimedOut";
  declare readonly packageName: string;
  declare readonly functionName: string;
  declare readonly timeoutMs?: number;
}

export class InvokeUnreachable extends tagged(errorsTag("InvokeUnreachable")) {
  declare readonly _tag: "@forst/errors/InvokeUnreachable";
  declare readonly packageName: string;
  declare readonly functionName: string;
  declare readonly baseUrl: string;
}

export class InvokeBaseUrlMissing extends tagged(errorsTag("InvokeBaseUrlMissing")) {
  declare readonly _tag: "@forst/errors/InvokeBaseUrlMissing";
  declare readonly envVar: string;
  declare readonly nodeEnv: string;
}

export class InvokeStreamAborted extends tagged(errorsTag("InvokeStreamAborted")) {
  declare readonly _tag: "@forst/errors/InvokeStreamAborted";
  declare readonly packageName: string;
  declare readonly functionName: string;
  declare readonly rowIndex: number;
}

export class ContractVersionMismatch extends tagged(errorsTag("ContractVersionMismatch")) {
  declare readonly _tag: "@forst/errors/ContractVersionMismatch";
  declare readonly expectedContractVersion: string;
  declare readonly serverContractVersion: string;
}

export class ForstTestServerFailed extends tagged(errorsTag("ForstTestServerFailed")) {
  declare readonly _tag: "@forst/errors/ForstTestServerFailed";
  declare readonly reason: "cli_missing" | "spawn_failed" | "ready_timeout" | "unreachable";
  declare readonly installCommand?: string;
  declare readonly causeMessage?: string;
}

export class ForstUnknownFailure extends tagged(errorsTag("ForstUnknownFailure")) {
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

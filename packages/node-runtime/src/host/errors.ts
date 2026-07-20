import { Data } from "effect";

type TaggedErrorCtor<Tag extends string> = new <
  A extends Record<string, unknown> = {}
>(
  args: A
) => Error & { readonly _tag: Tag } & Readonly<A>;

const HostSocketRequiredErrorBase = Data.TaggedError(
  "HostSocketRequiredError"
) as TaggedErrorCtor<"HostSocketRequiredError">;
/** Raised when `FORST_NODE_SOCKET` is unset on Unix. */
export class HostSocketRequiredError extends HostSocketRequiredErrorBase<{
  readonly message: "FORST_NODE_SOCKET is required on Unix";
}> {}

const HostReadyPathUnsetErrorBase = Data.TaggedError(
  "HostReadyPathUnsetError"
) as TaggedErrorCtor<"HostReadyPathUnsetError">;
/** Raised when {@link signalForstAppReady} runs before the host is started. */
export class HostReadyPathUnsetError extends HostReadyPathUnsetErrorBase<{
  readonly message: "signalForstAppReady: host not started or ready path unset; call startForstNodeHost first";
}> {}

const HostSocketPathUnsetErrorBase = Data.TaggedError(
  "HostSocketPathUnsetError"
) as TaggedErrorCtor<"HostSocketPathUnsetError">;
/** Raised when the host socket path is unset during app-ready signaling. */
export class HostSocketPathUnsetError extends HostSocketPathUnsetErrorBase<{
  readonly message: "signalForstAppReady: host socket path unset";
}> {}

/** Builds a {@link HostSocketRequiredError}. */
export function hostSocketRequired(): HostSocketRequiredError {
  return new HostSocketRequiredError({ message: "FORST_NODE_SOCKET is required on Unix" });
}

/** Builds a {@link HostReadyPathUnsetError}. */
export function hostReadyPathUnset(): HostReadyPathUnsetError {
  return new HostReadyPathUnsetError({
    message:
      "signalForstAppReady: host not started or ready path unset; call startForstNodeHost first",
  });
}

/** Builds a {@link HostSocketPathUnsetError}. */
export function hostSocketPathUnset(): HostSocketPathUnsetError {
  return new HostSocketPathUnsetError({
    message: "signalForstAppReady: host socket path unset",
  });
}

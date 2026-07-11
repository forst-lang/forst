import { Data } from "effect";

export class HostSocketRequiredError extends Data.TaggedError("HostSocketRequiredError")<{
  readonly message: "FORST_NODE_SOCKET is required on Unix";
}> {}

export class HostReadyPathUnsetError extends Data.TaggedError("HostReadyPathUnsetError")<{
  readonly message: "signalForstAppReady: host not started or ready path unset; call startForstNodeHost first";
}> {}

export class HostSocketPathUnsetError extends Data.TaggedError("HostSocketPathUnsetError")<{
  readonly message: "signalForstAppReady: host socket path unset";
}> {}

export function hostSocketRequired(): HostSocketRequiredError {
  return new HostSocketRequiredError({ message: "FORST_NODE_SOCKET is required on Unix" });
}

export function hostReadyPathUnset(): HostReadyPathUnsetError {
  return new HostReadyPathUnsetError({
    message:
      "signalForstAppReady: host not started or ready path unset; call startForstNodeHost first",
  });
}

export function hostSocketPathUnset(): HostSocketPathUnsetError {
  return new HostSocketPathUnsetError({
    message: "signalForstAppReady: host socket path unset",
  });
}

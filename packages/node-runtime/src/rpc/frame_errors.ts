import { Data } from "effect";

/** Thrown during protobuf decode when a varint exceeds the supported bit width. */
export class ProtoVarintOverflowError extends Data.TaggedError("ProtoVarintOverflowError")<{
  readonly message: "varint overflow";
}> {}

/** Thrown when a varint field ends before its continuation bit clears (truncated stream). */
export class ProtoTruncatedVarintError extends Data.TaggedError("ProtoTruncatedVarintError")<{
  readonly message: "truncated varint";
}> {}

/** Thrown when a length-delimited protobuf field claims more bytes than remain in the buffer. */
export class ProtoTruncatedLengthDelimitedFieldError extends Data.TaggedError(
  "ProtoTruncatedLengthDelimitedFieldError"
)<{
  readonly message: "truncated length-delimited field";
}> {}

/** Thrown when WireRequest/WireResponse fields use a wire type incompatible with the proto schema. */
export class ProtoUnexpectedWireTypeError extends Data.TaggedError(
  "ProtoUnexpectedWireTypeError"
)<{
  readonly container: "WireRequest" | "WireResponse";
  readonly wireType: number;
  readonly message: string;
}> {}

/** Thrown when ErrorDetail/Frame containers encounter wire types the decoder does not handle. */
export class ProtoUnsupportedWireTypeError extends Data.TaggedError(
  "ProtoUnsupportedWireTypeError"
)<{
  readonly container: "ErrorDetail" | "Frame";
  readonly wireType: number;
  readonly message: string;
}> {}

/** Thrown when an encoded frame body exceeds the configured max size (DoS guard on the bridge). */
export class ProtoFrameTooLargeError extends Data.TaggedError("ProtoFrameTooLargeError")<{
  readonly maxLen: number;
  readonly message: string;
}> {}

/** Thrown when the 4-byte length prefix is zero (invalid framing on the Go<->Node stream). */
export class ProtoEmptyFrameError extends Data.TaggedError("ProtoEmptyFrameError")<{
  readonly message: "empty proto frame";
}> {}

/** Factory for varint overflow failures during manual protobuf parsing. */
export function protoVarintOverflow(): ProtoVarintOverflowError {
  return new ProtoVarintOverflowError({ message: "varint overflow" });
}

/** Factory for truncated varint failures (incomplete byte stream). */
export function protoTruncatedVarint(): ProtoTruncatedVarintError {
  return new ProtoTruncatedVarintError({ message: "truncated varint" });
}

/** Factory for truncated length-delimited field failures. */
export function protoTruncatedLengthDelimitedField(): ProtoTruncatedLengthDelimitedFieldError {
  return new ProtoTruncatedLengthDelimitedFieldError({
    message: "truncated length-delimited field",
  });
}

/** Factory for unexpected wire type in request/response sub-messages. */
export function protoUnexpectedWireType(
  container: "WireRequest" | "WireResponse",
  wireType: number
): ProtoUnexpectedWireTypeError {
  return new ProtoUnexpectedWireTypeError({
    container,
    wireType,
    message: `unexpected wire type ${wireType} in ${container}`,
  });
}

/** Factory for unsupported wire type in top-level frame or error detail parsing. */
export function protoUnsupportedWireType(
  container: "ErrorDetail" | "Frame",
  wireType: number
): ProtoUnsupportedWireTypeError {
  return new ProtoUnsupportedWireTypeError({
    container,
    wireType,
    message: `unsupported wire type ${wireType} in ${container}`,
  });
}

/** Factory for oversize frame bodies rejected before decode (matches Go host limits). */
export function protoFrameTooLarge(maxLen: number): ProtoFrameTooLargeError {
  return new ProtoFrameTooLargeError({
    maxLen,
    message: `proto frame exceeds max size ${maxLen} bytes`,
  });
}

/** Factory for zero-length frame bodies (protocol violation on the length-prefixed stream). */
export function protoEmptyFrame(): ProtoEmptyFrameError {
  return new ProtoEmptyFrameError({ message: "empty proto frame" });
}

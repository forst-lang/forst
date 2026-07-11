import { Data } from "effect";

export class ProtoVarintOverflowError extends Data.TaggedError("ProtoVarintOverflowError")<{
  readonly message: "varint overflow";
}> {}

export class ProtoTruncatedVarintError extends Data.TaggedError("ProtoTruncatedVarintError")<{
  readonly message: "truncated varint";
}> {}

export class ProtoTruncatedLengthDelimitedFieldError extends Data.TaggedError(
  "ProtoTruncatedLengthDelimitedFieldError"
)<{
  readonly message: "truncated length-delimited field";
}> {}

export class ProtoUnexpectedWireTypeError extends Data.TaggedError(
  "ProtoUnexpectedWireTypeError"
)<{
  readonly container: "WireRequest" | "WireResponse";
  readonly wireType: number;
  readonly message: string;
}> {}

export class ProtoUnsupportedWireTypeError extends Data.TaggedError(
  "ProtoUnsupportedWireTypeError"
)<{
  readonly container: "ErrorDetail" | "Frame";
  readonly wireType: number;
  readonly message: string;
}> {}

export class ProtoFrameTooLargeError extends Data.TaggedError("ProtoFrameTooLargeError")<{
  readonly maxLen: number;
  readonly message: string;
}> {}

export class ProtoEmptyFrameError extends Data.TaggedError("ProtoEmptyFrameError")<{
  readonly message: "empty proto frame";
}> {}

export function protoVarintOverflow(): ProtoVarintOverflowError {
  return new ProtoVarintOverflowError({ message: "varint overflow" });
}

export function protoTruncatedVarint(): ProtoTruncatedVarintError {
  return new ProtoTruncatedVarintError({ message: "truncated varint" });
}

export function protoTruncatedLengthDelimitedField(): ProtoTruncatedLengthDelimitedFieldError {
  return new ProtoTruncatedLengthDelimitedFieldError({
    message: "truncated length-delimited field",
  });
}

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

export function protoFrameTooLarge(maxLen: number): ProtoFrameTooLargeError {
  return new ProtoFrameTooLargeError({
    maxLen,
    message: `proto frame exceeds max size ${maxLen} bytes`,
  });
}

export function protoEmptyFrame(): ProtoEmptyFrameError {
  return new ProtoEmptyFrameError({ message: "empty proto frame" });
}

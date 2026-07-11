import { Data } from "effect";

export class CliMissingRootValueError extends Data.TaggedError("CliMissingRootValueError")<{
  readonly message: "missing value for --root";
}> {}

export class CliMissingFormatValueError extends Data.TaggedError("CliMissingFormatValueError")<{
  readonly message: "missing value for --format";
}> {}

export class CliMissingFilesValueError extends Data.TaggedError("CliMissingFilesValueError")<{
  readonly message: "missing value for --files";
}> {}

export class CliUnknownArgumentError extends Data.TaggedError("CliUnknownArgumentError")<{
  readonly arg: string;
  readonly message: string;
}> {}

export class CliFilesRequiredError extends Data.TaggedError("CliFilesRequiredError")<{
  readonly message: "at least one file is required via --files";
}> {}

export class CliUnsupportedFormatError extends Data.TaggedError("CliUnsupportedFormatError")<{
  readonly format: string;
  readonly message: string;
}> {}

export function cliMissingRootValue(): CliMissingRootValueError {
  return new CliMissingRootValueError({ message: "missing value for --root" });
}

export function cliMissingFormatValue(): CliMissingFormatValueError {
  return new CliMissingFormatValueError({ message: "missing value for --format" });
}

export function cliMissingFilesValue(): CliMissingFilesValueError {
  return new CliMissingFilesValueError({ message: "missing value for --files" });
}

export function cliUnknownArgument(arg: string): CliUnknownArgumentError {
  return new CliUnknownArgumentError({ arg, message: `unknown argument: ${arg}` });
}

export function cliFilesRequired(): CliFilesRequiredError {
  return new CliFilesRequiredError({
    message: "at least one file is required via --files",
  });
}

export function cliUnsupportedFormat(format: string): CliUnsupportedFormatError {
  return new CliUnsupportedFormatError({
    format,
    message: `unsupported format: ${format}`,
  });
}

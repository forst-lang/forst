import { Data } from "effect";

/** Thrown when `--root` is present but has no following value. */
export class CliMissingRootValueError extends Data.TaggedError("CliMissingRootValueError")<{
  readonly message: "missing value for --root";
}> {}

/** Thrown when `--format` is present but has no following value. */
export class CliMissingFormatValueError extends Data.TaggedError("CliMissingFormatValueError")<{
  readonly message: "missing value for --format";
}> {}

/** Thrown when `--files` is present but has no following value. */
export class CliMissingFilesValueError extends Data.TaggedError("CliMissingFilesValueError")<{
  readonly message: "missing value for --files";
}> {}

/** Thrown when argv contains a flag the indexer CLI does not recognize. */
export class CliUnknownArgumentError extends Data.TaggedError("CliUnknownArgumentError")<{
  readonly arg: string;
  readonly message: string;
}> {}

/** Thrown when indexing is requested without at least one `--files` entry. */
export class CliFilesRequiredError extends Data.TaggedError("CliFilesRequiredError")<{
  readonly message: "at least one file is required via --files";
}> {}

/** Thrown when `--format` names an output format the indexer cannot emit. */
export class CliUnsupportedFormatError extends Data.TaggedError("CliUnsupportedFormatError")<{
  readonly format: string;
  readonly message: string;
}> {}

/** Builds a typed error for a missing `--root` value. */
export function cliMissingRootValue(): CliMissingRootValueError {
  return new CliMissingRootValueError({ message: "missing value for --root" });
}

/** Builds a typed error for a missing `--format` value. */
export function cliMissingFormatValue(): CliMissingFormatValueError {
  return new CliMissingFormatValueError({ message: "missing value for --format" });
}

/** Builds a typed error for a missing `--files` value. */
export function cliMissingFilesValue(): CliMissingFilesValueError {
  return new CliMissingFilesValueError({ message: "missing value for --files" });
}

/** Builds a typed error for an unrecognized CLI argument. */
export function cliUnknownArgument(arg: string): CliUnknownArgumentError {
  return new CliUnknownArgumentError({ arg, message: `unknown argument: ${arg}` });
}

/** Builds a typed error when no input files were provided. */
export function cliFilesRequired(): CliFilesRequiredError {
  return new CliFilesRequiredError({
    message: "at least one file is required via --files",
  });
}

/** Builds a typed error for an unsupported `--format` value. */
export function cliUnsupportedFormat(format: string): CliUnsupportedFormatError {
  return new CliUnsupportedFormatError({
    format,
    message: `unsupported format: ${format}`,
  });
}

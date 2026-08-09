package transformerts

import (
	"fmt"
	"strings"
)

// ErrorsPackageName is the npm package holding shared invoke/harness/unknown failure classes.
const ErrorsPackageName = "@forst/errors"

// ErrorsDependencyRange is the semver range written to generated client package.json dependencies.
const ErrorsDependencyRange = ">=0.1.0"

// ErrorsModuleSpecifier is the relative import path for the generated errors aggregator (domain + re-exports).
const ErrorsModuleSpecifier = "./errors.js"

// ReservedGeneratePackageName is the npm package name reserved for the shared error catalog.
const ReservedGeneratePackageName = "@forst/errors"

// ErrorField describes one constructor prop on a tagged invoke failure class.
type ErrorField struct {
	Name     string
	TSType   string
	Optional bool
}

// ErrorClass is one tagged failure class in generated client error modules.
type ErrorClass struct {
	Name    string
	Tag     string // client _tag value
	WireTag string // wire protocol lookup key; defaults to Name when empty
	Fields  []ErrorField
}

// ErrorCatalog is the single source of truth for invoke transport error classes.
// Class names never end in "Error" (sidecar condition naming).
var ErrorCatalog = []ErrorClass{
	{
		Name: "InvokeRejected",
		Tag:  "InvokeRejected",
		Fields: []ErrorField{
			{Name: "packageName", TSType: "string"},
			{Name: "functionName", TSType: "string"},
			{Name: "serverError", TSType: "string", Optional: true},
		},
	},
	{
		Name: "InvokeHttpFailure",
		Tag:  "InvokeHttpFailure",
		Fields: []ErrorField{
			{Name: "packageName", TSType: "string"},
			{Name: "functionName", TSType: "string"},
			{Name: "status", TSType: "number"},
			{Name: "responseText", TSType: "string"},
		},
	},
	{
		Name: "InvokeTimedOut",
		Tag:  "InvokeTimedOut",
		Fields: []ErrorField{
			{Name: "packageName", TSType: "string"},
			{Name: "functionName", TSType: "string"},
			{Name: "timeoutMs", TSType: "number", Optional: true},
		},
	},
	{
		Name: "InvokeUnreachable",
		Tag:  "InvokeUnreachable",
		Fields: []ErrorField{
			{Name: "packageName", TSType: "string"},
			{Name: "functionName", TSType: "string"},
			{Name: "baseUrl", TSType: "string"},
		},
	},
	{
		Name: "InvokeBaseUrlMissing",
		Tag:  "InvokeBaseUrlMissing",
		Fields: []ErrorField{
			{Name: "envVar", TSType: "string"},
			{Name: "nodeEnv", TSType: "string"},
		},
	},
	{
		Name: "InvokeStreamAborted",
		Tag:  "InvokeStreamAborted",
		Fields: []ErrorField{
			{Name: "packageName", TSType: "string"},
			{Name: "functionName", TSType: "string"},
			{Name: "rowIndex", TSType: "number"},
		},
	},
	{
		Name: "ContractVersionMismatch",
		Tag:  "ContractVersionMismatch",
		Fields: []ErrorField{
			{Name: "expectedContractVersion", TSType: "string"},
			{Name: "serverContractVersion", TSType: "string"},
		},
	},
}

// HarnessErrorCatalog describes test harness failures (re-exported from @forst/errors).
var HarnessErrorCatalog = []ErrorClass{
	{
		Name: "ForstTestServerFailed",
		Tag:  "ForstTestServerFailed",
		Fields: []ErrorField{
			{Name: "reason", TSType: "\"cli_missing\" | \"spawn_failed\" | \"ready_timeout\" | \"unreachable\""},
			{Name: "installCommand", TSType: "string", Optional: true},
			{Name: "causeMessage", TSType: "string", Optional: true},
		},
	},
}

func errorsPackageImport(runtime ClientRuntime) string {
	if runtime == RuntimeEffect {
		return ErrorsPackageName + "/effect"
	}
	return ErrorsPackageName
}

func clientTagPrefix(npmPackageName string) string {
	return effectTagPrefix(npmPackageName)
}

func namespacedClientTag(prefix, shortName string) string {
	return prefix + "/" + shortName
}

func domainWireTag(c ErrorClass) string {
	if c.WireTag != "" {
		return c.WireTag
	}
	if c.Tag != "" && !strings.HasPrefix(c.Tag, "@") {
		return c.Tag
	}
	return c.Name
}

func domainErrorsWithClientTags(npmPackageName string, domainErrors []ErrorClass) []ErrorClass {
	prefix := clientTagPrefix(npmPackageName)
	out := make([]ErrorClass, len(domainErrors))
	for i, c := range domainErrors {
		out[i] = c
		out[i].WireTag = domainWireTag(c)
		out[i].Tag = namespacedClientTag(prefix, c.Name)
	}
	return out
}

// ErrorClassNames returns every invoke failure class name from ErrorCatalog in catalog order.
func ErrorClassNames() []string {
	names := make([]string, len(ErrorCatalog))
	for i, c := range ErrorCatalog {
		names[i] = c.Name
	}
	return names
}

// HarnessErrorClassNames returns harness-only error class names (not in InvokeFailure).
func HarnessErrorClassNames() []string {
	names := make([]string, len(HarnessErrorCatalog))
	for i, c := range HarnessErrorCatalog {
		names[i] = c.Name
	}
	return names
}

// ReservedClientErrorNames returns class names that domain errors must not use.
func ReservedClientErrorNames() []string {
	names := append([]string(nil), ErrorClassNames()...)
	names = append(names, HarnessErrorClassNames()...)
	names = append(names, UnknownFailureClass.Name)
	return sortDedupeStrings(names)
}

// ValidateDomainErrors rejects nominal errors whose names collide with client error classes.
func ValidateDomainErrors(domainErrors []ErrorClass) error {
	reserved := make(map[string]struct{}, len(ReservedClientErrorNames()))
	for _, n := range ReservedClientErrorNames() {
		reserved[n] = struct{}{}
	}
	for _, c := range domainErrors {
		if _, ok := reserved[c.Name]; ok {
			return fmt.Errorf("domain error %q conflicts with reserved Forst client error name", c.Name)
		}
	}
	return nil
}

// RootReexportedDomainErrorNames returns domain error classes re-exported from dist/index.*
func RootReexportedDomainErrorNames(domainErrors []ErrorClass) []string {
	domainErrors = MergeDomainErrors(domainErrors)
	names := make([]string, 0, len(domainErrors)+1)
	for _, c := range domainErrors {
		names = append(names, c.Name)
	}
	names = append(names, UnknownFailureClass.Name)
	return sortDedupeStrings(names)
}

func writeTaggedHelperJS(b *strings.Builder) {
	b.WriteString(`const tagged = (tag) =>
  class extends Error {
    constructor(props) {
      super(props?.message ?? tag);
      Object.assign(this, props);
      Object.defineProperty(this, "_tag", {
        value: tag,
        enumerable: true,
        writable: false,
      });
      this.name = tag;
      Object.setPrototypeOf(this, new.target.prototype);
      if (Error.captureStackTrace) Error.captureStackTrace(this, new.target);
    }
  };

`)
}

func emitErrorClassDTS(b *strings.Builder, c ErrorClass) {
	fmt.Fprintf(b, "export declare class %s extends Error {\n", c.Name)
	fmt.Fprintf(b, "  readonly _tag: %q;\n", c.Tag)
	for _, f := range c.Fields {
		if f.Name == "message" {
			continue
		}
		opt := ""
		if f.Optional {
			opt = " | undefined"
		}
		fmt.Fprintf(b, "  readonly %s: %s%s;\n", f.Name, f.TSType, opt)
	}
	b.WriteString("  constructor(props: {\n")
	for _, f := range c.Fields {
		if f.Name == "message" {
			continue
		}
		optMark := ""
		if f.Optional {
			optMark = "?"
		}
		fmt.Fprintf(b, "    readonly %s%s: %s;\n", f.Name, optMark, f.TSType)
	}
	b.WriteString("    readonly message?: string;\n")
	b.WriteString("  });\n")
	b.WriteString("}\n\n")
}

func emitErrorClassESM(b *strings.Builder, c ErrorClass) {
	fmt.Fprintf(b, "export class %s extends tagged(%q) {}\n", c.Name, c.Tag)
}

func writeEffectImport(b *strings.Builder) {
	b.WriteString(`import { Data } from "effect";

`)
}

func emitEffectErrorClassFields(b *strings.Builder, c ErrorClass) {
	b.WriteString("{\n")
	for _, f := range c.Fields {
		if f.Name == "message" {
			continue
		}
		optMark := ""
		if f.Optional {
			optMark = "?"
		}
		fmt.Fprintf(b, "  readonly %s%s: %s;\n", f.Name, optMark, f.TSType)
	}
	b.WriteString("  readonly message?: string;\n")
	b.WriteString("}>")
}

func emitEffectErrorClassESM(b *strings.Builder, c ErrorClass) {
	fmt.Fprintf(b, "export class %s extends Data.TaggedError(%q) {}\n\n", c.Name, c.Tag)
}

func emitEffectErrorClassDTS(b *strings.Builder, c ErrorClass) {
	fmt.Fprintf(b, "export declare class %s extends Data.TaggedError(%q)<", c.Name, c.Tag)
	emitEffectErrorClassFields(b, c)
	b.WriteString(" {}\n\n")
}

func writeSharedErrorsImport(b *strings.Builder, runtime ClientRuntime) {
	fmt.Fprintf(b, "import { ForstUnknownFailure } from %q;\n\n", errorsPackageImport(runtime))
}

func writeErrorsAggregatorReexports(b *strings.Builder, runtime ClientRuntime) {
	pkg := errorsPackageImport(runtime)
	b.WriteString("export {\n")
	for _, name := range ErrorClassNames() {
		fmt.Fprintf(b, "  %s,\n", name)
	}
	b.WriteString("  isInvokeFailure,\n")
	b.WriteString("  ForstUnknownFailure,\n")
	for _, name := range HarnessErrorClassNames() {
		fmt.Fprintf(b, "  %s,\n", name)
	}
	fmt.Fprintf(b, "} from %q;\n\n", pkg)
}

func writeErrorsAggregatorReexportsDTS(b *strings.Builder, runtime ClientRuntime) {
	pkg := errorsPackageImport(runtime)
	b.WriteString("export {\n")
	for _, name := range ErrorClassNames() {
		fmt.Fprintf(b, "  %s,\n", name)
	}
	b.WriteString("  isInvokeFailure,\n")
	b.WriteString("  ForstUnknownFailure,\n")
	for _, name := range HarnessErrorClassNames() {
		fmt.Fprintf(b, "  %s,\n", name)
	}
	fmt.Fprintf(b, "} from %q;\n\n", pkg)
	fmt.Fprintf(b, "export type { InvokeFailure } from %q;\n", pkg)
}

// EmitErrorsESM returns dist/errors.js (domain errors + shared re-exports).
func EmitErrorsESM(npmPackageName string, domainErrors []ErrorClass, runtime ClientRuntime) string {
	domainErrors = domainErrorsWithClientTags(npmPackageName, MergeDomainErrors(domainErrors))
	var b strings.Builder
	b.WriteString(`// Auto-generated Forst client errors (domain + shared re-exports).
// Generated by Forst TypeScript Transformer.
// Do not edit by hand.

`)
	writeSharedErrorsImport(&b, runtime)
	if runtime == RuntimeEffect {
		if len(domainErrors) > 0 {
			writeEffectImport(&b)
			for _, c := range domainErrors {
				emitEffectErrorClassESM(&b, c)
			}
		}
	} else if len(domainErrors) > 0 {
		writeTaggedHelperJS(&b)
		for _, c := range domainErrors {
			emitErrorClassESM(&b, c)
		}
	}
	b.WriteString("\n")
	emitDomainRegistryJS(&b, domainErrors)
	b.WriteString("\n")
	writeErrorsAggregatorReexports(&b, runtime)
	return b.String()
}

func emitDomainRegistryJS(b *strings.Builder, domainErrors []ErrorClass) {
	b.WriteString("export const DOMAIN_ERROR_REGISTRY = {\n")
	for _, c := range domainErrors {
		wireTag := c.WireTag
		if wireTag == "" {
			wireTag = c.Name
		}
		fmt.Fprintf(b, "  %q: %s,\n", wireTag, c.Name)
	}
	b.WriteString("};\n\n")
	b.WriteString(`export const decodeDomainError = (errorValue, ctx = {}) => {
  const tag = errorValue?.tag;
  const Ctor = tag ? DOMAIN_ERROR_REGISTRY[tag] : undefined;
  const payload =
    errorValue?.payload && typeof errorValue.payload === "object"
      ? errorValue.payload
      : {};
  const base = {
    message: errorValue?.message ?? ctx.serverError ?? tag ?? "ForstUnknownFailure",
    serverError: ctx.serverError,
    packageName: ctx.packageName,
    functionName: ctx.functionName,
  };
  if (!Ctor) {
    return new ForstUnknownFailure({ ...base, tag });
  }
  return new Ctor({ ...payload, ...base });
};
`)
}

// EmitErrorsDTS returns dist/errors.d.ts.
func EmitErrorsDTS(npmPackageName string, domainErrors []ErrorClass, runtime ClientRuntime) string {
	domainErrors = domainErrorsWithClientTags(npmPackageName, MergeDomainErrors(domainErrors))
	var b strings.Builder
	b.WriteString(`// Auto-generated Forst client errors (domain + shared re-exports).
// Generated by Forst TypeScript Transformer.
// Do not edit by hand.

`)
	fmt.Fprintf(&b, "import type { ForstUnknownFailure } from %q;\n\n", errorsPackageImport(runtime))
	if runtime == RuntimeEffect {
		if len(domainErrors) > 0 {
			writeEffectImport(&b)
		}
	} else {
		b.WriteString(`export type TaggedError<
  Tag extends string,
  A extends Record<string, unknown> = {}
> = Error & { readonly _tag: Tag } & Readonly<A>;

`)
	}
	for _, c := range domainErrors {
		if runtime == RuntimeEffect {
			emitEffectErrorClassDTS(&b, c)
		} else {
			emitErrorClassDTS(&b, c)
		}
	}
	if len(domainErrors) > 0 {
		b.WriteString("export type ForstError =\n")
		for _, c := range domainErrors {
			fmt.Fprintf(&b, "  | %s\n", c.Name)
		}
		fmt.Fprintf(&b, "  | %s\n", UnknownFailureClass.Name)
		b.WriteString(";\n\n")
	} else {
		b.WriteString("export type ForstError = ForstUnknownFailure;\n\n")
	}
	b.WriteString("export declare const DOMAIN_ERROR_REGISTRY: Record<string, new (props: Record<string, unknown>) => Error>;\n\n")
	b.WriteString("export declare function decodeDomainError(\n")
	b.WriteString("  errorValue: { tag?: string; payload?: Record<string, unknown>; message?: string } | undefined,\n")
	b.WriteString("  ctx?: { packageName?: string; functionName?: string; serverError?: string }\n")
	b.WriteString("): ForstError;\n\n")
	writeErrorsAggregatorReexportsDTS(&b, runtime)
	return b.String()
}

// EmitDomainErrorsESM is an alias for EmitErrorsESM.
func EmitDomainErrorsESM(npmPackageName string, domainErrors []ErrorClass, runtime ClientRuntime) string {
	return EmitErrorsESM(npmPackageName, domainErrors, runtime)
}

// EmitDomainErrorsDTS is an alias for EmitErrorsDTS.
func EmitDomainErrorsDTS(npmPackageName string, domainErrors []ErrorClass, runtime ClientRuntime) string {
	return EmitErrorsDTS(npmPackageName, domainErrors, runtime)
}

func writeHarnessErrorReexportsESM(b *strings.Builder, runtime ClientRuntime) {
	pkg := errorsPackageImport(runtime)
	fmt.Fprintf(b, "import { ForstTestServerFailed } from %q;\n", pkg)
	b.WriteString("export { ForstTestServerFailed };\n")
}

func writeHarnessErrorReexportsDTS(b *strings.Builder, runtime ClientRuntime) {
	pkg := errorsPackageImport(runtime)
	fmt.Fprintf(b, "import type { ForstTestServerFailed } from %q;\n", pkg)
	fmt.Fprintf(b, "export { ForstTestServerFailed } from %q;\n", pkg)
}

// EmitHarnessErrorESM returns harness error re-exports for testing.js.
func EmitHarnessErrorESM(_ string, runtime ClientRuntime) string {
	if len(HarnessErrorCatalog) == 0 {
		return ""
	}
	var b strings.Builder
	writeHarnessErrorReexportsESM(&b, runtime)
	return b.String()
}

// EmitHarnessErrorDTS returns harness error re-export declarations for testing.d.ts.
func EmitHarnessErrorDTS(_ string, runtime ClientRuntime) string {
	if len(HarnessErrorCatalog) == 0 {
		return ""
	}
	var b strings.Builder
	writeHarnessErrorReexportsDTS(&b, runtime)
	return b.String()
}

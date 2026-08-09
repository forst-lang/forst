package transformerts

import (
	"fmt"
	"strings"
)

// DomainErrorsModuleSpecifier is the relative import path for generated domain errors.
const DomainErrorsModuleSpecifier = "./domain-errors.js"

// InvokeErrorsModuleSpecifier is the relative import path for generated invoke transport errors.
const InvokeErrorsModuleSpecifier = "./errors.js"

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

// ForstBuiltInTagPrefix is the namespace for generic Forst client errors (invoke, harness, unknown failure).
const ForstBuiltInTagPrefix = "@forst"

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

// HarnessErrorCatalog describes test harness failures (emitted in testing.js, not errors.js).
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

func clientTagPrefix(npmPackageName string) string {
	return effectTagPrefix(npmPackageName)
}

func namespacedClientTag(prefix, shortName string) string {
	return prefix + "/" + shortName
}

func forstBuiltInTag(shortName string) string {
	return ForstBuiltInTagPrefix + "/" + shortName
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

func invokeCatalogWithTags(_ string) []ErrorClass {
	out := make([]ErrorClass, len(ErrorCatalog))
	for i, c := range ErrorCatalog {
		out[i] = c
		out[i].Tag = forstBuiltInTag(c.Name)
	}
	return out
}

func unknownFailureWithTag(_ string) ErrorClass {
	c := UnknownFailureClass
	c.Tag = forstBuiltInTag(c.Name)
	return c
}

func harnessCatalogWithTags() []ErrorClass {
	out := make([]ErrorClass, len(HarnessErrorCatalog))
	for i, c := range HarnessErrorCatalog {
		out[i] = c
		out[i].Tag = forstBuiltInTag(c.Name)
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

// EmitDomainErrorsESM returns dist/domain-errors.js.
func EmitDomainErrorsESM(npmPackageName string, domainErrors []ErrorClass, runtime ClientRuntime) string {
	domainErrors = domainErrorsWithClientTags(npmPackageName, MergeDomainErrors(domainErrors))
	unknown := unknownFailureWithTag(npmPackageName)
	var b strings.Builder
	b.WriteString(`// Auto-generated Forst domain errors.
// Generated by Forst TypeScript Transformer.
// Do not edit by hand.

`)
	if runtime == RuntimeEffect {
		writeEffectImport(&b)
		for _, c := range domainErrors {
			emitEffectErrorClassESM(&b, c)
		}
		emitEffectErrorClassESM(&b, unknown)
	} else {
		writeTaggedHelperJS(&b)
		for _, c := range domainErrors {
			emitErrorClassESM(&b, c)
		}
		emitErrorClassESM(&b, unknown)
	}
	b.WriteString("\n")
	emitDomainRegistryJS(&b, domainErrors)
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

// EmitDomainErrorsDTS returns dist/domain-errors.d.ts.
func EmitDomainErrorsDTS(npmPackageName string, domainErrors []ErrorClass, runtime ClientRuntime) string {
	domainErrors = domainErrorsWithClientTags(npmPackageName, MergeDomainErrors(domainErrors))
	unknown := unknownFailureWithTag(npmPackageName)
	var b strings.Builder
	b.WriteString(`// Auto-generated Forst domain errors.
// Generated by Forst TypeScript Transformer.
// Do not edit by hand.

`)
	if runtime == RuntimeEffect {
		writeEffectImport(&b)
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
	if runtime == RuntimeEffect {
		emitEffectErrorClassDTS(&b, unknown)
	} else {
		emitErrorClassDTS(&b, unknown)
	}
	if len(domainErrors) > 0 {
		b.WriteString("export type ForstError =\n")
		for _, c := range domainErrors {
			fmt.Fprintf(&b, "  | %s\n", c.Name)
		}
		fmt.Fprintf(&b, "  | %s\n", unknown.Name)
		b.WriteString(";\n\n")
	} else {
		b.WriteString("export type ForstError = ForstUnknownFailure;\n\n")
	}
	b.WriteString("export declare const DOMAIN_ERROR_REGISTRY: Record<string, new (props: Record<string, unknown>) => Error>;\n\n")
	b.WriteString("export declare function decodeDomainError(\n")
	b.WriteString("  errorValue: { tag?: string; payload?: Record<string, unknown>; message?: string } | undefined,\n")
	b.WriteString("  ctx?: { packageName?: string; functionName?: string; serverError?: string }\n")
	b.WriteString("): ForstError;\n")
	return b.String()
}

// EmitInvokeErrorsESM returns dist/errors.js.
func EmitInvokeErrorsESM(npmPackageName string, runtime ClientRuntime) string {
	catalog := invokeCatalogWithTags(npmPackageName)
	var b strings.Builder
	b.WriteString(`// Auto-generated Forst invoke transport errors.
// Generated by Forst TypeScript Transformer.
// Do not edit by hand.

`)
	if runtime == RuntimeEffect {
		writeEffectImport(&b)
		for _, c := range catalog {
			emitEffectErrorClassESM(&b, c)
		}
	} else {
		writeTaggedHelperJS(&b)
		for _, c := range catalog {
			emitErrorClassESM(&b, c)
		}
	}
	b.WriteString("\n")
	b.WriteString("const INVOKE_FAILURE_TAGS = new Set([\n")
	for _, c := range catalog {
		fmt.Fprintf(&b, "  %q,\n", c.Tag)
	}
	b.WriteString("]);\n\n")
	b.WriteString("export const isInvokeFailure = (u) =>\n")
	b.WriteString("  u instanceof Error && INVOKE_FAILURE_TAGS.has(u._tag);\n")
	return b.String()
}

// EmitInvokeErrorsDTS returns dist/errors.d.ts.
func EmitInvokeErrorsDTS(npmPackageName string, runtime ClientRuntime) string {
	catalog := invokeCatalogWithTags(npmPackageName)
	var b strings.Builder
	b.WriteString(`// Auto-generated Forst invoke transport errors.
// Generated by Forst TypeScript Transformer.
// Do not edit by hand.

`)
	if runtime == RuntimeEffect {
		writeEffectImport(&b)
	}
	for _, c := range catalog {
		if runtime == RuntimeEffect {
			emitEffectErrorClassDTS(&b, c)
		} else {
			emitErrorClassDTS(&b, c)
		}
	}
	b.WriteString("export type InvokeFailure =\n")
	for _, c := range catalog {
		fmt.Fprintf(&b, "  | %s\n", c.Name)
	}
	b.WriteString(";\n\n")
	b.WriteString("export declare function isInvokeFailure(u: unknown): u is InvokeFailure;\n")
	return b.String()
}

// EmitHarnessErrorESM returns the harness error class definition for testing.js.
func EmitHarnessErrorESM(_ string, runtime ClientRuntime) string {
	catalog := harnessCatalogWithTags()
	if len(catalog) == 0 {
		return ""
	}
	var b strings.Builder
	if runtime == RuntimeEffect {
		writeEffectImport(&b)
		for _, c := range catalog {
			emitEffectErrorClassESM(&b, c)
		}
	} else {
		writeTaggedHelperJS(&b)
		for _, c := range catalog {
			emitErrorClassESM(&b, c)
		}
	}
	return b.String()
}

// EmitHarnessErrorDTS returns the harness error class declaration for testing.d.ts.
func EmitHarnessErrorDTS(_ string, runtime ClientRuntime) string {
	catalog := harnessCatalogWithTags()
	if len(catalog) == 0 {
		return ""
	}
	var b strings.Builder
	if runtime == RuntimeEffect {
		writeEffectImport(&b)
		for _, c := range catalog {
			emitEffectErrorClassDTS(&b, c)
		}
	} else {
		for _, c := range catalog {
			emitErrorClassDTS(&b, c)
		}
	}
	return b.String()
}

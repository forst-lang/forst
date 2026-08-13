package transformerts

import (
	"fmt"
	"sort"
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
	Name         string
	ForstPackage string // Forst package that owns this nominal error
	Tag          string // client _tag value
	WireTag      string // wire protocol lookup key; defaults to ForstPackage/Name when empty
	Fields       []ErrorField
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

func namespacedClientTag(prefix, forstPkg, shortName string) string {
	if forstPkg != "" {
		return prefix + "/" + forstPkg + "/" + shortName
	}
	return prefix + "/" + shortName
}

func domainWireTag(c ErrorClass) string {
	if c.WireTag != "" {
		return c.WireTag
	}
	if c.ForstPackage != "" {
		return c.ForstPackage + "/" + c.Name
	}
	if c.Tag != "" && !strings.HasPrefix(c.Tag, "@") {
		return c.Tag
	}
	return c.Name
}

func domainErrorsWithClientTags(npmPackageName string, domainErrors []ErrorClass) ([]ErrorClass, error) {
	prefix := clientTagPrefix(npmPackageName)
	out := make([]ErrorClass, len(domainErrors))
	for i, c := range domainErrors {
		if c.ForstPackage == "" {
			return nil, fmt.Errorf("domain error %q missing Forst package name", c.Name)
		}
		out[i] = c
		out[i].WireTag = domainWireTag(c)
		out[i].Tag = namespacedClientTag(prefix, c.ForstPackage, c.Name)
	}
	return out, nil
}

// PackageDomainErrorsFileStem returns the dist/pkg module stem for one package's domain errors.
func PackageDomainErrorsFileStem(forpstPkg string) string {
	return forpstPkg + ".errors"
}

// StampDomainErrorPackages sets ForstPackage on every domain error in out.
func StampDomainErrorPackages(out *TypeScriptOutput) error {
	if out == nil || len(out.DomainErrors) == 0 {
		return nil
	}
	pkg := out.PackageName
	if pkg == "" {
		pkg = out.SourceFileStem
	}
	if pkg == "" {
		return fmt.Errorf("domain errors require a Forst package name")
	}
	for i := range out.DomainErrors {
		out.DomainErrors[i].ForstPackage = pkg
	}
	return nil
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
// Only bare names unique across Forst packages are re-exported from the package root.
func RootReexportedDomainErrorNames(domainErrors []ErrorClass) []string {
	merged, err := MergeDomainErrors(domainErrors)
	if err != nil {
		return []string{UnknownFailureClass.Name}
	}
	counts := make(map[string]int, len(merged))
	for _, c := range merged {
		counts[c.Name]++
	}
	names := make([]string, 0, len(merged)+1)
	for _, c := range merged {
		if counts[c.Name] == 1 {
			names = append(names, c.Name)
		}
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

type domainErrorImportBinding struct {
	ClassName  string
	ImportName string
	ModulePath string
	WireTag    string
}

func domainErrorsGroupedByPackage(domainErrors []ErrorClass) map[string][]ErrorClass {
	byPkg := make(map[string][]ErrorClass)
	for _, c := range domainErrors {
		byPkg[c.ForstPackage] = append(byPkg[c.ForstPackage], c)
	}
	return byPkg
}

func domainErrorImportBindings(domainErrors []ErrorClass, modulePathFor func(forpstPkg string) string) []domainErrorImportBinding {
	nameCounts := make(map[string]int, len(domainErrors))
	for _, c := range domainErrors {
		nameCounts[c.Name]++
	}
	out := make([]domainErrorImportBinding, 0, len(domainErrors))
	for _, c := range domainErrors {
		importName := c.Name
		if nameCounts[c.Name] > 1 {
			importName = ServiceClassName(c.ForstPackage) + c.Name
		}
		out = append(out, domainErrorImportBinding{
			ClassName:  c.Name,
			ImportName: importName,
			ModulePath: modulePathFor(c.ForstPackage),
			WireTag:    domainWireTag(c),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].WireTag != out[j].WireTag {
			return out[i].WireTag < out[j].WireTag
		}
		return out[i].ImportName < out[j].ImportName
	})
	return out
}

func writeDomainErrorImports(b *strings.Builder, bindings []domainErrorImportBinding) {
	byModule := make(map[string][]domainErrorImportBinding)
	for _, binding := range bindings {
		byModule[binding.ModulePath] = append(byModule[binding.ModulePath], binding)
	}
	modules := make([]string, 0, len(byModule))
	for module := range byModule {
		modules = append(modules, module)
	}
	sort.Strings(modules)
	for _, module := range modules {
		parts := byModule[module]
		sort.Slice(parts, func(i, j int) bool {
			return parts[i].ImportName < parts[j].ImportName
		})
		specs := make([]string, 0, len(parts))
		for _, binding := range parts {
			if binding.ImportName == binding.ClassName {
				specs = append(specs, binding.ClassName)
			} else {
				specs = append(specs, binding.ClassName+" as "+binding.ImportName)
			}
		}
		fmt.Fprintf(b, "import { %s } from %q;\n", strings.Join(specs, ", "), module)
	}
}

func writeDomainErrorReexports(b *strings.Builder, bindings []domainErrorImportBinding) {
	if len(bindings) == 0 {
		return
	}
	b.WriteString("export {\n")
	for _, binding := range bindings {
		if binding.ImportName == binding.ClassName {
			fmt.Fprintf(b, "  %s,\n", binding.ClassName)
		} else {
			fmt.Fprintf(b, "  %s as %s,\n", binding.ImportName, binding.ClassName)
		}
	}
	b.WriteString("};\n\n")
}

func emitDomainErrorClasses(b *strings.Builder, domainErrors []ErrorClass, runtime ClientRuntime) {
	if len(domainErrors) == 0 {
		return
	}
	if runtime == RuntimeEffect {
		writeEffectImport(b)
		for _, c := range domainErrors {
			emitEffectErrorClassESM(b, c)
		}
		return
	}
	writeTaggedHelperJS(b)
	for _, c := range domainErrors {
		emitErrorClassESM(b, c)
	}
}

func emitDomainErrorClassDeclarations(b *strings.Builder, domainErrors []ErrorClass, runtime ClientRuntime) {
	for _, c := range domainErrors {
		if runtime == RuntimeEffect {
			emitEffectErrorClassDTS(b, c)
		} else {
			emitErrorClassDTS(b, c)
		}
	}
}

// EmitPackageDomainErrorsESM returns dist/pkg/<pkg>.errors.js for one Forst package.
func EmitPackageDomainErrorsESM(npmPackageName, forstPkg string, domainErrors []ErrorClass, runtime ClientRuntime) (string, error) {
	tagged, err := domainErrorsWithClientTags(npmPackageName, domainErrors)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString(`// Auto-generated Forst domain errors for package ` + forstPkg + `.
// Generated by Forst TypeScript Transformer.
// Do not edit by hand.

`)
	emitDomainErrorClasses(&b, tagged, runtime)
	return b.String(), nil
}

// EmitPackageDomainErrorsDTS returns dist/pkg/<pkg>.errors.d.ts for one Forst package.
func EmitPackageDomainErrorsDTS(npmPackageName, forstPkg string, domainErrors []ErrorClass, runtime ClientRuntime) (string, error) {
	tagged, err := domainErrorsWithClientTags(npmPackageName, domainErrors)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString(`// Auto-generated Forst domain errors for package ` + forstPkg + `.
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
	emitDomainErrorClassDeclarations(&b, tagged, runtime)
	return b.String(), nil
}

// EmitErrorsESM returns dist/errors.js (domain errors + shared re-exports).
func EmitErrorsESM(npmPackageName string, domainErrors []ErrorClass, runtime ClientRuntime) (string, error) {
	merged, err := MergeDomainErrors(domainErrors)
	if err != nil {
		return "", err
	}
	tagged, err := domainErrorsWithClientTags(npmPackageName, merged)
	if err != nil {
		return "", err
	}
	bindings := domainErrorImportBindings(tagged, func(forpstPkg string) string {
		return "./pkg/" + PackageDomainErrorsFileStem(forpstPkg) + ".js"
	})
	var b strings.Builder
	b.WriteString(`// Auto-generated Forst client errors (domain + shared re-exports).
// Generated by Forst TypeScript Transformer.
// Do not edit by hand.

`)
	writeSharedErrorsImport(&b, runtime)
	if len(bindings) > 0 {
		b.WriteString("\n")
		writeDomainErrorImports(&b, bindings)
		b.WriteString("\n")
	}
	emitDomainRegistryJSWithBindings(&b, bindings)
	b.WriteString("\n")
	if len(bindings) > 0 {
		writeDomainErrorReexports(&b, bindings)
	}
	writeErrorsAggregatorReexports(&b, runtime)
	return b.String(), nil
}

func emitDomainRegistryJSWithBindings(b *strings.Builder, bindings []domainErrorImportBinding) {
	b.WriteString("export const DOMAIN_ERROR_REGISTRY = {\n")
	for _, binding := range bindings {
		fmt.Fprintf(b, "  %q: %s,\n", binding.WireTag, binding.ImportName)
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

func emitDomainRegistryJS(b *strings.Builder, domainErrors []ErrorClass) {
	bindings := make([]domainErrorImportBinding, 0, len(domainErrors))
	for _, c := range domainErrors {
		bindings = append(bindings, domainErrorImportBinding{
			ClassName:  c.Name,
			ImportName: c.Name,
			WireTag:      domainWireTag(c),
		})
	}
	emitDomainRegistryJSWithBindings(b, bindings)
}

// EmitErrorsDTS returns dist/errors.d.ts.
func EmitErrorsDTS(npmPackageName string, domainErrors []ErrorClass, runtime ClientRuntime) (string, error) {
	merged, err := MergeDomainErrors(domainErrors)
	if err != nil {
		return "", err
	}
	tagged, err := domainErrorsWithClientTags(npmPackageName, merged)
	if err != nil {
		return "", err
	}
	bindings := domainErrorImportBindings(tagged, func(forpstPkg string) string {
		return "./pkg/" + PackageDomainErrorsFileStem(forpstPkg) + ".js"
	})
	var b strings.Builder
	b.WriteString(`// Auto-generated Forst client errors (domain + shared re-exports).
// Generated by Forst TypeScript Transformer.
// Do not edit by hand.

`)
	fmt.Fprintf(&b, "import type { ForstUnknownFailure } from %q;\n", errorsPackageImport(runtime))
	if len(bindings) > 0 {
		b.WriteString("\n")
		writeDomainErrorImports(&b, bindings)
		b.WriteString("\n")
	}
	if len(bindings) > 0 {
		b.WriteString("export type ForstError =\n")
		for _, binding := range bindings {
			fmt.Fprintf(&b, "  | %s\n", binding.ImportName)
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
	if len(bindings) > 0 {
		writeDomainErrorReexports(&b, bindings)
	}
	writeErrorsAggregatorReexportsDTS(&b, runtime)
	return b.String(), nil
}

// EmitDomainErrorsESM is an alias for EmitErrorsESM.
func EmitDomainErrorsESM(npmPackageName string, domainErrors []ErrorClass, runtime ClientRuntime) (string, error) {
	return EmitErrorsESM(npmPackageName, domainErrors, runtime)
}

// EmitDomainErrorsDTS is an alias for EmitErrorsDTS.
func EmitDomainErrorsDTS(npmPackageName string, domainErrors []ErrorClass, runtime ClientRuntime) (string, error) {
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

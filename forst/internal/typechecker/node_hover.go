package typechecker

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"forst/internal/hoverdoc"
	"forst/internal/nodeinterop"
)

// NodeHoverMarkdown returns hover text for a TypeScript export via node import (e.g. payment.create).
func (tc *TypeChecker) NodeHoverMarkdown(moduleLocal, exportName string) (string, bool) {
	if tc == nil || moduleLocal == "" || exportName == "" {
		return "", false
	}
	mod, ok := tc.nodeModuleForLocal(moduleLocal)
	if !ok || mod.Index == nil {
		return "", false
	}
	exp, ok := mod.Index.ExportByName(exportName)
	if !ok {
		return "", false
	}
	return formatNodeExportAliasHover(*exp), true
}

// NodeImportPathHoverMarkdown returns TS-style hover for a node import path string literal.
func (tc *TypeChecker) NodeImportPathHoverMarkdown(importPath string) (string, bool) {
	if tc == nil || strings.TrimSpace(importPath) == "" {
		return "", false
	}
	for _, binding := range tc.nodeImportsByLocal {
		if binding.Import.Path != importPath || binding.AbsPath == "" {
			continue
		}
		return formatNodeModulePathHover(binding.AbsPath), true
	}
	return "", false
}

// NodeModuleHoverMarkdown returns hover for a node import local name (e.g. payment).
func (tc *TypeChecker) NodeModuleHoverMarkdown(moduleLocal string) (string, bool) {
	if tc == nil || moduleLocal == "" {
		return "", false
	}
	if _, ok := tc.nodeModuleForLocal(moduleLocal); !ok {
		return "", false
	}
	return formatNodeModuleAliasHover(moduleLocal), true
}

func formatNodeModuleAliasHover(moduleLocal string) string {
	return hoverdoc.TypeScriptBlock("module " + moduleLocal)
}

func formatNodeModulePathHover(absPath string) string {
	return hoverdoc.TypeScriptModuleBlock(formatNodeModulePathDisplay(absPath))
}

func formatNodeExportAliasHover(exp nodeinterop.IndexExport) string {
	return hoverdoc.TypeScriptBlock("(alias) " + formatTSExportSignatureBody(exp))
}

func formatNodeModulePathDisplay(absPath string) string {
	absPath = filepath.Clean(absPath)
	for _, ext := range []string{".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs"} {
		if strings.HasSuffix(absPath, ext) {
			absPath = strings.TrimSuffix(absPath, ext)
			break
		}
	}
	return filepath.ToSlash(absPath)
}

func formatTSExportSignatureBody(exp nodeinterop.IndexExport) string {
	prefix := exportTSKeywordBody(exp.Kind)
	params := make([]string, 0, len(exp.Parameters))
	for _, p := range exp.Parameters {
		params = append(params, fmt.Sprintf("%s: %s", p.Name, formatIndexTypeTS(p.Type)))
	}
	sig := fmt.Sprintf("%s%s(%s)", prefix, exp.Name, strings.Join(params, ", "))
	ret := formatExportReturnTS(exp)
	if ret != "" {
		sig += ": " + ret
	}
	return sig
}

func exportTSKeywordBody(kind string) string {
	switch kind {
	case nodeinterop.ExportKindAsyncFunction:
		return "async function "
	case nodeinterop.ExportKindGenerator:
		return "function* "
	case nodeinterop.ExportKindAsyncGenerator:
		return "async function* "
	default:
		return "function "
	}
}

func formatExportReturnTS(exp nodeinterop.IndexExport) string {
	switch exp.Kind {
	case nodeinterop.ExportKindGenerator:
		return formatGeneratorReturnTS("Generator", exp)
	case nodeinterop.ExportKindAsyncGenerator:
		return formatGeneratorReturnTS("AsyncGenerator", exp)
	case nodeinterop.ExportKindAsyncFunction:
		if exp.ReturnType != nil {
			return "Promise<" + formatIndexTypeTS(*exp.ReturnType) + ">"
		}
		return "Promise<void>"
	default:
		if exp.ReturnType != nil {
			return formatIndexTypeTS(*exp.ReturnType)
		}
		return "void"
	}
}

func formatGeneratorReturnTS(typeName string, exp nodeinterop.IndexExport) string {
	if exp.YieldType != nil {
		return typeName + "<" + formatIndexTypeTS(*exp.YieldType) + ">"
	}
	return typeName
}

func formatIndexTypeTS(t nodeinterop.IndexType) string {
	if t.Binary {
		if t.Element != nil {
			return "Buffer"
		}
		return "Buffer"
	}
	switch strings.TrimSpace(t.Kind) {
	case "string":
		return "string"
	case "number":
		return "number"
	case "boolean":
		return "boolean"
	case "void":
		return "void"
	case "bytes":
		return "Uint8Array"
	case "array":
		if t.Element != nil {
			return formatIndexTypeTS(*t.Element) + "[]"
		}
		return "unknown[]"
	case "object":
		if len(t.Fields) == 0 {
			return "object"
		}
		names := make([]string, 0, len(t.Fields))
		for name := range t.Fields {
			if name == "$binary" {
				continue
			}
			names = append(names, name)
		}
		sort.Strings(names)
		parts := make([]string, 0, len(names))
		for _, name := range names {
			parts = append(parts, fmt.Sprintf("%s: %s", name, formatIndexTypeTS(t.Fields[name])))
		}
		return "{ " + strings.Join(parts, "; ") + " }"
	default:
		return "unknown"
	}
}

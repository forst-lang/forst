package main

import (
	"fmt"
	"sort"
	"strings"

	"forst/internal/genplugin"
	"forst/internal/semantic"
)

func emitORPCContract(procs []proc, z *zodEnc) string {
	var b strings.Builder
	b.WriteString(genplugin.TSGeneratedHeader)
	fmt.Fprintf(&b, "// Generator: forst-gen-orpc %s\n\n", version)
	b.WriteString("import { eventIterator, oc } from \"@orpc/contract\";\n")
	b.WriteString("import { z } from \"zod\";\n")
	b.WriteString("import { invokePositional, invokeStream } from \"./invoke.js\";\n")
	if n := z.importList(); n != "" {
		fmt.Fprintf(&b, "import { %s } from \"./zod.js\";\n", n)
	}
	b.WriteString("\n")

	byNS, order := groupProcs(procs)
	for _, ns := range order {
		fmt.Fprintf(&b, "export const %s = {\n", ns)
		for i, p := range byNS[ns] {
			if i > 0 {
				b.WriteString(",\n")
			}
			ident := genplugin.TSIdentifier(p.method.FieldName)
			fmt.Fprintf(&b, "  %s: oc\n", ident)
			fmt.Fprintf(&b, "    .input(%s)\n", inputSchema(p, z))
			fmt.Fprintf(&b, "    .output(%s)", outputSchema(p, z))
			if errs := errorSchema(p, z); errs != "" {
				fmt.Fprintf(&b, "\n    .errors(%s)", errs)
			}
			if p.http != nil {
				fmt.Fprintf(&b, "\n    .route({ method: %q, path: %q })", p.http.Method, p.http.Path)
			}
			b.WriteString(",\n")
			fmt.Fprintf(&b, "  %sImplement: %s", ident, implementExpr(p))
		}
		b.WriteString(",\n};\n\n")
	}
	if len(order) == 0 {
		b.WriteString("export const procedures = {} as const;\n")
	}
	return b.String()
}

func emitTRPC(procs []proc, z *zodEnc) string {
	var b strings.Builder
	b.WriteString(genplugin.TSGeneratedHeader)
	fmt.Fprintf(&b, "// Generator: forst-gen-orpc %s\n\n", version)
	b.WriteString("import { initTRPC } from \"@trpc/server\";\n")
	b.WriteString("import { invokePositional, invokeStream } from \"./invoke.js\";\n")
	if n := z.importList(); n != "" {
		fmt.Fprintf(&b, "import { %s } from \"./zod.js\";\n", n)
	}
	b.WriteString("\nconst t = initTRPC.create();\n")
	b.WriteString("const publicProcedure = t.procedure;\n\n")

	byNS, order := groupProcs(procs)
	for _, ns := range order {
		fmt.Fprintf(&b, "export const %s = t.router({\n", ns)
		for _, p := range byNS[ns] {
			ident := genplugin.TSIdentifier(p.method.FieldName)
			fmt.Fprintf(&b, "  %s: publicProcedure\n", ident)
			fmt.Fprintf(&b, "    .input(%s)\n", inputSchema(p, z))
			pkg, fn := p.method.Function.Package, p.method.Function.Name
			args := invokeArgs(p.method)
			switch p.kind {
			case "query":
				fmt.Fprintf(&b, "    .query(({ input }) => invokePositional(%q, %q, [%s])),\n", pkg, fn, args)
			case "subscription":
				fmt.Fprintf(&b, "    .subscription(() => invokeStream(%q, %q, [%s])),\n", pkg, fn, args)
			default:
				fmt.Fprintf(&b, "    .mutation(({ input }) => invokePositional(%q, %q, [%s])),\n", pkg, fn, args)
			}
		}
		b.WriteString("});\n\n")
	}
	if len(order) == 0 {
		b.WriteString("export const procedures = t.router({});\n")
	}
	return b.String()
}

func groupProcs(procs []proc) (map[string][]proc, []string) {
	byNS := map[string][]proc{}
	var order []string
	for _, p := range procs {
		ns := genplugin.TSIdentifier(genplugin.TypeShortName(p.surface.TypeID))
		if _, ok := byNS[ns]; !ok {
			order = append(order, ns)
		}
		byNS[ns] = append(byNS[ns], p)
	}
	return byNS, order
}

func (z *zodEnc) importList() string {
	seen := map[string]struct{}{}
	var names []string
	for id := range z.needIDs {
		if _, ok := z.types[id]; !ok {
			continue
		}
		n := z.nameOf(id)
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		names = append(names, n)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

func inputSchema(p proc, z *zodEnc) string {
	if p.inputID == "" || p.inputID == "void" || len(p.method.Function.Params) == 0 {
		return "z.void()"
	}
	return z.expr(p.inputID)
}

func outputSchema(p proc, z *zodEnc) string {
	inner := zodOutput(p.view.Success, z)
	if p.view.Stream {
		return fmt.Sprintf("eventIterator(%s)", inner)
	}
	if p.view.Void {
		return "z.void()"
	}
	return inner
}

func zodOutput(t semantic.Type, z *zodEnc) string {
	if t.Kind == "" || t.Kind == "unknown" {
		return "z.unknown()"
	}
	if t.ID != "" && t.ID != t.Kind {
		return z.expr(t.ID)
	}
	return z.inline(t)
}

func errorSchema(p proc, z *zodEnc) string {
	ids := genplugin.NominalErrors(z.types, p.method.Function)
	if len(ids) == 0 {
		return ""
	}
	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		parts = append(parts, fmt.Sprintf("%s: { data: %s }", genplugin.TypeShortName(id), z.expr(id)))
	}
	return "{ " + strings.Join(parts, ", ") + " }"
}

func implementExpr(p proc) string {
	pkg := p.method.Function.Package
	fn := p.method.Function.Name
	if pkg == "" || fn == "" {
		return "undefined"
	}
	args := invokeArgs(p.method)
	if p.kind == "subscription" {
		return fmt.Sprintf("() => invokeStream(%q, %q, [%s])", pkg, fn, args)
	}
	return fmt.Sprintf("(input: any) => invokePositional(%q, %q, [%s])", pkg, fn, args)
}

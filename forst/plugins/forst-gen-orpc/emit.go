package main

import (
	"fmt"
	"strings"

	"forst/internal/genplugin"
	"forst/internal/semantic"
)

type orpcOpt struct {
	Markers []string         `json:"markers"`
	Queries []string         `json:"queries"`
	Routes  map[string]route `json:"routes"`
	Style   string           `json:"style"` // "orpc" | "trpc"
	Client  string           `json:"clientImport"`
}

type route struct {
	Method string `json:"method"`
	Path   string `json:"path"`
}

type proc struct {
	surface genplugin.RouterSurface
	method  genplugin.RouterMethod
	key     string
	kind    string
	inputID string
	view    genplugin.CallView
	http    *route
}

func emitORPC(req *semantic.GenerateRequest) (semantic.GenerateResponse, error) {
	opt := orpcOpt{Markers: []string{"Router"}, Style: "orpc", Client: "@forst/client"}
	if err := genplugin.UnmarshalPluginOpt(req, &opt); err != nil {
		return semantic.GenerateResponse{}, err
	}
	if len(opt.Markers) == 0 {
		opt.Markers = []string{"Router"}
	}
	if opt.Style == "" {
		opt.Style = "orpc"
	}
	if opt.Client == "" {
		opt.Client = "@forst/client"
	}

	surfaces := genplugin.RouterSurfaces(req, opt.Markers)
	querySet := stringSet(opt.Queries)
	z := newZodEnc(req.Types)

	var diags []semantic.Diagnostic
	var procs []proc
	for _, surface := range surfaces {
		for _, m := range surface.Methods {
			key := surface.TypeID + "." + m.FieldName
			kind := procedureKind(key, m, querySet, req.Types)
			if m.Function.ID == "" {
				diags = append(diags, semantic.Diagnostic{
					Severity: "warning",
					Message:  fmt.Sprintf("contract member %q has no bound function; invoke adapter omitted", m.FieldName),
					TypeID:   surface.TypeID,
				})
			} else if !m.Function.Runnable {
				diags = append(diags, semantic.Diagnostic{
					Severity: "warning",
					Message:  "procedure function is not runnable (unsatisfied providers)",
					TypeID:   surface.TypeID,
					Span:     m.Function.Span,
				})
			}
			p := proc{
				surface: surface,
				method:  m,
				key:     key,
				kind:    kind,
				inputID: m.Function.Input,
				view:    genplugin.DerivedCall(req.Types, m.Function),
			}
			if r, ok := opt.Routes[key]; ok && r.Path != "" {
				if r.Method == "" {
					r.Method = httpMethod(kind)
				}
				cp := r
				p.http = &cp
			}
			z.need(m.Function.Input)
			if in, ok := req.Types[m.Function.Input]; ok {
				for _, f := range in.Fields {
					z.need(f.Type)
				}
			}
			if !p.view.Void {
				z.need(p.view.Success.ID)
			}
			for _, errID := range genplugin.NominalErrors(req.Types, m.Function) {
				z.need(errID)
			}
			procs = append(procs, p)
		}
	}

	files := []semantic.OutputFile{
		{Path: "zod.ts", Content: z.emit()},
		{Path: "invoke.ts", Content: emitInvokeTS(opt.Client)},
	}
	switch opt.Style {
	case "trpc":
		files = append(files, semantic.OutputFile{Path: "router.ts", Content: emitTRPC(procs, z)})
	default:
		files = append(files, semantic.OutputFile{Path: "contract.ts", Content: emitORPCContract(procs, z)})
	}
	files = append(files, semantic.OutputFile{Path: "meta.json", Content: genplugin.MustJSON(map[string]any{
		"generator":    "forst-gen-orpc",
		"version":      version,
		"style":        opt.Style,
		"surfaceCount": len(surfaces),
		"procCount":    len(procs),
	})})

	return semantic.GenerateResponse{
		ProtocolVersion: semantic.ProtocolVersion,
		Files:           files,
		Diagnostics:     diags,
	}, nil
}

func emitInvokeTS(clientImport string) string {
	var b strings.Builder
	b.WriteString(genplugin.TSGeneratedHeader)
	fmt.Fprintf(&b, "// Generator: forst-gen-orpc %s\n\n", version)
	fmt.Fprintf(&b, "import { createInvokeClient } from %q;\n\n", genplugin.ClientModuleSpecifier(clientImport))
	b.WriteString("const client = createInvokeClient();\n\n")
	b.WriteString("export async function invokePositional(packageName: string, functionName: string, args: unknown[]) {\n")
	b.WriteString("  return client.invokeFunction(packageName, functionName, args);\n")
	b.WriteString("}\n\n")
	b.WriteString("export function invokeStream(packageName: string, functionName: string, args: unknown[] = []) {\n")
	b.WriteString("  return client.invokeStream(packageName, functionName, args);\n")
	b.WriteString("}\n")
	return b.String()
}

func procedureKind(key string, m genplugin.RouterMethod, queries map[string]struct{}, types map[string]semantic.Type) string {
	if genplugin.DerivedCall(types, m.Function).Stream {
		return "subscription"
	}
	if genplugin.TypeOrChainHasConstraint(types, m.Sig, "Query") ||
		genplugin.TypeOrChainHasConstraint(types, m.FieldType, "Query") {
		return "query"
	}
	if _, ok := queries[key]; ok {
		return "query"
	}
	if genplugin.TypeOrChainHasConstraint(types, m.Sig, "Mutation") ||
		genplugin.TypeOrChainHasConstraint(types, m.FieldType, "Mutation") {
		return "mutation"
	}
	return "mutation"
}

func httpMethod(kind string) string {
	switch kind {
	case "query", "subscription":
		return "GET"
	default:
		return "POST"
	}
}

func stringSet(items []string) map[string]struct{} {
	out := make(map[string]struct{}, len(items))
	for _, s := range items {
		out[s] = struct{}{}
	}
	return out
}

func invokeArgs(m genplugin.RouterMethod) string {
	if len(m.Function.Params) == 0 {
		return ""
	}
	return genplugin.InputFieldAccess("input", m.Function.Params)
}

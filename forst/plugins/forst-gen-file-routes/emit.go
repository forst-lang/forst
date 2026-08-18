package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"forst/internal/genplugin"
	"forst/internal/semantic"
)

type fileRoutesOpt struct {
	Markers    []string `json:"markers"`
	RoutesRoot string   `json:"routesRoot"`
	ParamStyle string   `json:"paramStyle"`
	Client     string   `json:"clientImport"`
}

func emitFileRoutes(req *semantic.GenerateRequest) (semantic.GenerateResponse, error) {
	opt := fileRoutesOpt{
		Markers:    []string{"Router"},
		RoutesRoot: "app/api",
		ParamStyle: "$id",
		Client:     "@forst/client",
	}
	if req.Plugin != nil && len(req.Plugin.Opt) > 0 {
		_ = json.Unmarshal(req.Plugin.Opt, &opt)
	}
	if len(opt.Markers) == 0 {
		opt.Markers = []string{"Router"}
	}
	if opt.RoutesRoot == "" {
		opt.RoutesRoot = "app/api"
	}
	if opt.ParamStyle == "" {
		opt.ParamStyle = "$id"
	}
	if opt.Client == "" {
		opt.Client = "@forst/client"
	}
	clientMod := genplugin.ClientModuleSpecifier(opt.Client)

	surfaces, diags := genplugin.FileRouterSurfaces(req, genplugin.FileRouteOptions{
		RoutesRoot: opt.RoutesRoot,
		Markers:    opt.Markers,
		ParamStyle: opt.ParamStyle,
	})

	var emitted []genplugin.FileRouterSurface
	var perRoute [][]httpMember
	for _, s := range surfaces {
		methods := collectHTTPMethods(s, &diags)
		emitted = append(emitted, s)
		perRoute = append(perRoute, methods)
	}

	files := []semantic.OutputFile{
		{Path: "runtime.ts", Content: emitFileRuntime(clientMod)},
		{Path: "registry.ts", Content: emitRegistry(emitted, perRoute)},
		{Path: "routes.ts", Content: emitFileRouteConfig(emitted)},
	}
	for i, s := range emitted {
		files = append(files, semantic.OutputFile{
			Path:    s.HandlerFile,
			Content: emitFileHandler(s, perRoute[i]),
		})
	}
	files = append(files, semantic.OutputFile{
		Path: "meta.json",
		Content: genplugin.MustJSON(map[string]any{
			"generator":  "forst-gen-file-routes",
			"version":    version,
			"routesRoot": opt.RoutesRoot,
			"paramStyle": opt.ParamStyle,
			"routeCount": len(surfaces),
		}),
	})

	return semantic.GenerateResponse{
		ProtocolVersion: semantic.ProtocolVersion,
		Files:           files,
		Diagnostics:     diags,
	}, nil
}

func emitFileRuntime(clientMod string) string {
	var b strings.Builder
	b.WriteString(genplugin.TSGeneratedHeader)
	fmt.Fprintf(&b, "// Generator: forst-gen-file-routes %s\n\n", version)
	fmt.Fprintf(&b, "import { createInvokeClient } from %q;\n\n", clientMod)
	b.WriteString("export const client = createInvokeClient();\n\n")
	b.WriteString(genplugin.TSReadBodyHelper)
	b.WriteString("\n")
	b.WriteString(genplugin.TSMatchRouteHelper)
	return b.String()
}

func emitRegistry(surfaces []genplugin.FileRouterSurface, methods [][]httpMember) string {
	var b strings.Builder
	b.WriteString(genplugin.TSGeneratedHeader)
	fmt.Fprintf(&b, "// Generator: forst-gen-file-routes %s\n\n", version)
	b.WriteString("import { client, matchRoute, readBody } from \"./runtime.js\";\n\n")
	b.WriteString("export type RouteHandler = {\n")
	b.WriteString("  methods: Record<string, { package: string; function: string; params: string[] }>;\n")
	b.WriteString("  file: string;\n")
	b.WriteString("};\n\n")
	b.WriteString("export const routes: Record<string, RouteHandler> = {\n")
	first := true
	for i, s := range surfaces {
		ms := methods[i]
		if len(ms) == 0 {
			continue
		}
		if !first {
			b.WriteString(",\n")
		}
		first = false
		fmt.Fprintf(&b, "  %q: {\n", s.RoutePath)
		fmt.Fprintf(&b, "    file: %q,\n", s.File)
		b.WriteString("    methods: {\n")
		mfirst := true
		for _, m := range ms {
			if !mfirst {
				b.WriteString(",\n")
			}
			mfirst = false
			paramNames := make([]string, 0, len(m.Function.Params))
			for _, p := range m.Function.Params {
				paramNames = append(paramNames, p.Name)
			}
			fmt.Fprintf(&b, "      %q: { package: %q, function: %q, params: %s }",
				m.verb, m.Function.Package, m.Function.Name, genplugin.JSONArrayExpr(paramNames))
		}
		b.WriteString("\n    }\n  }")
	}
	b.WriteString("\n};\n\n")
	b.WriteString("export async function dispatch(method: string, req: Request): Promise<Response> {\n")
	b.WriteString("  const url = new URL(req.url);\n")
	b.WriteString("  const matched = matchRoute(url.pathname, routes);\n")
	b.WriteString("  if (!matched) return new Response(\"not found\", { status: 404 });\n")
	b.WriteString("  const target = matched.value.methods[method.toUpperCase()];\n")
	b.WriteString("  if (!target) return new Response(\"method not allowed\", { status: 405 });\n")
	b.WriteString("  const body = method.toUpperCase() === \"GET\" || method.toUpperCase() === \"HEAD\"\n")
	b.WriteString("    ? Object.fromEntries(url.searchParams.entries())\n")
	b.WriteString("    : await readBody(req);\n")
	b.WriteString("  const args = target.params.map((name) => matched.params[name] ?? body[name]);\n")
	b.WriteString("  const result = await client.invokeFunction(target.package, target.function, args);\n")
	b.WriteString("  return Response.json(result);\n")
	b.WriteString("}\n")
	return b.String()
}

func emitFileRouteConfig(surfaces []genplugin.FileRouterSurface) string {
	var b strings.Builder
	b.WriteString(genplugin.TSGeneratedHeader)
	fmt.Fprintf(&b, "// Generator: forst-gen-file-routes %s\n\n", version)
	b.WriteString("/** Spread into a user-owned Next catch-all or RR `routes.ts`. */\n")
	b.WriteString("export const forstApiRoutes = [\n")
	for i, s := range surfaces {
		if i > 0 {
			b.WriteString(",\n")
		}
		fmt.Fprintf(&b, "  { path: %q, module: %q, id: %q }",
			s.RoutePath, "./"+s.HandlerFile, genplugin.TSIdentifier(strings.ReplaceAll(s.RRPath, "/", "_")))
	}
	b.WriteString("\n] as const;\n")
	return b.String()
}

type httpMember struct {
	verb string
	genplugin.RouterMethod
}

func collectHTTPMethods(s genplugin.FileRouterSurface, diags *[]semantic.Diagnostic) []httpMember {
	var out []httpMember
	for _, m := range s.Methods {
		verb, ok := genplugin.HTTPVerb(m.FieldName)
		if !ok {
			lower := strings.ToLower(m.FieldName)
			if lower == "loader" {
				verb, ok = "GET", true
			} else if lower == "action" {
				verb, ok = "POST", true
			}
		}
		if !ok {
			*diags = append(*diags, semantic.Diagnostic{
				Severity: "warning",
				Message:  fmt.Sprintf("method field %q is not a known HTTP verb, loader, or action; skipped", m.FieldName),
				TypeID:   s.TypeID,
				Span:     m.Function.Span,
			})
			continue
		}
		if m.Function.ID == "" {
			*diags = append(*diags, semantic.Diagnostic{
				Severity: "error",
				Message:  fmt.Sprintf("HTTP member %q has no bound function", m.FieldName),
				TypeID:   s.TypeID,
			})
			continue
		}
		out = append(out, httpMember{verb: verb, RouterMethod: m})
	}
	return out
}

func emitFileHandler(s genplugin.FileRouterSurface, methods []httpMember) string {
	var b strings.Builder
	b.WriteString(genplugin.TSGeneratedHeader)
	fmt.Fprintf(&b, "// Generator: forst-gen-file-routes %s\n", version)
	fmt.Fprintf(&b, "// Source: %s\n\n", s.File)
	b.WriteString("import { client, readBody } from \"../runtime.js\";\n\n")
	for _, m := range methods {
		args := genplugin.ArgAccessExpr(m.Function.Params, s.PathParams, "params", "body")
		fmt.Fprintf(&b, "export async function %s(params: Record<string, string>, request?: Request) {\n", m.verb)
		b.WriteString("  const body = request ? await readBody(request) : {};\n")
		fmt.Fprintf(&b, "  return client.invokeFunction(%q, %q, [%s]);\n", m.Function.Package, m.Function.Name, args)
		b.WriteString("}\n\n")
	}
	return b.String()
}

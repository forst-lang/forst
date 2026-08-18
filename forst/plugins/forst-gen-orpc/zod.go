package main

import (
	"fmt"
	"sort"
	"strings"

	"forst/internal/genplugin"
	"forst/internal/semantic"
)

type zodEnc struct {
	types   map[string]semantic.Type
	needIDs map[string]struct{}
	emitted map[string]string // type id → TS const name
	byName  map[string]string // TS const name → type id
	used    map[string]int
	order   []string
	cycle   map[string]bool
}

func newZodEnc(types map[string]semantic.Type) *zodEnc {
	return &zodEnc{
		types:   types,
		needIDs: map[string]struct{}{},
		emitted: map[string]string{},
		byName:  map[string]string{},
		used:    map[string]int{},
		cycle:   map[string]bool{},
	}
}

func (z *zodEnc) need(id string) {
	if id == "" || id == "void" {
		return
	}
	if _, ok := primitiveKinds[id]; ok && !genplugin.IsPublishedTypeID(id) {
		return
	}
	if !genplugin.IsPublishedTypeID(id) && strings.Contains(id, ".") {
		// still need synthesized input shapes
		if strings.HasSuffix(id, ".input") {
			z.needIDs[id] = struct{}{}
			return
		}
	}
	if genplugin.IsPublishedTypeID(id) || strings.HasSuffix(id, ".input") {
		z.needIDs[id] = struct{}{}
	}
}

var primitiveKinds = map[string]struct{}{
	"string": {}, "int": {}, "float": {}, "bool": {}, "bytes": {}, "void": {},
	"error": {}, "array": {}, "map": {}, "shape": {}, "result": {},
	"channel": {}, "func": {}, "unknown": {}, "nominalError": {},
}

func (z *zodEnc) nameOf(id string) string {
	if n, ok := z.emitted[id]; ok {
		return n
	}
	var base string
	if strings.HasSuffix(id, ".input") {
		base = genplugin.TSIdentifier(strings.ReplaceAll(strings.TrimSuffix(id, ".input"), ".", "_")) + "Input"
	} else {
		base = genplugin.TSIdentifier(genplugin.UniqueTypeName(id, z.used))
	}
	name := base + "Schema"
	if prev, ok := z.byName[name]; ok && prev != id {
		base = genplugin.TSIdentifier(strings.ReplaceAll(id, ".", "_"))
		name = base + "Schema"
	}
	z.emitted[id] = name
	z.byName[name] = id
	return name
}

func (z *zodEnc) expr(id string) string {
	if id == "" || id == "void" {
		return "z.void()"
	}
	if _, ok := z.needIDs[id]; ok || genplugin.IsPublishedTypeID(id) {
		if _, known := z.types[id]; known {
			return z.nameOf(id)
		}
	}
	return z.inline(genplugin.Lookup(z.types, id))
}

func (z *zodEnc) emit() string {
	ids := z.sortedNeedIDs()
	var b strings.Builder
	b.WriteString(genplugin.TSGeneratedHeader)
	fmt.Fprintf(&b, "// Generator: forst-gen-orpc %s\n\n", version)
	b.WriteString("import { z } from \"zod\";\n\n")
	for _, id := range ids {
		t, ok := z.types[id]
		if !ok {
			continue
		}
		fmt.Fprintf(&b, "export const %s = %s;\n", z.nameOf(id), z.inline(t))
	}
	if len(ids) == 0 {
		b.WriteString("export {};\n")
	}
	return b.String()
}

func (z *zodEnc) sortedNeedIDs() []string {
	ids := make([]string, 0, len(z.needIDs))
	for id := range z.needIDs {
		ids = append(ids, id)
	}
	deps := map[string][]string{}
	for _, id := range ids {
		t, ok := z.types[id]
		if !ok {
			continue
		}
		deps[id] = z.typeRefs(t)
	}
	var order []string
	seen := map[string]struct{}{}
	var visit func(string)
	visit = func(id string) {
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		for _, dep := range deps[id] {
			visit(dep)
		}
		order = append(order, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		visit(id)
	}
	return order
}

func (z *zodEnc) typeRefs(t semantic.Type) []string {
	var refs []string
	add := func(id string) {
		if id == "" || id == "void" {
			return
		}
		if _, ok := z.needIDs[id]; ok {
			refs = append(refs, id)
		}
	}
	add(t.Element)
	add(t.Value)
	add(t.Inner)
	add(t.Underlying)
	add(t.Success)
	add(t.Failure)
	add(t.Payload)
	for _, id := range t.Members {
		add(id)
	}
	for _, f := range t.Fields {
		add(f.Type)
	}
	return refs
}

func (z *zodEnc) inline(t semantic.Type) string {
	if t.ID != "" {
		if z.cycle[t.ID] {
			return "z.lazy(() => z.any())"
		}
		z.cycle[t.ID] = true
		defer func() { z.cycle[t.ID] = false }()
	}
	resolved := genplugin.FollowAlias(z.types, t)
	chain := genplugin.ConstraintChain(z.types, t)
	body := z.kindExpr(resolved)
	return applyZodConstraints(body, resolved.Kind, chain)
}

func (z *zodEnc) kindExpr(t semantic.Type) string {
	switch t.Kind {
	case "string":
		return "z.string()"
	case "int":
		return "z.number().int()"
	case "float":
		return "z.number()"
	case "bool":
		return "z.boolean()"
	case "bytes":
		return "z.string()"
	case "void":
		return "z.void()"
	case "error":
		return "z.object({ message: z.string() }).passthrough()"
	case "array":
		return fmt.Sprintf("z.array(%s)", z.expr(t.Element))
	case "map":
		return fmt.Sprintf("z.record(z.string(), %s)", z.expr(t.Value))
	case "pointer":
		return fmt.Sprintf("%s.nullable()", z.expr(t.Inner))
	case "union":
		return "z.union([" + z.memberList(t.Members) + "])"
	case "intersection":
		if len(t.Members) == 0 {
			return "z.any()"
		}
		expr := z.expr(t.Members[0])
		for _, id := range t.Members[1:] {
			expr = fmt.Sprintf("z.intersection(%s, %s)", expr, z.expr(id))
		}
		return expr
	case "tuple":
		return "z.tuple([" + z.memberList(t.Members) + "])"
	case "result":
		return fmt.Sprintf(
			"z.union([z.object({ ok: z.literal(true), value: %s }), z.object({ ok: z.literal(false), error: %s })])",
			z.expr(t.Success), z.expr(t.Failure),
		)
	case "shape", "nominalError":
		return z.objectExpr(t)
	case "channel":
		return z.expr(t.Element)
	case "alias":
		return z.expr(t.Underlying)
	default:
		return "z.unknown()"
	}
}

func (z *zodEnc) memberList(ids []string) string {
	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		parts = append(parts, z.expr(id))
	}
	return strings.Join(parts, ", ")
}

func (z *zodEnc) objectExpr(t semantic.Type) string {
	fields := t.Fields
	if t.Kind == "nominalError" && t.Payload != "" {
		fields = genplugin.Lookup(z.types, t.Payload).Fields
	}
	var b strings.Builder
	b.WriteString("z.object({\n")
	for _, f := range fields {
		if f.Method {
			continue
		}
		inner := z.expr(f.Type)
		if f.Optional {
			inner += ".optional()"
		}
		fmt.Fprintf(&b, "  %s: %s,\n", genplugin.TSIdentifier(f.Name), inner)
	}
	b.WriteString("})")
	return b.String()
}

func applyZodConstraints(expr, kind string, chain []semantic.Constraint) string {
	for _, c := range chain {
		if c.Origin != "builtin" {
			continue
		}
		switch c.Name {
		case "Min":
			if n, ok := numericArg(c.Args); ok && zodBoundApplies(kind, c.Applies) {
				expr += fmt.Sprintf(".min(%v)", intOrFloat(n, kind, c.Applies))
			}
		case "Max":
			if n, ok := numericArg(c.Args); ok && zodBoundApplies(kind, c.Applies) {
				expr += fmt.Sprintf(".max(%v)", intOrFloat(n, kind, c.Applies))
			}
		case "LessThan":
			if (kind == "int" || kind == "float") {
				if n, ok := numericArg(c.Args); ok {
					expr += fmt.Sprintf(".lt(%v)", n)
				}
			}
		case "GreaterThan":
			if kind == "int" || kind == "float" {
				if n, ok := numericArg(c.Args); ok {
					expr += fmt.Sprintf(".gt(%v)", n)
				}
			}
		case "HasPrefix":
			if prefix, ok := stringArg(c.Args); ok && kind == "string" {
				expr += fmt.Sprintf(".regex(/^%s/)", quoteMeta(prefix))
			}
		case "Contains":
			if sub, ok := stringArg(c.Args); ok && kind == "string" {
				expr += fmt.Sprintf(".regex(/.*%s.*/)", quoteMeta(sub))
			}
		case "NotEmpty":
			if kind == "string" || kind == "array" {
				expr += ".min(1)"
			}
		case "True":
			expr = "z.literal(true)"
		case "False":
			expr = "z.literal(false)"
		case "Nil":
			expr = expr + ".nullable()"
		case "Present":
			expr = expr + ".refine((v) => v != null)"
		case "Value":
			if len(c.Args) >= 1 {
				expr = fmt.Sprintf("z.literal(%v)", zodLiteral(c.Args[0]))
			}
		}
	}
	return expr
}

func zodBoundApplies(kind, applies string) bool {
	target := applies
	if target == "" {
		switch kind {
		case "string", "array", "int", "float":
			return true
		default:
			return false
		}
	}
	switch target {
	case "length":
		return kind == "string"
	case "items":
		return kind == "array"
	case "value":
		return kind == "int" || kind == "float"
	default:
		return false
	}
}

func zodLiteral(v any) string {
	switch x := v.(type) {
	case string:
		return fmt.Sprintf("%q", x)
	case bool:
		return fmt.Sprintf("%t", x)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func intOrFloat(n float64, kind, applies string) any {
	if kind == "int" || applies == "length" || applies == "items" {
		return int(n)
	}
	return n
}

func numericArg(args []any) (float64, bool) {
	if len(args) == 0 {
		return 0, false
	}
	switch v := args[0].(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	default:
		return 0, false
	}
}

func stringArg(args []any) (string, bool) {
	if len(args) == 0 {
		return "", false
	}
	s, ok := args[0].(string)
	return s, ok
}

func quoteMeta(s string) string {
	b := make([]byte, 0, len(s)*2)
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '.', '+', '*', '?', '(', ')', '|', '[', ']', '{', '}', '\\', '^', '$', '/':
			b = append(b, '\\', c)
		default:
			b = append(b, c)
		}
	}
	return string(b)
}

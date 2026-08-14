package transformerts

import (
	"strings"
	"testing"
)

func TestDirectInvokeExportLines_emitsBoundNamespace(t *testing.T) {
	lines := DirectInvokeExportLines("main", []FunctionSignature{
		{Name: "ListTodos", ReturnType: "ListTodosResponse"},
		{
			Name:       "AddTodo",
			ReturnType: "AddTodoResponse",
			Parameters: []Parameter{{Name: "input", Type: "AddTodoRequest"}},
		},
	})
	text := strings.Join(lines, "\n")
	for _, frag := range []string{
		"export const $main = {",
		"ListTodos: async ()",
		"AddTodo: async (input: AddTodoRequest)",
		"getDefaultInvokeClient",
		"'main', 'ListTodos', []",
		"'main', 'AddTodo', [input]",
	} {
		if !strings.Contains(text, frag) {
			t.Fatalf("missing %q in:\n%s", frag, text)
		}
	}
	for _, banned := range []string{"@forst/client", "@forst/sidecar", "export async function ListTodos"} {
		if strings.Contains(text, banned) {
			t.Fatalf("direct invoke exports must not contain %q:\n%s", banned, text)
		}
	}
}

func TestDirectInvokeClientImportLine_usesInlinedTransport(t *testing.T) {
	got := DirectInvokeClientImportLine()
	want := "import { getDefaultInvokeClient } from './transport/runtime.js';"
	if got != want {
		t.Fatalf("DirectInvokeClientImportLine() = %q, want %q", got, want)
	}
	if strings.Contains(got, "@forst/client") {
		t.Fatalf("import must not use @forst/client: %q", got)
	}
}

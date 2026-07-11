package transformerts

import "strings"
import "testing"

func TestDirectInvokeExportLines_emitsLazyNamedExports(t *testing.T) {
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
		"export async function ListTodos",
		"export async function AddTodo",
		"getDefaultInvokeClient",
		"'main', 'ListTodos', []",
		"'main', 'AddTodo', [input]",
	} {
		if !strings.Contains(text, frag) {
			t.Fatalf("missing %q in:\n%s", frag, text)
		}
	}
}

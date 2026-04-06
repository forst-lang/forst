package printer

import (
	"strings"
	"testing"

	"forst/internal/lexer"
	"forst/internal/parser"

	"github.com/sirupsen/logrus"
)

// TestFormatSource_roundTrip_coversStmtVariants exercises the pretty-printer paths that
// FormatSource alone previously skipped (control flow, imports, type defs, type guards).
func TestFormatSource_roundTrip_coversStmtVariants(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		src  string
	}{
		{
			name: "import_group",
			src: `package main

import (
	"fmt"
)

func main() {
	fmt.Println("ok")
}
`,
		},
		{
			name: "type_alias_shape",
			src: `package main

type User = {
	id: Int,
}

func main() {
}
`,
		},
		{
			name: "type_guard",
			src: `package main

type Password = String

is (password Password) Strong {
	ensure password is Min(12)
}

func main() {
}
`,
		},
		{
			name: "if_else_chain",
			src: `package main

func main() {
	x := 1
	if x > 0 {
		return
	} else if x < 0 {
		return
	} else {
		return
	}
}
`,
		},
		{
			name: "if_with_init",
			src: `package main

func main() {
	if x := 1; x > 0 {
		return
	}
}
`,
		},
		{
			name: "for_variants",
			src: `package main

func main() {
	for {
		break
	}
	for true {
		continue
	}
	xs := [1, 2]
	for i, v := range xs {
		println(i, v)
	}
}
`,
		},
		{
			name: "var_decl_and_assign",
			src: `package main

func main() {
	var x: Int = 1
	x = 2
	y := 3
}
`,
		},
		{
			name: "defer_go",
			src: `package main

func f() {}

func main() {
	defer f()
	go f()
}
`,
		},
		{
			name: "refs_and_slice",
			src: `package main

func main() {
	xs := [1, 2]
	p := &xs
	println(*p)
}
`,
		},
		{
			name: "func_param_slice_type",
			src: `package main

func getCell(cells []String, idx Int): String {
	return cells[idx]
}
`,
		},
	}
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out, err := FormatSource(tc.src, tc.name+".ft", log)
			if err != nil {
				t.Fatalf("FormatSource: %v\n--- src ---\n%s", err, tc.src)
			}
			if !strings.HasSuffix(out, "\n") {
				t.Fatal("expected trailing newline")
			}
			l := lexer.New([]byte(out), tc.name+".ft", log)
			tokens := l.Lex()
			p := parser.New(tokens, tc.name+".ft", log)
			if _, err := p.ParseFile(); err != nil {
				t.Fatalf("re-parse: %v\n--- out ---\n%s", err, out)
			}
		})
	}
}

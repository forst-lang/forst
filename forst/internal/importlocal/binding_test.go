package importlocal

import (
	"testing"

	"forst/internal/ast"
)

func TestBindingFromAST(t *testing.T) {
	tests := []struct {
		name     string
		imp      ast.ImportNode
		opts     BindingOpts
		want     Binding
	}{
		{
			name: "go explicit alias",
			imp:  ast.ImportNode{Path: `"fmt"`, Alias: &ast.Ident{ID: "f"}},
			opts: BindingOpts{Kind: KindGo, ModuleID: "fmt"},
			want: Binding{Local: "f", ImportPath: `"fmt"`, ModuleID: "fmt", GoPath: "fmt", Skip: false},
		},
		{
			name: "go dot import",
			imp:  ast.ImportNode{Path: `"fmt"`, Alias: &ast.Ident{ID: "."}},
			opts: BindingOpts{Kind: KindGo},
			want: Binding{ImportPath: `"fmt"`, GoPath: "fmt", Skip: true},
		},
		{
			name: "go blank import",
			imp:  ast.ImportNode{Path: `"fmt"`, Alias: &ast.Ident{ID: "_"}, SideEffectOnly: true},
			opts: BindingOpts{Kind: KindGo},
			want: Binding{ImportPath: `"fmt"`, GoPath: "fmt", Skip: true},
		},
		{
			name: "go pkg name override",
			imp:  ast.ImportNode{Path: `"example.com/foo/type"`},
			opts: BindingOpts{Kind: KindGo, ModuleID: "example.com/foo/type", GoPkgName: "mylib"},
			want: Binding{Local: "mylib", ImportPath: `"example.com/foo/type"`, ModuleID: "example.com/foo/type", GoPath: "example.com/foo/type", Skip: false},
		},
		{
			name: "go path tail fallback",
			imp:  ast.ImportNode{Path: `"example.com/foo/type"`},
			opts: BindingOpts{Kind: KindGo, ModuleID: "example.com/foo/type"},
			want: Binding{Local: "type", ImportPath: `"example.com/foo/type"`, ModuleID: "example.com/foo/type", GoPath: "example.com/foo/type", Skip: false},
		},
		{
			name: "node implicit from module id",
			imp:  ast.ImportNode{Path: "./legacy/type.ts", NodeOptIn: true},
			opts: BindingOpts{Kind: KindNode, ModuleID: "legacy/type.ts"},
			want: Binding{Local: "type", ImportPath: "./legacy/type.ts", ModuleID: "legacy/type.ts", Skip: false},
		},
		{
			name: "node explicit alias",
			imp:  ast.ImportNode{Path: "./legacy/type.ts", Alias: &ast.Ident{ID: "typePkg"}, NodeOptIn: true},
			opts: BindingOpts{Kind: KindNode, ModuleID: "legacy/type.ts"},
			want: Binding{Local: "typePkg", ImportPath: "./legacy/type.ts", ModuleID: "legacy/type.ts", Skip: false},
		},
		{
			name: "node bare specifier",
			imp:  ast.ImportNode{Path: "map", NodeOptIn: true},
			opts: BindingOpts{Kind: KindNode, ModuleID: "map"},
			want: Binding{Local: "map", ImportPath: "map", ModuleID: "map", Skip: false},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BindingFromAST(tt.imp, tt.opts)
			if got.Local != tt.want.Local || got.Skip != tt.want.Skip || got.GoPath != tt.want.GoPath {
				t.Fatalf("BindingFromAST() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

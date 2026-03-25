package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	transformerts "forst/internal/transformer/ts"

	"github.com/sirupsen/logrus"
)

const minimalValidForst = `package main

type EchoRequest = {
	message: String
}

func Echo(input EchoRequest) {
	return {
		echo: input.message,
		timestamp: 1234567890,
	}
}
`

func TestGenerateCommand_requiresTarget(t *testing.T) {
	err := generateCommand(nil)
	if err == nil {
		t.Fatal("expected error when args empty")
	}
	if !strings.Contains(err.Error(), "requires a target") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGenerateCommand_statError(t *testing.T) {
	err := generateCommand([]string{filepath.Join(t.TempDir(), "nonexistent.ft")})
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if !strings.Contains(err.Error(), "failed to stat") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGenerateCommand_rejectsNonFtFile(t *testing.T) {
	tmp := t.TempDir()
	plain := filepath.Join(tmp, "readme.txt")
	if err := os.WriteFile(plain, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	err := generateCommand([]string{plain})
	if err == nil {
		t.Fatal("expected error for non-.ft file")
	}
	if !strings.Contains(err.Error(), ".ft extension") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGenerateCommand_emptyDirectoryHasNoFtFiles(t *testing.T) {
	dir := t.TempDir()
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("expected nil when no .ft files, got %v", err)
	}
}

func TestGenerateCommand_singleFtFileWritesGeneratedAndClient(t *testing.T) {
	dir := t.TempDir()
	ftPath := filepath.Join(dir, "sample.ft")
	if err := os.WriteFile(ftPath, []byte(minimalValidForst), 0644); err != nil {
		t.Fatal(err)
	}

	if err := generateCommand([]string{ftPath}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}

	for _, rel := range []string{
		"generated/types.d.ts",
		"generated/sample.d.ts",
		"generated/sample.client.ts",
		"client/index.ts",
		"client/package.json",
		"client/types.d.ts",
	} {
		path := filepath.Join(dir, rel)
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected file %s: %v", rel, err)
		}
	}

	types, err := os.ReadFile(filepath.Join(dir, "generated", "types.d.ts"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(types), "Echo") {
		t.Fatalf("types.d.ts should mention Echo; got:\n%s", types)
	}
}

func TestFindForstFiles_nestedAndFlat(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "nested")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{
		filepath.Join(root, "a.ft"),
		filepath.Join(sub, "b.ft"),
	} {
		if err := os.WriteFile(p, []byte("package main\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	files, err := findForstFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("want 2 .ft files, got %d: %v", len(files), files)
	}
}

func TestProcessForstFile_invalidForstReturnsError(t *testing.T) {
	dir := t.TempDir()
	ftPath := filepath.Join(dir, "bad.ft")
	if err := os.WriteFile(ftPath, []byte("not valid forst {{{"), 0644); err != nil {
		t.Fatal(err)
	}
	log := logrus.New()
	log.SetOutput(io.Discard)
	err := processForstFile(ftPath, dir, log)
	if err == nil {
		t.Fatal("expected error for unparseable file")
	}
}

func TestGenerateClientImplementation_containsInvokeAndTypes(t *testing.T) {
	out := &transformerts.TypeScriptOutput{
		Types: []string{"export interface Foo { x: number; }"},
		Functions: []transformerts.FunctionSignature{
			{
				Name:       "Bar",
				ReturnType: "string",
				Parameters: []transformerts.Parameter{
					{Name: "a", Type: "number"},
					{Name: "b", Type: "string"},
				},
			},
		},
	}
	s := generateClientImplementation("mypkg", out)
	for _, frag := range []string{
		"export function mypkg",
		"ForstSidecarClient",
		"export interface Foo",
		"Bar: async (a: number, b: string)",
		"client.invoke<string>('Bar'",
	} {
		if !strings.Contains(s, frag) {
			t.Fatalf("missing %q in:\n%s", frag, s)
		}
	}
}

func TestGenerateClientImplementation_emptyFunctionsStillExportsWrapper(t *testing.T) {
	s := generateClientImplementation("empty", &transformerts.TypeScriptOutput{})
	if !strings.Contains(s, "export function empty(client: ForstSidecarClient)") {
		t.Fatalf("unexpected output:\n%s", s)
	}
}

func TestExtractFunctionName(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"hello(", "hello"},
		{"export function hello(", "hello"},
		{"no paren", ""},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := extractFunctionName(tt.in); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGenerateClientIndex_importsPackages(t *testing.T) {
	idx := generateClientIndex([]string{
		filepath.Join("proj", "alpha.ft"),
		filepath.Join("proj", "beta.ft"),
	})
	for _, frag := range []string{
		"import { alpha } from '../generated/alpha.client'",
		"import { beta } from '../generated/beta.client'",
		"public alpha:",
		"public beta:",
	} {
		if !strings.Contains(idx, frag) {
			t.Fatalf("missing %q in index:\n%s", frag, idx)
		}
	}
}

func TestGenerateClientPackageJson_isValidJSONShape(t *testing.T) {
	j := generateClientPackageJson()
	if !strings.Contains(j, `"name": "@forst/client"`) || !strings.Contains(j, `"@forst/sidecar"`) {
		t.Fatalf("unexpected package.json:\n%s", j)
	}
}

func TestCopyFile_roundTrip(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")
	payload := []byte("hello generate")
	if err := os.WriteFile(src, payload, 0644); err != nil {
		t.Fatal(err)
	}
	if err := copyFile(src, dst); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(payload) {
		t.Fatalf("got %q, want %q", got, payload)
	}
}

func TestCopyFile_missingSource(t *testing.T) {
	err := copyFile(filepath.Join(t.TempDir(), "nope"), filepath.Join(t.TempDir(), "out"))
	if err == nil {
		t.Fatal("expected error")
	}
}

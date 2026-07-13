package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forst/internal/compiler"
)

type embeddedInvokeGoldenCase struct {
	name               string
	entryRel           string
	goldenRel          string
	exportStructFields bool
	mainMarkers        []string
	invokeMarkers      []string
}

func embeddedInvokeGoldenCases() []embeddedInvokeGoldenCase {
	return []embeddedInvokeGoldenCase{
		{
			name:               "embedded-invoke",
			entryRel:           "rfc/embedded-invoke/main.ft",
			goldenRel:          "rfc/embedded-invoke/main.go",
			exportStructFields: true,
			mainMarkers: []string{
				"package main",
				"func Echo(",
				"ForstInvokeWaitForShutdown",
				"func main()",
			},
			invokeMarkers: []string{
				"invokeembed.MustStartEmbedded",
				"forst_invoke_main_Echo",
				"reg.RegisterMeta",
				"ForstInvokeWaitForShutdown",
			},
		},
	}
}

type embeddedInvokeCompileOutput struct {
	Main   string
	Invoke string
}

func compileEmbeddedInvokePackageForGolden(t *testing.T, entryPath, root string, exportStructFields bool) embeddedInvokeCompileOutput {
	t.Helper()
	absEntry, err := filepath.Abs(entryPath)
	if err != nil {
		t.Fatal(err)
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	c := compiler.New(compiler.Args{
		Command:            "build",
		FilePath:           absEntry,
		PackageRoot:        absRoot,
		ExportStructFields: exportStructFields,
		LogLevel:           "error",
	}, exampleTestLogger())
	mainCode, _, invokeCode, _, _, err := c.CompileWithNodeRuntime()
	if err != nil {
		t.Fatalf("CompileWithNodeRuntime(%s): %v", absEntry, err)
	}
	return embeddedInvokeCompileOutput{Main: mainCode, Invoke: invokeCode}
}

func invokeServerGoldenPath(mainGoldenPath string) string {
	ext := filepath.Ext(mainGoldenPath)
	base := strings.TrimSuffix(mainGoldenPath, ext)
	if ext == "" {
		return base + "_forst_invoke_server.gen.go"
	}
	return base + "_forst_invoke_server.gen" + ext
}

func verifyEmbeddedInvokePackageCompileGolden(t *testing.T, expected, actual, goldenPath string, markers []string) {
	t.Helper()
	for _, marker := range markers {
		if !strings.Contains(actual, marker) {
			t.Errorf("output missing %q (golden %s)", marker, goldenPath)
		}
	}
	if len(expected) > 0 && len(actual) < len(expected)/2 {
		t.Errorf("output much shorter than golden (%d vs %d bytes)", len(actual), len(expected))
	}
}

func writeEmbeddedInvokePackageGolden(t *testing.T, tc embeddedInvokeGoldenCase) {
	t.Helper()
	inDir := examplesInDir(t)
	outDir := examplesOutDir(t)
	entry := filepath.Join(inDir, tc.entryRel)
	root := filepath.Dir(entry)
	goldenPath := filepath.Join(outDir, tc.goldenRel)
	invokeGoldenPath := invokeServerGoldenPath(goldenPath)

	out := compileEmbeddedInvokePackageForGolden(t, entry, root, tc.exportStructFields)
	if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(goldenPath, []byte(out.Main), 0o644); err != nil {
		t.Fatal(err)
	}
	if out.Invoke != "" {
		if err := os.WriteFile(invokeGoldenPath, []byte(out.Invoke), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote %s", invokeGoldenPath)
	}
	t.Logf("wrote %s", goldenPath)
}

func TestExampleEmbeddedInvokeCompileGolden(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping embedded-invoke goldens in -short mode")
	}
	for _, tc := range embeddedInvokeGoldenCases() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			inDir := examplesInDir(t)
			outDir := examplesOutDir(t)
			entry := filepath.Join(inDir, tc.entryRel)
			root := filepath.Dir(entry)
			goldenPath := filepath.Join(outDir, tc.goldenRel)
			invokeGoldenPath := invokeServerGoldenPath(goldenPath)

			actual := compileEmbeddedInvokePackageForGolden(t, entry, root, tc.exportStructFields)

			if os.Getenv("UPDATE_EMBEDDED_INVOKE_GOLDEN") == "1" || os.Getenv("UPDATE_EXAMPLES_GOLDENS") == "1" {
				writeEmbeddedInvokePackageGolden(t, tc)
				return
			}

			expectedMain, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("read golden %s: %v (set UPDATE_EMBEDDED_INVOKE_GOLDEN=1 to create)", goldenPath, err)
			}
			verifyEmbeddedInvokePackageCompileGolden(t, string(expectedMain), actual.Main, goldenPath, tc.mainMarkers)
			verifyCompanionPackageGoBuild(t, "fresh compile/"+tc.name, actual.Main, "", actual.Invoke)
			verifyCompanionGoldenFilesGoBuild(
				t,
				"committed goldens/"+tc.name,
				goldenPath,
				"",
				invokeGoldenPath,
			)

			if len(tc.invokeMarkers) > 0 {
				if actual.Invoke == "" {
					t.Fatalf("expected invoke server output for %s", tc.name)
				}
				expectedInvoke, err := os.ReadFile(invokeGoldenPath)
				if err != nil {
					t.Fatalf("read invoke golden %s: %v (set UPDATE_EMBEDDED_INVOKE_GOLDEN=1 to create)", invokeGoldenPath, err)
				}
				verifyEmbeddedInvokePackageCompileGolden(t, string(expectedInvoke), actual.Invoke, invokeGoldenPath, tc.invokeMarkers)
			}
		})
	}
}

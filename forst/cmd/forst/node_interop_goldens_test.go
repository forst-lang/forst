package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forst/internal/compiler"
)

type nodeInteropGoldenCase struct {
	name               string
	entryRel           string // under examples/in
	goldenRel          string // under examples/out
	exportStructFields bool
	mainMarkers        []string
	runtimeMarkers     []string
	invokeMarkers      []string
}

func nodeInteropGoldenCases() []nodeInteropGoldenCase {
	return []nodeInteropGoldenCase{
		{
			name:      "node-interop",
			entryRel:  "rfc/node-interop/main.ft",
			goldenRel: "rfc/node-interop/main.go",
			mainMarkers: []string{
				"package main",
				"forst_node_callsync_",
				"func main()",
			},
			runtimeMarkers: []string{
				"package main",
				"forstNodeManifestJSON",
				"forst/nodert",
				"nodert.CallSync",
			},
		},
		{
			name:               "sync",
			entryRel:           "rfc/node-interop/sync/main.ft",
			goldenRel:          "rfc/node-interop/sync/main.go",
			exportStructFields: true,
			mainMarkers: []string{
				"package main",
				"forst_node_callsync_",
				"result.Amount",
				"func main()",
			},
			runtimeMarkers: []string{
				"forstNodeManifestJSON",
				"nodert.CallSync",
			},
		},
		{
			name:               "promises",
			entryRel:           "rfc/node-interop/promises/main.ft",
			goldenRel:          "rfc/node-interop/promises/main.go",
			exportStructFields: true,
			mainMarkers: []string{
				"package main",
				"forst_node_callasync_",
				"concurrentEcho",
				"func main()",
			},
			runtimeMarkers: []string{
				"forstNodeManifestJSON",
				"nodert.CallAsync",
			},
		},
		{
			name:               "generators",
			entryRel:           "rfc/node-interop/generators/main.ft",
			goldenRel:          "rfc/node-interop/generators/main.go",
			exportStructFields: true,
			mainMarkers: []string{
				"package main",
				"forst_node_open_seq_",
				"forstNodeGenStepDone",
				"func main()",
			},
			runtimeMarkers: []string{
				"forstNodeManifestJSON",
				"nodert.OpenSeq",
			},
		},
		{
			name:               "async",
			entryRel:           "rfc/node-interop/async/main.ft",
			goldenRel:          "rfc/node-interop/async/main.go",
			exportStructFields: true,
			mainMarkers: []string{
				"package main",
				"forst_node_callasync_",
				"forst_node_open_seq_",
				"func main()",
			},
			runtimeMarkers: []string{
				"forstNodeManifestJSON",
				"nodert.CallAsync",
				"nodert.OpenSeq",
			},
		},
		{
			name:               "host",
			entryRel:           "rfc/node-interop/host/main.ft",
			goldenRel:          "rfc/node-interop/host/main.go",
			exportStructFields: true,
			mainMarkers: []string{
				"package main",
				"forst_node_callsync_",
				"func main()",
			},
			runtimeMarkers: []string{
				"forstNodeManifestJSON",
				"nodert.CallSync",
			},
		},
		{
			name:               "remix-serve",
			entryRel:           "rfc/node-interop/remix-serve/main.ft",
			goldenRel:          "rfc/node-interop/remix-serve/main.go",
			exportStructFields: true,
			mainMarkers: []string{
				"package main",
				"func ListTodos(",
				"func AddTodo(",
				"ForstInvokeWaitForShutdown",
				"bumpEditCount",
				"func main()",
			},
			runtimeMarkers: []string{
				"forstNodeManifestJSON",
				"nodert.CallSync",
				"nodert.CallAsync",
				"nodert.OpenSeq",
				"legacy/todos.ts",
			},
			invokeMarkers: []string{
				"invokeserver.MustStartEmbedded",
				"forst_invoke_main_ListTodos",
				"ForstInvokeWaitForShutdown",
			},
		},
		{
			name:               "modules",
			entryRel:           "rfc/node-interop/modules/main.ft",
			goldenRel:          "rfc/node-interop/modules/main.go",
			exportStructFields: true,
			mainMarkers: []string{
				"package main",
				"forst_node_callsync_",
				"func main()",
			},
			runtimeMarkers: []string{
				"forstNodeManifestJSON",
				"nodert.CallSync",
				"legacy/api/checkout.ts",
			},
		},
	}
}

func examplesInDir(t *testing.T) string {
	t.Helper()
	return filepath.Join("..", "..", "..", "examples", "in")
}

func examplesOutDir(t *testing.T) string {
	t.Helper()
	return filepath.Join("..", "..", "..", "examples", "out")
}

type nodeInteropCompileOutput struct {
	Main    string
	Runtime string
	Invoke  string
}

func compileNodeInteropPackageForGolden(t *testing.T, entry, packageRoot string, exportStructFields bool) nodeInteropCompileOutput {
	t.Helper()
	absEntry, err := filepath.Abs(entry)
	if err != nil {
		t.Fatal(err)
	}
	absRoot, err := filepath.Abs(packageRoot)
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
	mainCode, runtimeCode, invokeCode, err := c.CompileWithNodeRuntime()
	if err != nil {
		t.Fatalf("CompileWithNodeRuntime(%s): %v", absEntry, err)
	}
	return nodeInteropCompileOutput{Main: mainCode, Runtime: runtimeCode, Invoke: invokeCode}
}

func nodeRuntimeGoldenPath(mainGoldenPath string) string {
	ext := filepath.Ext(mainGoldenPath)
	base := strings.TrimSuffix(mainGoldenPath, ext)
	if ext == "" {
		return base + "_forst_node_runtime.gen.go"
	}
	return base + "_forst_node_runtime.gen" + ext
}

func verifyNodeInteropPackageCompileGolden(t *testing.T, expected, actual, goldenPath string, markers []string) {
	t.Helper()
	for _, marker := range markers {
		if !strings.Contains(actual, marker) {
			t.Errorf("output missing %q (golden %s)", marker, goldenPath)
		}
	}
	if strings.Contains(actual, "boundaryRoot") {
		t.Errorf("output must not embed boundaryRoot (golden %s)", goldenPath)
	}
	if strings.Contains(actual, "nodert.") && !strings.Contains(goldenPath, "runtime") {
		t.Errorf("main golden must not reference nodert directly (%s)", goldenPath)
	}
	if len(expected) > 0 && len(actual) < len(expected)/2 {
		t.Errorf("output much shorter than golden (%d vs %d bytes)", len(actual), len(expected))
	}
}

func writeNodeInteropPackageGolden(t *testing.T, tc nodeInteropGoldenCase) {
	t.Helper()
	inDir := examplesInDir(t)
	outDir := examplesOutDir(t)
	entry := filepath.Join(inDir, tc.entryRel)
	root := filepath.Dir(entry)
	goldenPath := filepath.Join(outDir, tc.goldenRel)
	runtimeGoldenPath := nodeRuntimeGoldenPath(goldenPath)

	out := compileNodeInteropPackageForGolden(t, entry, root, tc.exportStructFields)
	if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(goldenPath, []byte(out.Main), 0o644); err != nil {
		t.Fatal(err)
	}
	if out.Runtime != "" {
		if err := os.WriteFile(runtimeGoldenPath, []byte(out.Runtime), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote %s", runtimeGoldenPath)
	}
	if out.Invoke != "" {
		invokeGoldenPath := invokeServerGoldenPath(goldenPath)
		if err := os.WriteFile(invokeGoldenPath, []byte(out.Invoke), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote %s", invokeGoldenPath)
	}
	t.Logf("wrote %s", goldenPath)
}

func TestExampleNodeInteropPackagesCompileGolden(t *testing.T) {
	for _, tc := range nodeInteropGoldenCases() {
		t.Run(tc.name, func(t *testing.T) {
			inDir := examplesInDir(t)
			outDir := examplesOutDir(t)
			entry := filepath.Join(inDir, tc.entryRel)
			root := filepath.Dir(entry)
			goldenPath := filepath.Join(outDir, tc.goldenRel)
			runtimeGoldenPath := nodeRuntimeGoldenPath(goldenPath)

			actual := compileNodeInteropPackageForGolden(t, entry, root, tc.exportStructFields)

			if os.Getenv("UPDATE_NODE_INTEROP_GOLDEN") == "1" || os.Getenv("UPDATE_EXAMPLES_GOLDENS") == "1" {
				writeNodeInteropPackageGolden(t, tc)
				return
			}

			expectedMain, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("read golden %s: %v (set UPDATE_NODE_INTEROP_GOLDEN=1 to create)", goldenPath, err)
			}
			verifyNodeInteropPackageCompileGolden(t, string(expectedMain), actual.Main, goldenPath, tc.mainMarkers)

			if len(tc.runtimeMarkers) > 0 {
				if actual.Runtime == "" {
					t.Fatalf("expected node runtime output for %s", tc.name)
				}
				expectedRuntime, err := os.ReadFile(runtimeGoldenPath)
				if err != nil {
					t.Fatalf("read runtime golden %s: %v (set UPDATE_NODE_INTEROP_GOLDEN=1 to create)", runtimeGoldenPath, err)
				}
				verifyNodeInteropPackageCompileGolden(t, string(expectedRuntime), actual.Runtime, runtimeGoldenPath, tc.runtimeMarkers)
			}

			if len(tc.invokeMarkers) > 0 {
				if actual.Invoke == "" {
					t.Fatalf("expected invoke server output for %s", tc.name)
				}
				invokeGoldenPath := invokeServerGoldenPath(goldenPath)
				expectedInvoke, err := os.ReadFile(invokeGoldenPath)
				if err != nil {
					t.Fatalf("read invoke golden %s: %v (set UPDATE_NODE_INTEROP_GOLDEN=1 to create)", invokeGoldenPath, err)
				}
				verifyNodeInteropPackageCompileGolden(t, string(expectedInvoke), actual.Invoke, invokeGoldenPath, tc.invokeMarkers)
			}
		})
	}
}

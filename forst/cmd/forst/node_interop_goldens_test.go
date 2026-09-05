package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forst/internal/codegen/layout"
	"forst/internal/compiler"
	"forst/internal/ftconfig"
)

type nodeInteropGoldenCase struct {
	name               string
	entryRel           string // under examples/in
	goldenRel          string // under examples/out
	packageRootRel     string // optional; defaults to ftconfig boundary from entry, else dirname(entryRel)
	exportStructFields bool
	mainMarkers        []string
	runtimeMarkers     []string
	invokeMarkers      []string
	extraPackageNames  []string
}

func nodeInteropGoldenCases() []nodeInteropGoldenCase {
	return []nodeInteropGoldenCase{
		{
			name:      "bridge-interop",
			entryRel:  "rfc/bridge-interop/main.ft",
			goldenRel: "rfc/bridge-interop/main.go",
			mainMarkers: []string{
				"package main",
				"forst_bridge_callsync_",
				"func main()",
			},
			runtimeMarkers: []string{
				"package main",
				"forstBridgeManifestJSON",
				"forst/bridgert",
				"bridgert.CallSync",
			},
		},
		{
			name:               "sync",
			entryRel:           "rfc/bridge-interop/sync/main.ft",
			goldenRel:          "rfc/bridge-interop/sync/main.go",
			exportStructFields: true,
			mainMarkers: []string{
				"package main",
				"forst_bridge_callsync_",
				"result.Amount",
				"func main()",
			},
			runtimeMarkers: []string{
				"forstBridgeManifestJSON",
				"bridgert.CallSync",
			},
		},
		{
			name:               "promises",
			entryRel:           "rfc/bridge-interop/promises/main.ft",
			goldenRel:          "rfc/bridge-interop/promises/main.go",
			exportStructFields: true,
			mainMarkers: []string{
				"package main",
				"forst_bridge_callasync_",
				"concurrentEcho",
				"func main()",
			},
			runtimeMarkers: []string{
				"forstBridgeManifestJSON",
				"bridgert.CallAsync",
			},
		},
		{
			name:               "generators",
			entryRel:           "rfc/bridge-interop/generators/main.ft",
			goldenRel:          "rfc/bridge-interop/generators/main.go",
			exportStructFields: true,
			mainMarkers: []string{
				"package main",
				"forst_bridge_open_seq_",
				"forstBridgeGenStepDone",
				"func main()",
			},
			runtimeMarkers: []string{
				"forstBridgeManifestJSON",
				"bridgert.OpenSeq",
			},
		},
		{
			name:               "async",
			entryRel:           "rfc/bridge-interop/async/main.ft",
			goldenRel:          "rfc/bridge-interop/async/main.go",
			exportStructFields: true,
			mainMarkers: []string{
				"package main",
				"forst_bridge_callasync_",
				"forst_bridge_open_seq_",
				"func main()",
			},
			runtimeMarkers: []string{
				"forstBridgeManifestJSON",
				"bridgert.CallAsync",
				"bridgert.OpenSeq",
			},
		},
		{
			name:               "host",
			entryRel:           "rfc/bridge-interop/host/main.ft",
			goldenRel:          "rfc/bridge-interop/host/main.go",
			exportStructFields: true,
			mainMarkers: []string{
				"package main",
				"forst_bridge_callsync_",
				"func main()",
			},
			runtimeMarkers: []string{
				"forstBridgeManifestJSON",
				"bridgert.CallSync",
			},
		},
		{
			name:               "remix-serve",
			entryRel:           "rfc/bridge-interop/remix-serve/main/main.ft",
			goldenRel:          "rfc/bridge-interop/remix-serve/main.go",
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
				"forstBridgeManifestJSON",
				"bridgert.CallSync",
				"legacy/todos.js",
			},
			invokeMarkers: []string{
				"invokeembed.MustPrepareEmbeddedHostAuth",
				"invokeembed.MustStartEmbedded",
				"forst_invoke_main_ListTodos",
				"ForstInvokeWaitForShutdown",
			},
		},
		{
			name:               "modules",
			entryRel:           "rfc/bridge-interop/modules/main.ft",
			goldenRel:          "rfc/bridge-interop/modules/main.go",
			exportStructFields: true,
			mainMarkers: []string{
				"package main",
				"forst_bridge_callsync_",
				"func main()",
			},
			runtimeMarkers: []string{
				"forstBridgeManifestJSON",
				"bridgert.CallSync",
				"legacy/api/checkout.js",
			},
		},
		{
			name:               "multi-package-dev",
			entryRel:           "rfc/bridge-interop/multi-package-dev/main/main.ft",
			goldenRel:          "rfc/bridge-interop/multi-package-dev/main.go",
			exportStructFields: true,
			mainMarkers: []string{
				"package main",
				"forst_bridge_callsync_",
				"func main()",
				"ForstInvokeWaitForShutdown",
			},
			runtimeMarkers: []string{
				"forstBridgeManifestJSON",
				"bridgert.CallSync",
				"hostPing",
			},
			invokeMarkers: []string{
				"invokeembed.MustPrepareEmbeddedHostAuth",
				"invokeembed.MustStartEmbedded",
				"forst_invoke_auth_Hash",
				"forst.run.temp/auth",
			},
			extraPackageNames: []string{"auth"},
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
	Extra   map[string]string
}

func nodeInteropPackageRoot(t *testing.T, inDir string, tc nodeInteropGoldenCase) string {
	t.Helper()
	if tc.packageRootRel != "" {
		return filepath.Join(inDir, tc.packageRootRel)
	}
	entryDir := filepath.Dir(filepath.Join(inDir, tc.entryRel))
	// After Go-aligned package layout, entry files live in package subdirs
	// (e.g. main/main.ft). Sandbox linking needs the ftconfig project root.
	if root, err := ftconfig.BoundaryRootFromDir(entryDir); err == nil && root != "" {
		return root
	}
	return entryDir
}

func TestNodeInteropPackageRoot_nestedMainUsesFtconfigBoundary(t *testing.T) {
	inDir := examplesInDir(t)
	cases := []struct {
		name     string
		entryRel string
		wantRel  string
	}{
		{
			name:     "multi-package-dev",
			entryRel: "rfc/bridge-interop/multi-package-dev/main/main.ft",
			wantRel:  "rfc/bridge-interop/multi-package-dev",
		},
		{
			name:     "remix-serve",
			entryRel: "rfc/bridge-interop/remix-serve/main/main.ft",
			wantRel:  "rfc/bridge-interop/remix-serve",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := nodeInteropPackageRoot(t, inDir, nodeInteropGoldenCase{entryRel: tc.entryRel})
			want := filepath.Join(inDir, tc.wantRel)
			if got != want {
				t.Fatalf("package root = %q, want %q (not entry package dir)", got, want)
			}
		})
	}
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
	mainCode, runtimeCode, invokeCode, extraPkgs, _, err := c.CompileWithBridgeRuntime()
	if err != nil {
		t.Fatalf("CompileWithBridgeRuntime(%s): %v", absEntry, err)
	}
	return nodeInteropCompileOutput{Main: mainCode, Runtime: runtimeCode, Invoke: invokeCode, Extra: extraPkgs}
}

func bridgeRuntimeGoldenPath(mainGoldenPath string) string {
	ext := filepath.Ext(mainGoldenPath)
	base := strings.TrimSuffix(mainGoldenPath, ext)
	if ext == "" {
		return base + "_forst_0_bridge_runtime.gen.go"
	}
	return base + "_forst_0_bridge_runtime.gen" + ext
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
	if strings.Contains(actual, "bridgert.") && !strings.Contains(goldenPath, "runtime") {
		t.Errorf("main golden must not reference bridgert directly (%s)", goldenPath)
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
	root := nodeInteropPackageRoot(t, inDir, tc)
	goldenPath := filepath.Join(outDir, tc.goldenRel)
	runtimeGoldenPath := bridgeRuntimeGoldenPath(goldenPath)

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
	if err := writeNodeInteropExtraPackageGoldens(goldenPath, out.Extra); err != nil {
		t.Fatal(err)
	}
	for pkg := range out.Extra {
		t.Logf("wrote %s", filepath.Join(filepath.Dir(goldenPath), pkg, pkg+layout.SuffixGen))
	}
	t.Logf("wrote %s", goldenPath)
}

func writeNodeInteropExtraPackageGoldens(mainGoldenPath string, extraPackages map[string]string) error {
	return compiler.WriteExtraPackagesForOutput(mainGoldenPath, extraPackages)
}

func TestExampleNodeInteropPackagesCompileGolden(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping bridge-interop goldens in -short mode")
	}
	for _, tc := range nodeInteropGoldenCases() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			inDir := examplesInDir(t)
			outDir := examplesOutDir(t)
			entry := filepath.Join(inDir, tc.entryRel)
			root := nodeInteropPackageRoot(t, inDir, tc)
			absRoot, err := filepath.Abs(root)
			if err != nil {
				t.Fatal(err)
			}
			goldenPath := filepath.Join(outDir, tc.goldenRel)
			runtimeGoldenPath := bridgeRuntimeGoldenPath(goldenPath)

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
			verifyCompanionPackageGoBuild(t, companionGoBuildOpts{
				Label:            "fresh compile/" + tc.name,
				MainCode:         actual.Main,
				BridgeRuntimeCode:  actual.Runtime,
				InvokeServerCode: actual.Invoke,
				ExtraPackages:    actual.Extra,
				BoundaryRoot:     absRoot,
			})

			runtimeGoldenPathForBuild := ""
			if len(tc.runtimeMarkers) > 0 {
				runtimeGoldenPathForBuild = runtimeGoldenPath
			}
			invokeGoldenPathForBuild := ""
			if len(tc.invokeMarkers) > 0 {
				invokeGoldenPathForBuild = invokeServerGoldenPath(goldenPath)
			}
			verifyCompanionGoldenFilesGoBuild(t, companionGoldenFilesGoBuildOpts{
				Label:             "committed goldens/" + tc.name,
				MainGoldenPath:    goldenPath,
				RuntimeGoldenPath: runtimeGoldenPathForBuild,
				InvokeGoldenPath:  invokeGoldenPathForBuild,
				BoundaryRoot:      absRoot,
			})

			for _, pkg := range tc.extraPackageNames {
				if actual.Extra[pkg] == "" {
					t.Fatalf("expected extra package %q for %s", pkg, tc.name)
				}
				if !strings.Contains(actual.Extra[pkg], "package "+pkg) {
					t.Fatalf("extra package %q missing package declaration for %s", pkg, tc.name)
				}
				genPath := filepath.Join(filepath.Dir(goldenPath), pkg, pkg+layout.SuffixGen)
				expectedExtra, err := os.ReadFile(genPath)
				if err != nil {
					t.Fatalf("read extra package golden %s: %v (set UPDATE_NODE_INTEROP_GOLDEN=1 to create)", genPath, err)
				}
				if len(expectedExtra) > 0 && len(actual.Extra[pkg]) < len(expectedExtra)/2 {
					t.Errorf("extra package %q much shorter than golden (%d vs %d bytes)", pkg, len(actual.Extra[pkg]), len(expectedExtra))
				}
			}

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

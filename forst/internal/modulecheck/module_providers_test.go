package modulecheck_test

import (
	"os"
	"path/filepath"
	"testing"

	"forst/internal/modulecheck"
	"forst/internal/testmod"
)

func TestCheckModuleProviders_twoPhaseMultiPackageModule(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), testmod.GoModContent("sibling_demo"))
	catalogDir := filepath.Join(dir, "catalog")
	ordersDir := filepath.Join(dir, "orders")
	for _, d := range []string{catalogDir, ordersDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeFile(t, filepath.Join(catalogDir, "types.ft"), `package catalog

type Config = {
	Version: String,
}
`)
	writeFile(t, filepath.Join(ordersDir, "use.ft"), `package orders

import "sibling_demo/catalog"

func Use(cfg: catalog.Config) {
	println(cfg.Version)
}

func main() {
	Use({ Version: "1.0" })
}
`)
	result, err := modulecheck.CheckModuleProviders(nil, modulecheck.Options{ModuleRoot: dir})
	if err != nil {
		t.Fatalf("CheckModuleProviders: %v", err)
	}
	if result.ForstPackageTypeChecker("orders") == nil {
		t.Fatalf("missing orders typechecker; packages=%v", result.ForstPkgToFiles)
	}
	if result.ForstPackageTypeChecker("catalog") == nil {
		t.Fatalf("missing catalog typechecker; packages=%v", result.ForstPkgToFiles)
	}
}

func TestCheckModuleProviders_crossPkg(t *testing.T) {
	root := "../../../examples/in/rfc/providers/cross_pkg"
	result, err := modulecheck.CheckModuleProviders(nil, modulecheck.Options{ModuleRoot: root})
	if err != nil {
		t.Fatalf("CheckModuleProviders: %v", err)
	}
	apiTC := result.ForstPackageTypeChecker("api")
	if apiTC == nil {
		t.Fatal("missing api tc")
	}
	slots := apiTC.FunctionProviders["HandleRequest"]
	if len(slots) != 1 || slots[0].RootIdent != "Logger" {
		t.Fatalf("HandleRequest providers = %v", slots)
	}
	authTC := result.ForstPackageTypeChecker("auth")
	if authTC == nil {
		t.Fatal("missing auth tc")
	}
	logSlots := authTC.FunctionProviders["LogEvent"]
	if len(logSlots) != 1 {
		t.Fatalf("LogEvent providers = %v", logSlots)
	}
	if path, ok := apiTC.ImportPathForLocal("auth"); !ok || path != "providers_cross_pkg_demo/auth" {
		t.Fatalf("api import path for auth = %q ok=%v", path, ok)
	}
	if got := result.ImportPathForForstPackage("api"); got != "providers_cross_pkg_demo/api" {
		t.Fatalf("ImportPathForForstPackage(api) = %q", got)
	}
}

func TestCheckModuleProviders_crossPkg_missingWiringAtRoot(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), testmod.GoModContent("cross_neg"))
	authDir := filepath.Join(dir, "auth")
	apiDir := filepath.Join(dir, "api")
	for _, d := range []string{authDir, apiDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeFile(t, filepath.Join(authDir, "log.ft"), `package auth

type Logger = { Info(msg String) }

func LogEvent(id String) {
	use logger: Logger
	logger.Info(id)
}
`)
	writeFile(t, filepath.Join(apiDir, "handle.ft"), `package api

import "cross_neg/auth"

func HandleRequest(id String) {
	auth.LogEvent(id)
}
`)
	writeFile(t, filepath.Join(apiDir, "handle_test.ft"), `package api

import "testing"

func TestHandleRequest(t *testing.T) {
	HandleRequest("tok")
}
`)
	_, err := modulecheck.CheckModuleProviders(nil, modulecheck.Options{ModuleRoot: dir})
	if err == nil {
		t.Fatal("expected wiring root error")
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCheckModuleProviders_mainWiringRoot(t *testing.T) {
	root := filepath.Join("..", "..", "..", "examples", "in", "rfc", "providers")
	result, err := modulecheck.CheckModuleProviders(nil, modulecheck.Options{ModuleRoot: root})
	if err != nil {
		t.Fatalf("CheckModuleProviders: %v", err)
	}
	tc := result.ForstPackageTypeChecker("providers_demo")
	if tc == nil {
		t.Fatal("missing providers_demo tc")
	}
	if len(tc.FunctionProviders["mainWiringDemo"]) != 0 {
		t.Fatalf("mainWiringDemo should be runnable, providers = %v", tc.FunctionProviders["mainWiringDemo"])
	}
}

func TestCheckModuleProviders_genericSiblingImports(t *testing.T) {
	dir := t.TempDir()
	const mod = "generic_sibling_demo"
	writeFile(t, filepath.Join(dir, "go.mod"), testmod.GoModContent(mod))

	internalDir := filepath.Join(dir, "internal")
	modelsDir := filepath.Join(internalDir, "models")
	apiDir := filepath.Join(internalDir, "api")
	for _, d := range []string{internalDir, modelsDir, apiDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	// Package-named .ft in parent dir: internal/metadata.ft declares package metadata.
	writeFile(t, filepath.Join(internalDir, "metadata.ft"), `package metadata

var Revision = "1.0"
`)
	writeFile(t, filepath.Join(modelsDir, "models.ft"), `package models

type Record = {
	Name: String,
}
`)
	writeFile(t, filepath.Join(apiDir, "api.ft"), `package api

import "generic_sibling_demo/internal/metadata"
import "generic_sibling_demo/internal/models"

func LabelFor(rec models.Record): String {
	return metadata.Revision + rec.Name
}
`)

	result, err := modulecheck.CheckModuleProviders(nil, modulecheck.Options{ModuleRoot: dir})
	if err != nil {
		t.Fatalf("CheckModuleProviders: %v", err)
	}

	importMap := result.ImportPathToForstPkg()
	if importMap[mod+"/internal/metadata"] != "metadata" {
		t.Fatalf("package-named file import path: %v", importMap)
	}

	metadataTC := result.ForstPackageTypeChecker("metadata")
	if metadataTC == nil {
		t.Fatal("missing metadata typechecker")
	}
	if _, ok := metadataTC.CurrentScope().LookupVariableType("Revision"); !ok {
		t.Fatal("Revision not registered in metadata package")
	}

	apiTC := result.ForstPackageTypeChecker("api")
	if apiTC == nil {
		t.Fatal("missing api typechecker")
	}
	path, ok := apiTC.ImportPathForLocal("metadata")
	if !ok {
		t.Fatal("api package missing import local metadata")
	}
	if importMap[path] != "metadata" {
		t.Fatalf("import map[%q] = %q, want metadata", path, importMap[path])
	}

	modelsTC := result.ForstPackageTypeChecker("models")
	if modelsTC == nil {
		t.Fatal("missing models typechecker")
	}
	if _, ok := modelsTC.Defs["Record"]; !ok {
		t.Fatal("Record missing from models Defs")
	}
}

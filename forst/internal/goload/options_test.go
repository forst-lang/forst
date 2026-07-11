package goload

import (
	"errors"
	"testing"

	"golang.org/x/tools/go/packages"
)

func TestWithPackagesLoader_usesInjectedLoader(t *testing.T) {
	t.Parallel()
	called := false
	fake := func(_ *packages.Config, patterns ...string) ([]*packages.Package, error) {
		called = true
		if len(patterns) != 1 || patterns[0] != "fmt" {
			t.Fatalf("patterns = %v", patterns)
		}
		return nil, errInjectedForTest
	}
	_, err := LoadByPkgPath(t.TempDir(), []string{"fmt"}, WithPackagesLoader(fake))
	if !called {
		t.Fatal("expected injected loader to run")
	}
	if !errors.Is(err, errInjectedForTest) {
		t.Fatalf("err = %v", err)
	}
}

func TestSetPackagesLoaderForTest_restoresPrevious(t *testing.T) {
	restore := SetPackagesLoaderForTest(func(_ *packages.Config, _ ...string) ([]*packages.Package, error) {
		return nil, errInjectedForTest
	})
	defer restore()

	_, err := LoadByPkgPath(t.TempDir(), []string{"fmt"})
	if !errors.Is(err, errInjectedForTest) {
		t.Fatalf("global loader err = %v", err)
	}

	restore()

	dir := moduleRootFromWD(t)
	if _, err := LoadByPkgPath(dir, []string{"fmt"}); err != nil {
		t.Fatalf("restored loader failed: %v", err)
	}
}

func TestSetPackagesLoaderForTest_nilUsesPackagesLoad(t *testing.T) {
	restore := SetPackagesLoaderForTest(func(_ *packages.Config, _ ...string) ([]*packages.Package, error) {
		return nil, errInjectedForTest
	})
	defer restore()

	restoreNil := SetPackagesLoaderForTest(nil)
	defer restoreNil()

	dir := moduleRootFromWD(t)
	if _, err := LoadByPkgPath(dir, []string{"fmt"}); err != nil {
		t.Fatalf("packages.Load after nil reset: %v", err)
	}
}

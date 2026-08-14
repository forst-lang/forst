package transformerts

import (
	"strings"
	"testing"
)

func TestValidateForstPackageName_rejectsDollarSign(t *testing.T) {
	err := ValidateForstPackageName("$testing")
	if err == nil {
		t.Fatal("expected error for package containing $")
	}
	if !strings.Contains(err.Error(), "$") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateForstPackageName_rejectsHyphenatedName(t *testing.T) {
	err := ValidateForstPackageName("user-auth")
	if err == nil {
		t.Fatal("expected error for hyphenated package name")
	}
	if !strings.Contains(err.Error(), "Go package name") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateForstPackageName_allowsTestingErrorsEffect(t *testing.T) {
	for _, pkg := range []string{"testing", "errors", "effect", "auth", "main", "_internal"} {
		t.Run(pkg, func(t *testing.T) {
			if err := ValidateForstPackageName(pkg); err != nil {
				t.Fatalf("%q should be allowed: %v", pkg, err)
			}
		})
	}
}

func TestValidateForstPackageName_rejectsGoKeyword(t *testing.T) {
	for _, kw := range []string{"type", "func", "var"} {
		t.Run(kw, func(t *testing.T) {
			err := ValidateForstPackageName(kw)
			if err == nil {
				t.Fatalf("expected error for Go keyword %q", kw)
			}
			if !strings.Contains(err.Error(), "keyword") {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateForstPackageName_allowsJSReservedWords(t *testing.T) {
	for _, name := range []string{"function", "class", "enum", "await", "yield", "export"} {
		t.Run(name, func(t *testing.T) {
			if err := ValidateForstPackageName(name); err != nil {
				t.Fatalf("%q should be allowed with $ namespace aliasing: %v", name, err)
			}
		})
	}
}

func TestPackageNamespaceExport_prefixesDollar(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"auth", "$auth"},
		{"main", "$main"},
		{"function", "$function"},
		{"user_auth", "$user_auth"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			if got := PackageNamespaceExport(tc.in); got != tc.want {
				t.Fatalf("PackageNamespaceExport(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestGeneratedTypeExport_prefixesDollar(t *testing.T) {
	if got := GeneratedTypeExport("ComparePasswordRequest"); got != "$ComparePasswordRequest" {
		t.Fatalf("got %q", got)
	}
}

func TestGeneratedFailureAliasExport_prefixesDollar(t *testing.T) {
	if got := GeneratedFailureAliasExport("ComparePassword"); got != "$ComparePasswordFailure" {
		t.Fatalf("got %q", got)
	}
}

func TestValidateForstPackageName_allowsInvokeCatalogNames(t *testing.T) {
	for _, pkg := range []string{"InvokeRejected", "InvokeFailure"} {
		t.Run(pkg, func(t *testing.T) {
			if err := ValidateForstPackageName(pkg); err != nil {
				t.Fatalf("%q should be allowed as a Forst package name: %v", pkg, err)
			}
		})
	}
}

func TestValidateForstPackageNames_dedupes(t *testing.T) {
	if err := ValidateForstPackageNames([]string{"auth", "auth", "billing"}); err != nil {
		t.Fatal(err)
	}
	err := ValidateForstPackageNames([]string{"auth", "$auth"})
	if err == nil {
		t.Fatal("expected error for $ in one package")
	}
}

func TestPackageNames_dedupesAndSorts(t *testing.T) {
	got := PackageNames([]*TypeScriptOutput{
		{PackageName: "zebra"},
		nil,
		{PackageName: "alpha"},
		{PackageName: "zebra"},
		{PackageName: ""},
	})
	if len(got) != 2 || got[0] != "alpha" || got[1] != "zebra" {
		t.Fatalf("got %v", got)
	}
}

func TestValidateForstPackageNames_allowsDistinctSimilarNames(t *testing.T) {
	if err := ValidateForstPackageNames([]string{"user_auth", "userAuth"}); err != nil {
		t.Fatalf("distinct package names should be allowed: %v", err)
	}
}

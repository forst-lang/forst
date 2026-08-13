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
		if err := ValidateForstPackageName(pkg); err != nil {
			t.Fatalf("%q should be allowed: %v", pkg, err)
		}
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

func TestServiceClassName_pascalCase(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"bcrypt", "bcrypt", "Bcrypt"},
		{"user_auth", "user_auth", "UserAuth"},
		{"userAuth", "userAuth", "UserAuth"},
		{"main", "main", "Main"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ServiceClassName(tc.in); got != tc.want {
				t.Fatalf("ServiceClassName(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestValidateServiceClassNames_rejectsCollision(t *testing.T) {
	err := ValidateServiceClassNames([]string{"user_auth", "userAuth"})
	if err == nil {
		t.Fatal("expected collision error")
	}
	msg := err.Error()
	for _, frag := range []string{"user_auth", "userAuth", "UserAuth"} {
		if !strings.Contains(msg, frag) {
			t.Fatalf("missing %q in %s", frag, msg)
		}
	}
}

func TestValidateServiceClassNames_allowsDistinct(t *testing.T) {
	if err := ValidateServiceClassNames([]string{"bcrypt", "auth"}); err != nil {
		t.Fatal(err)
	}
}

func TestServiceClassName_symbolOnlyNameProducesEmptyString(t *testing.T) {
	for _, name := range []string{"___", "...", "---"} {
		if got := ServiceClassName(name); got != "" {
			t.Fatalf("ServiceClassName(%q) = %q, want empty", name, got)
		}
	}
}

func TestValidateServiceClassNames_collidesWhenBothProduceEmptyClass(t *testing.T) {
	err := ValidateServiceClassNames([]string{"___", "..."})
	if err == nil {
		t.Fatal("expected collision error for symbol-only package names")
	}
	msg := err.Error()
	for _, frag := range []string{"___", "...", "empty Effect service class"} {
		if !strings.Contains(msg, frag) {
			t.Fatalf("missing %q in %s", frag, msg)
		}
	}
}

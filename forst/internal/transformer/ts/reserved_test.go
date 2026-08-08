package transformerts

import (
	"strings"
	"testing"
)

func TestValidateReservedSubpaths_rejectsPackageNamedTesting(t *testing.T) {
	err := ValidateReservedSubpaths([]string{"auth", "testing"}, ReservedClientSubpaths)
	if err == nil {
		t.Fatal("expected error for package named testing")
	}
	msg := err.Error()
	for _, frag := range []string{
		`Forst package "testing"`,
		`"./testing"`,
		"testingSubpath",
	} {
		if !strings.Contains(msg, frag) {
			t.Fatalf("error missing %q:\n%s", frag, msg)
		}
	}
}

func TestValidateReservedSubpaths_allowsPackageNamedTypes(t *testing.T) {
	if err := ValidateReservedSubpaths([]string{"types"}, ReservedClientSubpaths); err != nil {
		t.Fatalf("types must be allowed: %v", err)
	}
}

func TestValidateReservedSubpaths_allowsPackageNamedIndexOrTransportOrCore(t *testing.T) {
	for _, pkg := range []string{"index", "transport", "core"} {
		if err := ValidateReservedSubpaths([]string{pkg}, ReservedClientSubpaths); err != nil {
			t.Fatalf("%s must be allowed: %v", pkg, err)
		}
	}
}

func TestValidateReservedSubpaths_respectsTestingSubpathOverride(t *testing.T) {
	reserved := map[string]string{"test-double": "testing subpath"}
	if err := ValidateReservedSubpaths([]string{"testing"}, reserved); err != nil {
		t.Fatalf("testing allowed when testingSubpath overridden: %v", err)
	}
	err := ValidateReservedSubpaths([]string{"test-double"}, reserved)
	if err == nil {
		t.Fatal("expected error for configured testingSubpath key")
	}
	if !strings.Contains(err.Error(), "test-double") {
		t.Fatalf("unexpected error: %v", err)
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

func TestValidateReservedSubpaths_rejectsCaseVariantOfTesting(t *testing.T) {
	for _, pkg := range []string{"Testing", "TESTING", "tEsTiNg"} {
		err := ValidateReservedSubpaths([]string{pkg}, ReservedClientSubpaths)
		if err == nil {
			t.Fatalf("expected error for case variant %q of reserved testing subpath", pkg)
		}
		if !strings.Contains(err.Error(), pkg) {
			t.Fatalf("error should name package %q: %v", pkg, err)
		}
	}
}

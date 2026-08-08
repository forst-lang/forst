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
	cases := map[string]string{
		"bcrypt":    "Bcrypt",
		"user_auth": "UserAuth",
		"userAuth":  "UserAuth",
		"main":      "Main",
	}
	for in, want := range cases {
		if got := ServiceClassName(in); got != want {
			t.Fatalf("ServiceClassName(%q) = %q, want %q", in, got, want)
		}
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

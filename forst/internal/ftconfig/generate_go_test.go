package ftconfig

import "testing"

func TestGenerateGoConfig_Validate_inactive(t *testing.T) {
	if err := (GenerateGoConfig{}).Validate(); err != nil {
		t.Fatalf("Validate() = %v", err)
	}
}

func TestGenerateGoConfig_Validate_requiresEntryAndOut(t *testing.T) {
	err := (GenerateGoConfig{Out: "./main.go"}).Validate()
	if err == nil {
		t.Fatal("expected error for missing entry")
	}
	err = (GenerateGoConfig{Entry: "./main.ft"}).Validate()
	if err == nil {
		t.Fatal("expected error for missing out")
	}
}

func TestGenerateGoConfig_Validate_rejectsAbsoluteOut(t *testing.T) {
	err := (GenerateGoConfig{
		Entry: "./main.ft",
		Out:   "/tmp/main.go",
	}).Validate()
	if err == nil {
		t.Fatal("expected error for absolute out")
	}
}

func TestGenerateGoConfig_IsConfigured(t *testing.T) {
	if (GenerateGoConfig{Entry: "./a.ft"}).IsConfigured() {
		t.Fatal("entry alone should not configure go emit")
	}
	if !(GenerateGoConfig{Entry: "./a.ft", Out: "./a.go"}).IsConfigured() {
		t.Fatal("entry and out should configure go emit")
	}
}

func TestGenerateGoConfig_EffectiveGoOut(t *testing.T) {
	g := GenerateGoConfig{Out: "out/main.go"}
	got := g.EffectiveGoOut("/proj")
	want := "/proj/out/main.go"
	if got != want {
		t.Fatalf("EffectiveGoOut() = %q want %q", got, want)
	}
}

package ftconfig

import "testing"

func TestGeneratePluginConfig_Validate(t *testing.T) {
	valid := GeneratePluginConfig{Name: "echo", Cmd: "forst-gen-echo", Out: "generated/echo"}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid config: %v", err)
	}
	if valid.EffectiveOutDir("/proj") != "/proj/generated/echo" {
		t.Fatalf("EffectiveOutDir")
	}
	if valid.ResolveCmd("/proj") != "/proj/forst-gen-echo" {
		t.Fatalf("ResolveCmd relative")
	}

	cases := []GeneratePluginConfig{
		{Name: "", Cmd: "x", Out: "out"},
		{Name: "x", Cmd: "", Out: "out"},
		{Name: "x", Cmd: "x", Out: ""},
		{Name: "x", Cmd: "x", Out: "/abs"},
		{Name: "x", Cmd: "x", Out: "../escape"},
	}
	for _, c := range cases {
		if err := c.Validate(); err == nil {
			t.Fatalf("expected error for %#v", c)
		}
	}
}

func TestGenerateConfig_Validate_plugins(t *testing.T) {
	cfg := GenerateConfig{
		PackageName:    DefaultPackageName,
		OutDir:         ".forst/client",
		Link:           "auto",
		Emit:           "js",
		TestingSubpath: "$testing",
		Plugins: []GeneratePluginConfig{
			{Name: "bad", Cmd: "x", Out: "../nope"},
		},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected plugin validation error")
	}
}

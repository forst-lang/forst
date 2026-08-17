package nodert

import (
	"os"
	"strings"
	"testing"

	"forst/internal/ftconfig"
)

func TestGetClient_hostMode_invokeOnlyDisabled(t *testing.T) {
	resetSupervisorForTest()
	t.Cleanup(resetSupervisorForTest)
	t.Setenv(EnvInvokeOnly, "1")
	ConfigureSupervisor(SupervisorConfig{
		HostMode: true,
	})
	_, err := GetClient()
	if err == nil {
		t.Fatal("expected error when FORST_INVOKE_ONLY=1")
	}
	if got := err.Error(); got == "" || !strings.Contains(got, EnvInvokeOnly) {
		t.Fatalf("err = %v", err)
	}
}

func TestInvokeOnlyEnabled(t *testing.T) {
	tests := []struct {
		name string
		env  string
		want bool
	}{
		{name: "unset", env: "", want: false},
		{name: "one", env: "1", want: true},
		{name: "true", env: "true", want: true},
		{name: "zero", env: "0", want: false},
		{name: "false", env: "false", want: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(EnvInvokeOnly, tc.env)
			if got := InvokeOnlyEnabled(); got != tc.want {
				t.Fatalf("InvokeOnlyEnabled() = %v want %v", got, tc.want)
			}
		})
	}
}

func TestInvokeAuthDisabledByEnv(t *testing.T) {
	tests := []struct {
		name string
		env  string
		want bool
	}{
		{name: "unset", env: "", want: false},
		{name: "off", env: "off", want: true},
		{name: "zero", env: "0", want: true},
		{name: "false", env: "false", want: true},
		{name: "on", env: "on", want: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(envInvokeAuth, tc.env)
			if got := invokeAuthDisabledByEnv(); got != tc.want {
				t.Fatalf("invokeAuthDisabledByEnv() = %v want %v", got, tc.want)
			}
		})
	}
}

func TestEnsureEmbeddedHostInvokeAuthRelay_earlyReturns(t *testing.T) {
	tests := []struct {
		name string
		cfg  *ftconfig.Config
		env  map[string]string
	}{
		{name: "nil config", cfg: nil},
		{name: "invoke only", cfg: &ftconfig.Config{Node: ftconfig.NodeConfig{HostMode: true}, Server: ftconfig.ServerConfig{Embedded: true}}, env: map[string]string{EnvInvokeOnly: "1"}},
		{name: "auth off", cfg: &ftconfig.Config{Node: ftconfig.NodeConfig{HostMode: true}, Server: ftconfig.ServerConfig{Embedded: true}}, env: map[string]string{envInvokeAuth: "off"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(EnvInvokeOnly, "")
			t.Setenv(envInvokeAuth, "")
			for k, v := range tc.env {
				t.Setenv(k, v)
			}
			if err := EnsureEmbeddedHostInvokeAuthRelay(tc.cfg); err != nil {
				t.Fatalf("EnsureEmbeddedHostInvokeAuthRelay() = %v", err)
			}
		})
	}
}

func TestEnsureEmbeddedHostInvokeAuthRelay_idempotent(t *testing.T) {
	if !SupportsInvokeAuthFDHandoff() {
		t.Skip("invoke auth fd handoff requires Unix")
	}
	SetActiveHostInvokeAuthRelay(nil)
	t.Cleanup(func() { SetActiveHostInvokeAuthRelay(nil) })

	cfg := &ftconfig.Config{
		Node:   ftconfig.NodeConfig{HostMode: true},
		Server: ftconfig.ServerConfig{Embedded: true},
	}
	if err := EnsureEmbeddedHostInvokeAuthRelay(cfg); err != nil {
		t.Fatalf("first EnsureEmbeddedHostInvokeAuthRelay: %v", err)
	}
	first := os.Getenv(envInvokeAuthFD)
	if first == "" {
		t.Fatal("expected FORST_INVOKE_AUTH_FD after first setup")
	}
	if err := EnsureEmbeddedHostInvokeAuthRelay(cfg); err != nil {
		t.Fatalf("second EnsureEmbeddedHostInvokeAuthRelay: %v", err)
	}
	if got := os.Getenv(envInvokeAuthFD); got != first {
		t.Fatalf("FORST_INVOKE_AUTH_FD changed on second call: %q -> %q", first, got)
	}
}

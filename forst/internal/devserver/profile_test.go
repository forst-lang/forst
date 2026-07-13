package devserver

import (
	"testing"

	"forst/internal/ftconfig"
)

func TestDevserver_profileDetection_executorVsRuntime(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		cfg  *ftconfig.Config
		want Profile
	}{
		{
			name: "pure forst defaults executor",
			cfg:  ftconfig.Default(),
			want: ProfileExecutor,
		},
		{
			name: "embedded selects runtime",
			cfg: func() *ftconfig.Config {
				c := ftconfig.Default()
				c.Server.Embedded = true
				return c
			}(),
			want: ProfileRuntime,
		},
		{
			name: "hostMode selects runtime",
			cfg: func() *ftconfig.Config {
				c := ftconfig.Default()
				c.Node.HostMode = true
				return c
			}(),
			want: ProfileRuntime,
		},
		{
			name: "explicit executor override",
			cfg: func() *ftconfig.Config {
				c := ftconfig.Default()
				c.Server.Embedded = true
				c.Dev.Profile = "executor"
				return c
			}(),
			want: ProfileExecutor,
		},
		{
			name: "explicit runtime override",
			cfg: func() *ftconfig.Config {
				c := ftconfig.Default()
				c.Dev.Profile = "runtime"
				return c
			}(),
			want: ProfileRuntime,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ResolveProfile(tc.cfg); got != tc.want {
				t.Fatalf("ResolveProfile() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestEffectiveListenPort_runtimeUsesInvokePort(t *testing.T) {
	t.Parallel()
	cfg := ftconfig.Default()
	cfg.Server.Embedded = true
	cfg.Server.Port = ""

	if got := EffectiveListenPort(cfg, ""); got != ftconfig.DefaultEmbeddedInvokePort {
		t.Fatalf("EffectiveListenPort() = %q, want %s", got, ftconfig.DefaultEmbeddedInvokePort)
	}
}

package main

import (
	"strings"
	"testing"
)

func TestEffectVersionAtLeast(t *testing.T) {
	cases := []struct {
		name    string
		version string
		floor   string
		want    bool
	}{
		{"equal", "3.17.0", "3.17.0", true},
		{"above", "3.21.4", "3.17.0", true},
		{"below", "3.14.2", "3.17.0", false},
		{"major_above", "4.0.0", "3.17.0", true},
		{"prerelease", "3.17.0-beta.1", "3.17.0", false},
		{"v_prefix", "v3.18.0", "3.17.0", true},
		{"leading_zero", "3.017.0", "3.17.0", true},
		{"empty_segment", "3..0", "3.17.0", false},
		{"overflow_component", "99999999999999999999.0.0", "3.17.0", false},
		{"plus_build_metadata", "3.17.0+abc123", "3.17.0", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := effectVersionAtLeast(tc.version, tc.floor); got != tc.want {
				t.Errorf("effectVersionAtLeast(%q, %q) = %v, want %v", tc.version, tc.floor, got, tc.want)
			}
		})
	}
}

func TestRequireEffectRuntime_missing(t *testing.T) {
	err := requireEffectRuntime(t.TempDir())
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	for _, frag := range []string{"found:    none", ">=3.17.0"} {
		if !strings.Contains(msg, frag) {
			t.Fatalf("missing %q in %s", frag, msg)
		}
	}
}

func TestRequireEffectRuntime_belowFloor(t *testing.T) {
	dir := t.TempDir()
	installEffectFixture(t, dir, "3.14.2")
	err := requireEffectRuntime(dir)
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	for _, frag := range []string{"effect@3.14.2", ">=3.17.0"} {
		if !strings.Contains(msg, frag) {
			t.Fatalf("missing %q in %s", frag, msg)
		}
	}
}

func TestRequireEffectRuntime_ok(t *testing.T) {
	dir := t.TempDir()
	installEffectFixture(t, dir, "3.21.4")
	if err := requireEffectRuntime(dir); err != nil {
		t.Fatal(err)
	}
}

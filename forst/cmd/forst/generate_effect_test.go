package main

import (
	"strings"
	"testing"
)

func TestEffectVersionAtLeast(t *testing.T) {
	cases := []struct {
		version string
		floor   string
		want    bool
	}{
		{"3.17.0", "3.17.0", true},
		{"3.21.4", "3.17.0", true},
		{"3.14.2", "3.17.0", false},
		{"4.0.0", "3.17.0", true},
		{"3.17.0-beta.1", "3.17.0", true},
		{"v3.18.0", "3.17.0", true},
	}
	for _, tc := range cases {
		if got := effectVersionAtLeast(tc.version, tc.floor); got != tc.want {
			t.Fatalf("effectVersionAtLeast(%q, %q) = %v, want %v", tc.version, tc.floor, got, tc.want)
		}
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

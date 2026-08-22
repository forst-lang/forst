package importlocal

import "testing"

func TestDefaultLocalFromModuleID(t *testing.T) {
	tests := []struct {
		moduleID string
		want     string
	}{
		{"legacy/payment.ts", "payment"},
		{"../foo/bar.tsx", "bar"},
		{"effect", "effect"},
		{"@effect/platform", "platform"},
		{"@scope/my-package", "my-package"},
	}
	for _, tt := range tests {
		t.Run(tt.moduleID, func(t *testing.T) {
			if got := DefaultLocalFromModuleID(tt.moduleID); got != tt.want {
				t.Fatalf("DefaultLocalFromModuleID(%q) = %q, want %q", tt.moduleID, got, tt.want)
			}
		})
	}
}

func TestSanitizeSegment(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"my-package", "my_package"},
		{"@scope/pkg", "scope_pkg"},
		{"123abc", "_123abc"},
		{"", "pkg"},
		{"---", "pkg"},
		{"already_ok", "already_ok"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := sanitizeSegment(tt.in); got != tt.want {
				t.Fatalf("sanitizeSegment(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

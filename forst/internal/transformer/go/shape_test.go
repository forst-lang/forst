//go:build go1.18
// +build go1.18

package transformergo

import (
	"testing"
)

func TestCapitalizeFirst(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"foo", "Foo"},
		{"Foo", "Foo"},
		{"f", "F"},
		{"", ""},
		{"ßeta", "SSeta"}, // Unicode: ß uppercases to SS (correct Unicode behavior)
		{"αβγ", "Αβγ"},    // Greek alpha
		{"1foo", "1foo"},  // Non-letter first char
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := capitalizeFirst(tt.input)
			if got != tt.expected {
				t.Errorf("capitalizeFirst(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

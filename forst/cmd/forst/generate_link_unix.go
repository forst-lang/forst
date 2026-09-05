//go:build !windows

package main

import "fmt"

// createJunction is only used on Windows; non-Windows callers use Symlink first.
func createJunction(_, _ string) error {
	return fmt.Errorf("junctions are only supported on windows")
}

package modulecheck

import (
	"testing"

	"forst/internal/typechecker"
)

func TestRevalidateDeferredWiringKeys_emptyPackages(t *testing.T) {
	t.Parallel()
	if err := revalidateDeferredWiringKeys(nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := revalidateDeferredWiringKeys(map[string]*typechecker.TypeChecker{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

package typechecker

import "testing"

func TestInitProvidersInference_preservesDeferAndForstPackage(t *testing.T) {
	t.Parallel()
	tc := New(nil, false)
	tc.SetDeferProvidersWiringRootCheck(true)
	tc.SetForstPackage("demo")
	tc.initProvidersInference()
	if !tc.providersEngine().DeferWiringRootCheck {
		t.Fatal("DeferWiringRootCheck should survive providers re-init")
	}
	if tc.providersEngine().ForstPackage != "demo" {
		t.Fatalf("ForstPackage = %q, want demo", tc.providersEngine().ForstPackage)
	}
}

package nodert

import (
	"strings"
	"testing"
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

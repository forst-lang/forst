package devserver

import (
	"forst/internal/invokeserver"
	"forst/bridgert"
)

func hostInvokeAuthDisabled() bool {
	return invokeserver.AuthDisabledByEnv()
}

func attachHostInvokeAuthRelay(hostOrch *HostOrchestrator, boundaryRoot string) error {
	if hostOrch == nil || !hostModeEnabled(boundaryRoot) || hostInvokeAuthDisabled() {
		return nil
	}
	if !bridgert.SupportsInvokeAuthFDHandoff() {
		// Windows has no ExtraFiles inheritance; connect/env token delivery covers auth there.
		return nil
	}
	if hostOrch.authRelay != nil {
		bridgert.SetActiveHostInvokeAuthRelay(hostOrch.authRelay)
		return nil
	}
	relay, err := bridgert.NewHostInvokeAuthRelay()
	if err != nil {
		return err
	}
	hostOrch.authRelay = relay
	bridgert.SetActiveHostInvokeAuthRelay(relay)
	return nil
}

package devserver

import (
	"forst/internal/invokeserver"
	"forst/nodert"
)

func hostInvokeAuthDisabled() bool {
	return invokeserver.AuthDisabledByEnv()
}

func attachHostInvokeAuthRelay(hostOrch *HostOrchestrator, boundaryRoot string) error {
	if hostOrch == nil || !hostModeEnabled(boundaryRoot) || hostInvokeAuthDisabled() {
		return nil
	}
	if hostOrch.authRelay != nil {
		nodert.SetActiveHostInvokeAuthRelay(hostOrch.authRelay)
		return nil
	}
	relay, err := nodert.NewHostInvokeAuthRelay()
	if err != nil {
		return err
	}
	hostOrch.authRelay = relay
	nodert.SetActiveHostInvokeAuthRelay(relay)
	return nil
}

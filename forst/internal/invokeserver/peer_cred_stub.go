//go:build !linux && !windows

// peer_cred_stub provides a no-op peercred reader on platforms without SO_PEERCRED.
package invokeserver

import "net"

// noopPeerCredentialReader never returns peer credentials.
type noopPeerCredentialReader struct{}

// defaultPeerCredentialReader returns the stub reader on non-Linux, non-Windows builds.
func defaultPeerCredentialReader() peerCredentialReader {
	return noopPeerCredentialReader{}
}

// PeerCredentials always reports ok=false.
func (noopPeerCredentialReader) PeerCredentials(net.Conn) (peerCredentials, bool) {
	return peerCredentials{}, false
}

// currentUID is unused when peercred is unavailable.
func currentUID() int {
	return 0
}

//go:build windows

// peer_cred_windows skips peercred on Windows where invoke uses TCP loopback auth.
package invokeserver

import "net"

// noopPeerCredentialReader never returns peer credentials on Windows.
type noopPeerCredentialReader struct{}

// defaultPeerCredentialReader returns the Windows stub reader.
func defaultPeerCredentialReader() peerCredentialReader {
	return noopPeerCredentialReader{}
}

// PeerCredentials always reports ok=false on Windows.
func (noopPeerCredentialReader) PeerCredentials(net.Conn) (peerCredentials, bool) {
	return peerCredentials{}, false
}

// currentUID is unused on Windows.
func currentUID() int {
	return 0
}

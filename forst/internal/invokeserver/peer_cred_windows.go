//go:build windows

package invokeserver

import "net"

type noopPeerCredentialReader struct{}

func defaultPeerCredentialReader() peerCredentialReader {
	return noopPeerCredentialReader{}
}

func (noopPeerCredentialReader) PeerCredentials(net.Conn) (peerCredentials, bool) {
	return peerCredentials{}, false
}

func currentUID() int {
	return 0
}

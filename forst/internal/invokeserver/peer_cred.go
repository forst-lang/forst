// peer_cred defines Unix socket peer credential checks for invoke auth.
package invokeserver

import "net"

// peerCredentials holds the UID and PID of the remote end of a Unix connection.
type peerCredentials struct {
	UID int
	PID int
}

// peerCredentialReader reads peer credentials from a connected net.Conn.
type peerCredentialReader interface {
	PeerCredentials(conn net.Conn) (peerCredentials, bool)
}

// verifyPeerUID reports whether conn's peer UID matches wantUID.
// When the reader cannot supply credentials, verification is skipped (returns true).
func verifyPeerUID(reader peerCredentialReader, conn net.Conn, wantUID int) bool {
	if reader == nil || conn == nil {
		return true
	}
	creds, ok := reader.PeerCredentials(conn)
	if !ok {
		return true
	}
	return creds.UID == wantUID
}

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

// PeerCredEnforced reports whether Unix transport requires a successful peer UID check.
func PeerCredEnforced() bool {
	return peerCredEnforced
}

// verifyPeerAccess reports whether conn's peer UID matches wantUID on enforced platforms.
// When peerCredEnforced is false, missing credentials are ignored (legacy permissive path).
func verifyPeerAccess(reader peerCredentialReader, conn net.Conn, wantUID int) bool {
	if !peerCredEnforced {
		return verifyPeerUIDPermissive(reader, conn, wantUID)
	}
	if reader == nil || conn == nil {
		return false
	}
	creds, ok := reader.PeerCredentials(conn)
	if !ok {
		return false
	}
	return creds.UID == wantUID
}

// verifyPeerUIDPermissive is the legacy check used when peercred is not enforced.
func verifyPeerUIDPermissive(reader peerCredentialReader, conn net.Conn, wantUID int) bool {
	if reader == nil || conn == nil {
		return true
	}
	creds, ok := reader.PeerCredentials(conn)
	if !ok {
		return true
	}
	return creds.UID == wantUID
}

package invokeserver

import "net"

type peerCredentials struct {
	UID int
	PID int
}

type peerCredentialReader interface {
	PeerCredentials(conn net.Conn) (peerCredentials, bool)
}

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

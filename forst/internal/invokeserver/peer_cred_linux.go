//go:build linux

// peer_cred_linux reads SO_PEERCRED on Linux Unix sockets for invoke auth.
package invokeserver

import (
	"net"
	"os"
	"syscall"
)

// unixPeerCredentialReader implements peerCredentialReader via SO_PEERCRED.
type unixPeerCredentialReader struct{}

// defaultPeerCredentialReader returns the Linux peercred reader.
func defaultPeerCredentialReader() peerCredentialReader {
	return unixPeerCredentialReader{}
}

// PeerCredentials reads UID and PID from conn using getsockopt SO_PEERCRED.
func (unixPeerCredentialReader) PeerCredentials(conn net.Conn) (peerCredentials, bool) {
	uc, ok := conn.(*net.UnixConn)
	if !ok {
		return peerCredentials{}, false
	}
	raw, err := uc.SyscallConn()
	if err != nil {
		return peerCredentials{}, false
	}
	var cred peerCredentials
	var credErr error
	err = raw.Control(func(fd uintptr) {
		u, e := syscall.GetsockoptUcred(int(fd), syscall.SOL_SOCKET, syscall.SO_PEERCRED)
		if e != nil {
			credErr = e
			return
		}
		cred.UID = int(u.Uid)
		cred.PID = int(u.Pid)
	})
	if err != nil || credErr != nil {
		return peerCredentials{}, false
	}
	return cred, true
}

// currentUID returns the effective UID of this process.
func currentUID() int {
	return os.Getuid()
}

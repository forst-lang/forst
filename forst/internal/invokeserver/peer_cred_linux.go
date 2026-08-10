//go:build linux

package invokeserver

import (
	"net"
	"os"
	"syscall"
)

type unixPeerCredentialReader struct{}

func defaultPeerCredentialReader() peerCredentialReader {
	return unixPeerCredentialReader{}
}

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

func currentUID() int {
	return os.Getuid()
}

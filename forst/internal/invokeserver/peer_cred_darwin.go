//go:build darwin

// peer_cred_darwin reads LOCAL_PEERCRED on Darwin Unix sockets for invoke auth.
package invokeserver

import (
	"net"
	"os"

	"golang.org/x/sys/unix"
)

// unixPeerCredentialReader implements peerCredentialReader via LOCAL_PEERCRED.
type unixPeerCredentialReader struct{}

// defaultPeerCredentialReader returns the Darwin peercred reader.
func defaultPeerCredentialReader() peerCredentialReader {
	return unixPeerCredentialReader{}
}

// PeerCredentials reads UID and PID from conn using getsockopt LOCAL_PEERCRED.
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
		x, e := unix.GetsockoptXucred(int(fd), unix.SOL_LOCAL, unix.LOCAL_PEERCRED)
		if e != nil {
			credErr = e
			return
		}
		cred.UID = int(x.Uid)
		pid, e := unix.GetsockoptInt(int(fd), unix.SOL_LOCAL, unix.LOCAL_PEERPID)
		if e == nil {
			cred.PID = pid
		}
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

// peerCredEnforced reports whether Unix peer UID checks are mandatory.
const peerCredEnforced = true

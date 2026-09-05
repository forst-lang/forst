package invokeserver

import (
	"net"
	"os"
	"path/filepath"
	"testing"
)

type stubPeerReader struct {
	creds peerCredentials
	ok    bool
}

func (s stubPeerReader) PeerCredentials(net.Conn) (peerCredentials, bool) {
	return s.creds, s.ok
}

func TestVerifyPeerAccess_enforcedRequiresConnAndCreds(t *testing.T) {
	if !PeerCredEnforced() {
		t.Skip("peercred not enforced on this platform")
	}
	reader := stubPeerReader{creds: peerCredentials{UID: 1000}, ok: true}
	if verifyPeerAccess(reader, nil, 1000) {
		t.Fatal("expected nil conn to fail")
	}
	if verifyPeerAccess(defaultPeerCredentialReader(), &net.TCPConn{}, 1000) {
		t.Fatal("expected missing peercred to fail")
	}
	readerFail := stubPeerReader{ok: false}
	if verifyPeerAccess(readerFail, &net.TCPConn{}, 1000) {
		t.Fatal("expected !ok peercred to fail")
	}
}

func TestVerifyPeerAccess_enforcedUIDMatch(t *testing.T) {
	if !PeerCredEnforced() {
		t.Skip("peercred not enforced on this platform")
	}
	reader := stubPeerReader{creds: peerCredentials{UID: 42}, ok: true}
	if !verifyPeerAccess(reader, &net.TCPConn{}, 42) {
		t.Fatal("expected matching UID")
	}
	if verifyPeerAccess(reader, &net.TCPConn{}, 99) {
		t.Fatal("expected UID mismatch to fail")
	}
}

func TestVerifyPeerAccess_permissiveWhenNotEnforced(t *testing.T) {
	if PeerCredEnforced() {
		t.Skip("only on platforms without peercred enforcement")
	}
	readerFail := stubPeerReader{ok: false}
	if !verifyPeerAccess(readerFail, nil, 0) {
		t.Fatal("expected permissive path to allow missing conn/creds")
	}
}

func TestPeerCredentials_realUnixConn(t *testing.T) {
	if !PeerCredEnforced() {
		t.Skip("peercred not enforced on this platform")
	}
	dir, err := os.MkdirTemp("", "forst-peercred-")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "s.sock")
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	ln, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	serverDone := make(chan net.Conn, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		serverDone <- conn
	}()

	client, err := net.Dial("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })

	serverConn := <-serverDone
	t.Cleanup(func() { _ = serverConn.Close() })

	reader := defaultPeerCredentialReader()
	creds, ok := reader.PeerCredentials(serverConn)
	if !ok {
		t.Fatal("expected peercred on accepted unix conn")
	}
	if creds.UID != currentUID() {
		t.Fatalf("uid = %d, want %d", creds.UID, currentUID())
	}
}

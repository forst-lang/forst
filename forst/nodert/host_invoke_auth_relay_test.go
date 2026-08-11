package nodert

import (
	"bufio"
	"encoding/json"
	"os"
	"testing"
	"time"
)

func TestHostInvokeAuthRelay_forwardsGoHandoffToHost(t *testing.T) {
	goRead, goWrite, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	hostRead, hostWrite, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})
	go func() {
		relayGoInvokeAuthToHost(goRead, hostWrite)
		close(done)
	}()

	payload := hostAuthHandoffPayload{Generation: 7, Token: "AQID"}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := goWrite.Write(append(raw, '\n')); err != nil {
		t.Fatal(err)
	}
	_ = goWrite.Close()

	lineCh := make(chan string, 1)
	go func() {
		scanner := bufio.NewScanner(hostRead)
		if scanner.Scan() {
			lineCh <- scanner.Text()
		}
		close(lineCh)
	}()

	select {
	case line := <-lineCh:
		var got hostAuthHandoffPayload
		if err := json.Unmarshal([]byte(line), &got); err != nil {
			t.Fatal(err)
		}
		if got.Generation != 7 || got.Token != "AQID" {
			t.Fatalf("handoff = %+v", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for host handoff")
	}
	<-done
}

func TestHostInvokeAuthRelay_PrepareGoChild(t *testing.T) {
	relay, err := NewHostInvokeAuthRelay()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = relay.Close() })

	if relay.HostRecvFD() != hostInvokeAuthFDNum {
		t.Fatalf("HostRecvFD = %d", relay.HostRecvFD())
	}
	if relay.GoHandoffEnv() != "FORST_INVOKE_AUTH_FD=3" {
		t.Fatalf("GoHandoffEnv = %q", relay.GoHandoffEnv())
	}
	if relay.HostRecvEnv() != "FORST_INVOKE_AUTH_RECV_FD=3" {
		t.Fatalf("HostRecvEnv = %q", relay.HostRecvEnv())
	}

	goWrite, err := relay.PrepareGoChild()
	if err != nil {
		t.Fatal(err)
	}
	if goWrite == nil {
		t.Fatal("expected go write end")
	}
}

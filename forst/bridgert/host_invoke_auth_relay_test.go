package bridgert

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

func TestPrepareActiveGoInvokeAuthHandoff_nilRelay(t *testing.T) {
	SetActiveHostInvokeAuthRelay(nil)
	t.Cleanup(func() { SetActiveHostInvokeAuthRelay(nil) })

	f, ok := PrepareActiveGoInvokeAuthHandoff()
	if ok || f != nil {
		t.Fatalf("got file=%v ok=%v want nil,false", f, ok)
	}
}

func TestPrepareActiveGoInvokeAuthHandoff_activeRelay(t *testing.T) {
	relay, err := NewHostInvokeAuthRelay()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		SetActiveHostInvokeAuthRelay(nil)
		_ = relay.Close()
	})
	SetActiveHostInvokeAuthRelay(relay)

	f, ok := PrepareActiveGoInvokeAuthHandoff()
	if !ok || f == nil {
		t.Fatalf("got file=%v ok=%v want write end,true", f, ok)
	}
	_ = f.Close()
}

func TestHostInvokeAuthRelay_PrepareGoChild_afterClose(t *testing.T) {
	relay, err := NewHostInvokeAuthRelay()
	if err != nil {
		t.Fatal(err)
	}
	if err := relay.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := relay.PrepareGoChild(); err == nil {
		t.Fatal("expected error after Close")
	}
}

func TestRelayHostInvokeAuthLine_rejectsInvalid(t *testing.T) {
	hostRead, hostWrite, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = hostRead.Close()
		_ = hostWrite.Close()
	})

	if err := relayHostInvokeAuthLine(hostWrite, []byte(`not-json`)); err == nil {
		t.Fatal("expected JSON error")
	}
	if err := relayHostInvokeAuthLine(hostWrite, []byte(`{"generation":0,"token":"AQID"}`)); err == nil {
		t.Fatal("expected generation validation error")
	}
	if err := relayHostInvokeAuthLine(hostWrite, []byte(`{"generation":1,"token":""}`)); err == nil {
		t.Fatal("expected token validation error")
	}
}

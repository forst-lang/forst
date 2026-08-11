package nodert

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"
)

const (
	// EnvInvokeAuthRecvFD names the fd the node host reads invoke auth handoff lines from.
	EnvInvokeAuthRecvFD = "FORST_INVOKE_AUTH_RECV_FD"
	envInvokeAuthFD     = "FORST_INVOKE_AUTH_FD"
	hostInvokeAuthFDNum = 3
)

// SupportsInvokeAuthFDHandoff reports whether auth can be delivered via inherited
// ExtraFiles descriptors. Windows has no ExtraFiles support; use env token delivery there.
func SupportsInvokeAuthFDHandoff() bool {
	return runtime.GOOS != "windows" && runtime.GOOS != "js"
}

type hostAuthHandoffPayload struct {
	Generation uint64 `json:"generation"`
	Token      string `json:"token"`
}

// HostInvokeAuthRelay forwards invoke auth from the embedded go child to the node host
// over inherited pipe fds (no disk token).
type HostInvokeAuthRelay struct {
	mu        sync.Mutex
	closed    bool
	hostWrite *os.File
	hostRead  *os.File
	goWrite   *os.File
	goRead    *os.File
}

// NewHostInvokeAuthRelay creates the host recv pipe and initial go handoff pipe.
func NewHostInvokeAuthRelay() (*HostInvokeAuthRelay, error) {
	hostRead, hostWrite, err := os.Pipe()
	if err != nil {
		return nil, fmt.Errorf("node runtime: host invoke auth pipe: %w", err)
	}
	r := &HostInvokeAuthRelay{
		hostRead:  hostRead,
		hostWrite: hostWrite,
	}
	if _, err := r.PrepareGoChild(); err != nil {
		_ = r.Close()
		return nil, err
	}
	return r, nil
}

// HostRecvFD returns the inherited fd number for the host read end.
func (r *HostInvokeAuthRelay) HostRecvFD() int {
	return hostInvokeAuthFDNum
}

// HostExtraFile is passed to the host child via exec.Cmd.ExtraFiles.
func (r *HostInvokeAuthRelay) HostExtraFile() *os.File {
	if r == nil {
		return nil
	}
	return r.hostRead
}

// PrepareGoChild closes any prior go handoff pipe and returns the write end for a new child.
func (r *HostInvokeAuthRelay) PrepareGoChild() (*os.File, error) {
	if r == nil {
		return nil, fmt.Errorf("node runtime: invoke auth relay is nil")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return nil, fmt.Errorf("node runtime: invoke auth relay is closed")
	}
	if r.goRead != nil {
		_ = r.goRead.Close()
		r.goRead = nil
	}
	if r.goWrite != nil {
		_ = r.goWrite.Close()
		r.goWrite = nil
	}
	goRead, goWrite, err := os.Pipe()
	if err != nil {
		return nil, fmt.Errorf("node runtime: go invoke auth pipe: %w", err)
	}
	r.goRead = goRead
	r.goWrite = goWrite
	go relayGoInvokeAuthToHost(goRead, r.hostWrite)
	return goWrite, nil
}

// GoHandoffEnv returns FORST_INVOKE_AUTH_FD=3 for the embedded go child.
func (r *HostInvokeAuthRelay) GoHandoffEnv() string {
	if r == nil {
		return ""
	}
	return fmt.Sprintf("%s=%d", envInvokeAuthFD, hostInvokeAuthFDNum)
}

// HostRecvEnv returns FORST_INVOKE_AUTH_RECV_FD=3 for the node host shim.
func (r *HostInvokeAuthRelay) HostRecvEnv() string {
	if r == nil {
		return ""
	}
	return fmt.Sprintf("%s=%d", EnvInvokeAuthRecvFD, hostInvokeAuthFDNum)
}

func relayGoInvokeAuthToHost(goRead io.ReadCloser, hostWrite io.Writer) {
	defer func() { _ = goRead.Close() }()
	scanner := bufio.NewScanner(goRead)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		if err := relayHostInvokeAuthLine(hostWrite, line); err != nil {
			return
		}
	}
}

func relayHostInvokeAuthLine(hostWrite io.Writer, line []byte) error {
	var payload hostAuthHandoffPayload
	if err := json.Unmarshal(line, &payload); err != nil {
		return err
	}
	if payload.Generation == 0 || payload.Token == "" {
		return fmt.Errorf("invalid invoke auth handoff")
	}
	if _, err := hostWrite.Write(append(append([]byte(nil), line...), '\n')); err != nil {
		return err
	}
	return nil
}

// Close releases relay pipe ends.
func (r *HostInvokeAuthRelay) Close() error {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closed = true
	var first error
	for _, f := range []*os.File{r.hostRead, r.hostWrite, r.goRead, r.goWrite} {
		if f == nil {
			continue
		}
		if err := f.Close(); err != nil && first == nil {
			first = err
		}
	}
	r.hostRead = nil
	r.hostWrite = nil
	r.goRead = nil
	r.goWrite = nil
	return first
}

var activeHostInvokeAuthRelay struct {
	mu    sync.RWMutex
	relay *HostInvokeAuthRelay
}

// SetActiveHostInvokeAuthRelay registers the relay used by runtime dev go spawns.
func SetActiveHostInvokeAuthRelay(relay *HostInvokeAuthRelay) {
	activeHostInvokeAuthRelay.mu.Lock()
	activeHostInvokeAuthRelay.relay = relay
	activeHostInvokeAuthRelay.mu.Unlock()
}

// ActiveHostInvokeAuthRelay returns the registered relay, if any.
func ActiveHostInvokeAuthRelay() *HostInvokeAuthRelay {
	activeHostInvokeAuthRelay.mu.RLock()
	defer activeHostInvokeAuthRelay.mu.RUnlock()
	return activeHostInvokeAuthRelay.relay
}

// PrepareActiveGoInvokeAuthHandoff prepares the go child handoff fd for a new embedded process.
func PrepareActiveGoInvokeAuthHandoff() (*os.File, bool) {
	relay := ActiveHostInvokeAuthRelay()
	if relay == nil {
		return nil, false
	}
	f, err := relay.PrepareGoChild()
	if err != nil {
		return nil, false
	}
	return f, true
}

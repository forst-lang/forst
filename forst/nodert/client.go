package nodert

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	logrus "github.com/sirupsen/logrus"
)

// DefaultCallTimeout is the maximum wait for a single RPC response.
var DefaultCallTimeout = 5 * time.Minute

type callResult struct {
	result json.RawMessage
	err    error
}

// Client sends RPC requests over length-prefixed protobuf frames (forst-node-proto-v1).
type Client struct {
	mu          sync.Mutex
	rawReader   io.Reader
	writer      io.Writer
	wireProto   string
	nextID      int64
	initialized bool
	manifest    Manifest
	log         *logrus.Logger
	callTimeout atomic.Int64
	maxMsgLen   atomic.Int32

	pendingMu sync.Mutex
	pending   map[int64]chan callResult

	readOnce sync.Once
	readDone chan struct{}
}

// NewClient constructs a client over the given transport streams.
func NewClient(reader io.Reader, writer io.Writer, log *logrus.Logger) *Client {
	if log == nil {
		log = logrus.New()
		log.SetLevel(logrus.ErrorLevel)
	}
	return &Client{
		rawReader: reader,
		writer:    writer,
		wireProto: WireProtocolProtoV1,
		nextID:    1,
		log:       log,
		pending:   make(map[int64]chan callResult),
		readDone:  make(chan struct{}),
	}
}

// SetCallTimeout overrides DefaultCallTimeout for this client (tests).
func (c *Client) SetCallTimeout(d time.Duration) {
	if c == nil {
		return
	}
	c.callTimeout.Store(int64(d))
}

// SetMaxMessageBytes overrides DefaultMaxMsgLen for this client.
func (c *Client) SetMaxMessageBytes(n int) {
	if c == nil {
		return
	}
	c.maxMsgLen.Store(int32(n))
}

func (c *Client) maxMessageBytes() int {
	if c == nil {
		return DefaultMaxMsgLen
	}
	if n := int(c.maxMsgLen.Load()); n > 0 {
		return n
	}
	return DefaultMaxMsgLen
}

func (c *Client) callTimeoutDuration() time.Duration {
	if c == nil {
		return DefaultCallTimeout
	}
	if ns := c.callTimeout.Load(); ns > 0 {
		return time.Duration(ns)
	}
	return DefaultCallTimeout
}

// PendingCount reports in-flight RPC waits (tests).
func (c *Client) PendingCount() int {
	if c == nil {
		return 0
	}
	c.pendingMu.Lock()
	defer c.pendingMu.Unlock()
	return len(c.pending)
}

// Initialized reports whether initialize completed successfully.
func (c *Client) Initialized() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.initialized
}

// Manifest returns the manifest used after initialize.
func (c *Client) Manifest() Manifest {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.manifest
}

// Initialize sends forst.node/initialize with the manifest allowlist.
func (c *Client) Initialize(manifest Manifest, filesExclude []string) error {
	if err := manifest.Validate(); err != nil {
		return fmt.Errorf("manifest: %w", err)
	}
	params := InitializeParams{
		ProtocolVersion:    ProtocolVersion,
		BoundaryRoot:       manifest.BoundaryRoot,
		Manifest:           manifest,
		FilesExclude:       filesExclude,
		SupportedProtocols: defaultSupportedProtocols(),
	}
	result, err := c.Call(MethodInitialize, params)
	if err != nil {
		return err
	}
	var initResult InitializeResult
	if len(result) > 0 {
		if err := json.Unmarshal(result, &initResult); err != nil {
			return fmt.Errorf("decode initialize result: %w", err)
		}
	}
	if initResult.Protocol != "" {
		c.mu.Lock()
		c.wireProto = initResult.Protocol
		c.mu.Unlock()
	}
	c.mu.Lock()
	c.initialized = true
	c.manifest = manifest
	c.mu.Unlock()
	c.log.WithFields(logrus.Fields{
		"component":     "nodert",
		"event":         "initialize",
		"boundary_root": manifest.BoundaryRoot,
		"export_count":  len(manifest.Exports),
		"protocol":      c.wireProto,
	}).Debug("node runtime initialized")
	return nil
}

// Ping sends forst.node/ping.
func (c *Client) Ping() error {
	_, err := c.Call(MethodPing, struct{}{})
	return err
}

// Shutdown sends forst.node/shutdown.
func (c *Client) Shutdown() error {
	_, err := c.Call(MethodShutdown, struct{}{})
	if err != nil {
		return err
	}
	c.mu.Lock()
	c.initialized = false
	c.mu.Unlock()
	c.log.WithFields(logrus.Fields{
		"component": "nodert",
		"event":     "shutdown",
	}).Debug("node runtime shutdown")
	return nil
}

// CallSync validates the manifest and invokes forst.node/call.
func (c *Client) CallSync(moduleID, exportName string, args json.RawMessage) (json.RawMessage, error) {
	c.mu.Lock()
	manifest := c.manifest
	c.mu.Unlock()
	if err := manifest.AllowCall(moduleID, exportName, ExportKindFunction); err != nil {
		c.log.WithFields(logrus.Fields{
			"component":   "nodert",
			"event":       "policy_reject",
			"module_id":   moduleID,
			"export_name": exportName,
		}).Debug("manifest rejected call")
		return nil, err
	}
	params := CallParams{
		ModuleID:   moduleID,
		ExportName: exportName,
		Args:       args,
	}
	result, err := c.Call(MethodCall, params)
	if err != nil {
		return nil, err
	}
	var callResult CallResult
	if err := json.Unmarshal(result, &callResult); err != nil {
		return result, nil
	}
	return callResult.Value, nil
}

// CallAsync validates the manifest and invokes forst.node/callAsync.
func (c *Client) CallAsync(moduleID, exportName string, args json.RawMessage) (json.RawMessage, error) {
	c.mu.Lock()
	manifest := c.manifest
	c.mu.Unlock()
	if err := manifest.AllowCall(moduleID, exportName, ExportKindAsyncFunction); err != nil {
		c.log.WithFields(logrus.Fields{
			"component":   "nodert",
			"event":       "policy_reject",
			"module_id":   moduleID,
			"export_name": exportName,
		}).Debug("manifest rejected callAsync")
		return nil, err
	}
	params := CallParams{
		ModuleID:   moduleID,
		ExportName: exportName,
		Args:       args,
	}
	result, err := c.Call(MethodCallAsync, params)
	if err != nil {
		return nil, err
	}
	var asyncResult CallResult
	if err := json.Unmarshal(result, &asyncResult); err != nil {
		return result, nil
	}
	return asyncResult.Value, nil
}

// Call sends a JSON-RPC request and waits for the correlated response.
func (c *Client) Call(method string, params any) (json.RawMessage, error) {
	if method != MethodInitialize && method != MethodShutdown && !c.Initialized() {
		return nil, ErrInitializeRequired
	}

	paramsJSON, err := marshalParams(params)
	if err != nil {
		return nil, err
	}

	c.ensureReadLoop()

	c.mu.Lock()
	id := c.nextID
	c.nextID++
	c.mu.Unlock()

	ch := make(chan callResult, 1)
	c.pendingMu.Lock()
	c.pending[id] = ch
	c.pendingMu.Unlock()

	c.log.WithFields(logrus.Fields{
		"component":  "nodert",
		"event":      "rpc_send",
		"rpc_method": method,
		"rpc_id":     id,
		"wire_proto": c.wireProto,
	}).Trace("sending rpc request")

	maxLen := c.maxMessageBytes()
	frame, ferr := newProtoRequestFrame(uint64(id), method, paramsJSON)
	if ferr != nil {
		c.removePending(id)
		return nil, ferr
	}
	c.mu.Lock()
	writeErr := WriteProtoFrame(c.writer, frame, maxLen)
	c.mu.Unlock()
	if writeErr != nil {
		c.removePending(id)
		return nil, writeErr
	}

	timer := time.NewTimer(c.callTimeoutDuration())
	defer timer.Stop()

	select {
	case res := <-ch:
		if res.err != nil {
			return nil, res.err
		}
		c.log.WithFields(logrus.Fields{
			"component":  "nodert",
			"event":      "rpc_recv",
			"rpc_method": method,
			"rpc_id":     id,
		}).Trace("received rpc response")
		return res.result, nil
	case <-timer.C:
		c.removePending(id)
		return nil, fmt.Errorf("%w after %s", ErrCallTimeout, c.callTimeoutDuration())
	}
}

func (c *Client) ensureReadLoop() {
	c.readOnce.Do(func() {
		go c.readLoop()
	})
}

func (c *Client) readLoop() {
	defer close(c.readDone)
	for {
		maxLen := c.maxMessageBytes()
		frame, err := ReadProtoFrame(c.rawReader, maxLen)
		if err != nil {
			c.failAllPending(err)
			return
		}
		id := int64(frame.GetId())
		var res callResult
		result, err := decodeProtoOK(frame)
		if err != nil {
			res.err = err
		} else {
			res.result = result
		}

		c.pendingMu.Lock()
		ch, ok := c.pending[id]
		if ok {
			delete(c.pending, id)
		}
		c.pendingMu.Unlock()
		if !ok {
			c.log.WithFields(logrus.Fields{
				"component": "nodert",
				"event":     "orphan_response",
				"rpc_id":    id,
			}).Debug("ignored uncorrelated rpc response")
			continue
		}
		ch <- res
	}
}

func (c *Client) removePending(id int64) {
	c.pendingMu.Lock()
	delete(c.pending, id)
	c.pendingMu.Unlock()
}

func (c *Client) failAllPending(err error) {
	if err != nil && (errors.Is(err, io.EOF) || errors.Is(err, io.ErrClosedPipe)) {
		err = fmt.Errorf("%w: %v", ErrNodeRuntimeDied, err)
	}
	c.pendingMu.Lock()
	defer c.pendingMu.Unlock()
	for id, ch := range c.pending {
		delete(c.pending, id)
		ch <- callResult{err: err}
	}
}

func marshalParams(params any) (json.RawMessage, error) {
	if params == nil {
		return nil, nil
	}
	switch p := params.(type) {
	case json.RawMessage:
		return p, nil
	default:
		data, err := json.Marshal(p)
		if err != nil {
			return nil, fmt.Errorf("marshal rpc params: %w", err)
		}
		return data, nil
	}
}

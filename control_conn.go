package mcpmoqt

import (
	"context"
	"io"
	"sync"
	"sync/atomic"

	"github.com/mengelbart/moqtransport"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Draft: draft-jennings-mcp-over-moqt-00 §2.2.1
// Control tracks: namespace "mcp/<session-id>/control", tracks:
//   - client-to-server
//   - server-to-client
//
// This implementation maps MCP JSON-RPC to MOQT objects on these tracks.
// NOTE: This implementation targets moqtransport (draft-11, moq-00).
// It is not wire-compatible with draft-16 stream/datagram encodings.
const (
	controlNS0 = "mcp"
	controlNS2 = "control"

	trackClientToServer = "client-to-server"
	trackServerToClient = "server-to-client"
)

func controlNamespace(sessionID string) []string {
	return []string{controlNS0, sessionID, controlNS2}
}

type publisherSlot struct {
	mu    sync.Mutex
	pub   moqtransport.Publisher
	ready chan struct{}
}

func newPublisherSlot() *publisherSlot {
	return &publisherSlot{ready: make(chan struct{})}
}

func (s *publisherSlot) set(pub moqtransport.Publisher) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pub != nil {
		return
	}
	s.pub = pub
	close(s.ready)
}

func (s *publisherSlot) get(ctx context.Context) (moqtransport.Publisher, error) {
	select {
	case <-ctx.Done():
		return nil, context.Cause(ctx)
	case <-s.ready:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pub, nil
}

// controlConn implements mcp.Connection over MOQT control tracks.
type controlConn struct {
	conn      moqtransport.Connection
	sessionID string

	recv *moqtransport.RemoteTrack
	send *publisherSlot

	nextGroup atomic.Uint64

	readCtx    context.Context
	cancelRead context.CancelFunc

	closed    bool
	mu        sync.Mutex
	closeOnce sync.Once
	done      chan struct{}
}

func newControlConn(conn moqtransport.Connection, sessionID string, recv *moqtransport.RemoteTrack, send *publisherSlot) *controlConn {
	readCtx, cancel := context.WithCancel(conn.Context())
	return &controlConn{
		conn:       conn,
		sessionID:  sessionID,
		recv:       recv,
		send:       send,
		readCtx:    readCtx,
		cancelRead: cancel,
		done:       make(chan struct{}),
	}
}

func (c *controlConn) SessionID() string { return c.sessionID }

func (c *controlConn) Read(ctx context.Context) (jsonrpc.Message, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-c.done:
		return nil, io.EOF
	default:
	}

	// Ensure that Close unblocks a Read that is waiting in ReadObject.
	combined, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		select {
		case <-c.readCtx.Done():
			cancel()
		case <-combined.Done():
		}
	}()

	obj, err := c.recv.ReadObject(combined)
	if err != nil {
		return nil, err
	}
	return jsonrpc.DecodeMessage(obj.Payload)
}

func (c *controlConn) Write(ctx context.Context, msg jsonrpc.Message) error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return mcp.ErrConnectionClosed
	}
	c.mu.Unlock()

	pub, err := c.send.get(ctx)
	if err != nil {
		return err
	}
	data, err := jsonrpc.EncodeMessage(msg)
	if err != nil {
		return err
	}

	groupID := c.nextGroup.Add(1) - 1
	sg, err := pub.OpenSubgroup(groupID, 0, 0)
	if err != nil {
		return err
	}
	if _, err := sg.WriteObject(0, data); err != nil {
		_ = sg.Close()
		return err
	}
	return sg.Close()
}

func (c *controlConn) Close() error {
	c.closeOnce.Do(func() {
		c.mu.Lock()
		c.closed = true
		close(c.done)
		c.mu.Unlock()
		if c.cancelRead != nil {
			c.cancelRead()
		}
		if c.recv != nil {
			_ = c.recv.Close()
		}
		if c.conn != nil {
			_ = c.conn.CloseWithError(0, "")
		}
	})
	return nil
}

// Copyright 2025 The MCP-MOQT Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package mcpmoqt

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/mengelbart/moqtransport"
)

// MOQTServerTransport implements the server side of MCP over MOQT transport.
type MOQTServerTransport struct {
	// Session is the underlying MOQT session
	Session *moqtransport.Session

	// SessionID is the MCP session identifier
	SessionID string

	// Namespace is the MOQT namespace for MCP tracks
	Namespace []string

	// ControlTrackNamespace is the namespace for control tracks
	ControlTrackNamespace []string

	// SessionConnections maps session IDs to their connections
	SessionConnections map[string]*moqtServerConn

	mu sync.Mutex
}

// NewMOQTServerTransport creates a new server transport.
func NewMOQTServerTransport(session *moqtransport.Session) *MOQTServerTransport {
	sessionID := generateSessionID()
	return &MOQTServerTransport{
		Session:               session,
		SessionID:             sessionID,
		Namespace:             []string{"mcp"},
		ControlTrackNamespace: []string{"mcp"},
		SessionConnections:    make(map[string]*moqtServerConn),
	}
}

// generateSessionID generates a unique session ID.
func generateSessionID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// Connect implements the Transport interface for server.
func (t *MOQTServerTransport) Connect(ctx context.Context) (Connection, error) {
	if t.Session == nil {
		return nil, fmt.Errorf("MOQT session not initialized")
	}

	conn := &moqtServerConn{
		session:               t.Session,
		sessionID:             t.SessionID,
		namespace:             t.Namespace,
		controlTrackNamespace: t.ControlTrackNamespace,
		incoming:              make(chan jsonrpc.Message, 100),
		done:                  make(chan struct{}),
	}

	// Register this connection in the session map
	t.mu.Lock()
	t.SessionConnections[t.SessionID] = conn
	t.mu.Unlock()

	// Initialize control tracks
	if err := conn.initControlTracks(ctx); err != nil {
		t.mu.Lock()
		delete(t.SessionConnections, t.SessionID)
		t.mu.Unlock()
		return nil, fmt.Errorf("failed to initialize control tracks: %w", err)
	}

	return conn, nil
}

// moqtServerConn implements the Connection interface for server.
type moqtServerConn struct {
	session               *moqtransport.Session
	sessionID             string
	namespace             []string
	controlTrackNamespace []string

	// Incoming messages from the client
	incoming chan jsonrpc.Message

	// Control tracks - use Publisher interface
	clientToServerPublisher moqtransport.Publisher
	serverToClientPublisher moqtransport.Publisher
	
	// Track object ID counters
	clientToServerObjectID uint64
	serverToClientObjectID uint64

	mu     sync.Mutex
	closed bool
	done   chan struct{}
}

// SessionID implements Connection.SessionID.
func (c *moqtServerConn) SessionID() string {
	return c.sessionID
}

// initControlTracks initializes the control tracks for MCP communication.
func (c *moqtServerConn) initControlTracks(ctx context.Context) error {
	// The tracks will be created when clients subscribe to them
	// For now, we set up handlers to receive subscriptions
	return nil
}

// Read implements Connection.Read.
func (c *moqtServerConn) Read(ctx context.Context) (jsonrpc.Message, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-c.done:
		return nil, fmt.Errorf("connection closed")
	case msg := <-c.incoming:
		return msg, nil
	}
}

// Write implements Connection.Write.
func (c *moqtServerConn) Write(ctx context.Context, msg jsonrpc.Message) error {
	c.mu.Lock()
	closed := c.closed
	c.mu.Unlock()

	if closed {
		return ErrConnectionClosed
	}

	// Encode message to JSON
	data, err := jsonrpc.EncodeMessage(msg)
	if err != nil {
		return fmt.Errorf("failed to encode message: %w", err)
	}

	// Publish to server-to-client control track
	if c.serverToClientPublisher == nil {
		return fmt.Errorf("server-to-client track not initialized")
	}

	// Use subgroup to write object
	// Group ID 0, Subgroup ID 0, Object ID increments per message
	c.mu.Lock()
	objectID := c.serverToClientObjectID
	c.serverToClientObjectID++
	c.mu.Unlock()

	subgroup, err := c.serverToClientPublisher.OpenSubgroup(0, 0, 1) // priority 1 for control
	if err != nil {
		return fmt.Errorf("failed to open subgroup: %w", err)
	}
	defer subgroup.Close()

	_, err = subgroup.WriteObject(objectID, data)
	return err
}

// Close implements Connection.Close.
func (c *moqtServerConn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true
	close(c.done)

	return nil
}

// getServerConnBySessionID retrieves a server connection by session ID.
// This is a helper function for handlers to find the right connection.
func getServerConnBySessionID(transport *MOQTServerTransport, sessionID string) *moqtServerConn {
	transport.mu.Lock()
	defer transport.mu.Unlock()
	return transport.SessionConnections[sessionID]
}

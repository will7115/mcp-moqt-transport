// Copyright 2025 The MCP-MOQT Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package mcpmoqt

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/mengelbart/moqtransport"
)

// MOQTClientTransport implements the client side of MCP over MOQT transport.
type MOQTClientTransport struct {
	// Session is the underlying MOQT session
	Session *moqtransport.Session

	// SessionID is the MCP session identifier (discovered from server)
	SessionID string

	// Namespace is the MOQT namespace for MCP tracks
	Namespace []string

	// ControlTrackNamespace is the namespace for control tracks
	ControlTrackNamespace []string

	mu sync.Mutex
}

// NewMOQTClientTransport creates a new client transport.
func NewMOQTClientTransport(session *moqtransport.Session) *MOQTClientTransport {
	return &MOQTClientTransport{
		Session:               session,
		Namespace:             []string{"mcp"},
		ControlTrackNamespace: []string{"mcp"},
	}
}

// Connect implements the Transport interface for client.
func (t *MOQTClientTransport) Connect(ctx context.Context) (Connection, error) {
	if t.Session == nil {
		return nil, fmt.Errorf("MOQT session not initialized")
	}

	// First, discover the session ID from the server
	sessionID, err := t.discoverSessionID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover session ID: %w", err)
	}

	t.mu.Lock()
	t.SessionID = sessionID
	t.mu.Unlock()

	conn := &moqtClientConn{
		session:               t.Session,
		sessionID:             sessionID,
		namespace:             t.Namespace,
		controlTrackNamespace: t.ControlTrackNamespace,
		incoming:              make(chan jsonrpc.Message, 100),
		done:                  make(chan struct{}),
	}

	// Initialize control tracks
	if err := conn.initControlTracks(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize control tracks: %w", err)
	}

	return conn, nil
}

// discoverSessionID discovers the session ID from the server using the discovery track.
// According to the draft, clients should use FETCH (not Subscribe) for discovery.
func (t *MOQTClientTransport) discoverSessionID(ctx context.Context) (string, error) {
	// Use FETCH to access discovery track (as per draft specification)
	namespace := []string{"mcp", "discovery"}
	track, err := t.Session.Fetch(ctx, namespace, "sessions")
	if err != nil {
		return "", fmt.Errorf("failed to fetch discovery track: %w", err)
	}
	defer track.Close()

	// Read discovery response
	obj, err := track.ReadObject(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to read discovery response: %w", err)
	}

	// Parse discovery response to extract session ID
	var discoveryResp struct {
		Result struct {
			SessionID string `json:"session_id"`
		} `json:"result"`
	}
	if err := json.Unmarshal(obj.Payload, &discoveryResp); err != nil {
		return "", fmt.Errorf("failed to parse discovery response: %w", err)
	}

	return discoveryResp.Result.SessionID, nil
}

// moqtClientConn implements the Connection interface for client.
type moqtClientConn struct {
	session               *moqtransport.Session
	sessionID             string
	namespace             []string
	controlTrackNamespace []string

	// Incoming messages from the server
	incoming chan jsonrpc.Message

	// Control tracks
	clientToServerPublisher moqtransport.Publisher
	serverToClientTrack     *moqtransport.RemoteTrack

	mu     sync.Mutex
	closed bool
	done   chan struct{}
}

// SessionID implements Connection.SessionID.
func (c *moqtClientConn) SessionID() string {
	return c.sessionID
}

// initControlTracks initializes the control tracks for MCP communication.
func (c *moqtClientConn) initControlTracks(ctx context.Context) error {
	// Subscribe to server-to-client control track
	namespace := append(c.controlTrackNamespace, c.sessionID, "control")
	track, err := c.session.Subscribe(ctx, namespace, "server-to-client")
	if err != nil {
		return fmt.Errorf("failed to subscribe to server-to-client track: %w", err)
	}
	c.serverToClientTrack = track

	// Start reading from server-to-client track
	go c.readFromTrack(ctx, track)

	return nil
}

// readFromTrack reads objects from a MOQT track and converts them to JSON-RPC messages.
func (c *moqtClientConn) readFromTrack(ctx context.Context, track *moqtransport.RemoteTrack) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-c.done:
			return
		default:
			obj, err := track.ReadObject(ctx)
			if err != nil {
				if err == io.EOF {
					return
				}
				// Log error but continue
				continue
			}

			// Decode JSON-RPC message from object payload
			msg, err := jsonrpc.DecodeMessage(obj.Payload)
			if err != nil {
				// Log error but continue
				continue
			}

			select {
			case c.incoming <- msg:
			case <-c.done:
				return
			}
		}
	}
}

// Read implements Connection.Read.
func (c *moqtClientConn) Read(ctx context.Context) (jsonrpc.Message, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-c.done:
		return nil, io.EOF
	case msg := <-c.incoming:
		return msg, nil
	}
}

// Write implements Connection.Write.
func (c *moqtClientConn) Write(ctx context.Context, msg jsonrpc.Message) error {
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

	// Publish to client-to-server control track
	if c.clientToServerPublisher == nil {
		return fmt.Errorf("client-to-server track not initialized")
	}

	// Use subgroup to write object
	c.mu.Lock()
	objectID := uint64(0) // In full implementation, increment per message
	c.mu.Unlock()

	subgroup, err := c.clientToServerPublisher.OpenSubgroup(0, 0, 1) // priority 1 for control
	if err != nil {
		return fmt.Errorf("failed to open subgroup: %w", err)
	}
	defer subgroup.Close()

	_, err = subgroup.WriteObject(objectID, data)
	return err
}

// Close implements Connection.Close.
func (c *moqtClientConn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true
	close(c.done)

	// Close tracks
	if c.serverToClientTrack != nil {
		_ = c.serverToClientTrack.Close()
	}

	return nil
}

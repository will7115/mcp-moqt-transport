// Copyright 2025 The MCP-MOQT Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package mcpmoqt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/mengelbart/moqtransport"
)

// ErrConnectionClosed is returned when sending a message to a connection that
// is closed or in the process of closing.
var ErrConnectionClosed = errors.New("connection closed")

// Transport is the interface for MCP over MOQT transport.
// It creates connections that communicate over MOQT sessions.
type Transport interface {
	// Connect returns a logical JSON-RPC connection over MOQT.
	Connect(ctx context.Context) (Connection, error)
}

// Connection is a logical bidirectional JSON-RPC connection over MOQT.
type Connection interface {
	// Read reads the next message to process off the connection.
	// Connections must allow Read to be called concurrently with Close.
	Read(context.Context) (jsonrpc.Message, error)

	// Write writes a new message to the connection.
	// Write may be called concurrently.
	Write(context.Context, jsonrpc.Message) error

	// Close closes the connection.
	// Close may be called multiple times, potentially concurrently.
	Close() error

	// SessionID returns the MCP session ID for this connection.
	SessionID() string
}

// MOQTTransport is a transport that communicates over MOQT.
type MOQTTransport struct {
	// Session is the underlying MOQT session
	Session *moqtransport.Session

	// SessionID is the MCP session identifier
	SessionID string

	// Namespace is the MOQT namespace for MCP tracks
	Namespace []string

	// ControlTrackNamespace is the namespace for control tracks
	ControlTrackNamespace []string
}

// Connect implements the Transport interface.
func (t *MOQTTransport) Connect(ctx context.Context) (Connection, error) {
	if t.Session == nil {
		return nil, errors.New("MOQT session not initialized")
	}

	conn := &moqtConn{
		session:               t.Session,
		sessionID:             t.SessionID,
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

// moqtConn implements the Connection interface for MOQT transport.
type moqtConn struct {
	session               *moqtransport.Session
	sessionID             string
	namespace             []string
	controlTrackNamespace []string

	// Incoming messages from the server
	incoming chan jsonrpc.Message

	// Control tracks for bidirectional communication
	clientToServerTrack *moqtransport.RemoteTrack
	serverToClientTrack *moqtransport.LocalTrack

	mu     sync.Mutex
	closed bool
	done   chan struct{}
}

// SessionID implements Connection.SessionID.
func (c *moqtConn) SessionID() string {
	return c.sessionID
}

// initControlTracks initializes the control tracks for MCP communication.
func (c *moqtConn) initControlTracks(ctx context.Context) error {
	// Subscribe to server-to-client control track
	namespace := append(c.controlTrackNamespace, c.sessionID, "control")
	track, err := c.session.Subscribe(ctx, namespace, "server-to-client")
	if err != nil {
		return fmt.Errorf("failed to subscribe to server-to-client track: %w", err)
	}
	c.clientToServerTrack = track

	// Start reading from server-to-client track
	go c.readFromTrack(ctx, track)

	return nil
}

// readFromTrack reads objects from a MOQT track and converts them to JSON-RPC messages.
func (c *moqtConn) readFromTrack(ctx context.Context, track *moqtransport.RemoteTrack) {
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
func (c *moqtConn) Read(ctx context.Context) (jsonrpc.Message, error) {
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
func (c *moqtConn) Write(ctx context.Context, msg jsonrpc.Message) error {
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
	// For now, we'll use a simple approach - in a full implementation,
	// we'd need to manage the local track properly
	namespace := append(c.controlTrackNamespace, c.sessionID, "control")
	
	// Create or get local track for client-to-server
	// This is a simplified version - full implementation would cache the track
	// TODO: Implement proper local track management
	
	return fmt.Errorf("write not fully implemented yet")
}

// Close implements Connection.Close.
func (c *moqtConn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true
	close(c.done)

	// Unsubscribe from tracks
	if c.clientToServerTrack != nil {
		_ = c.clientToServerTrack.Unsubscribe()
	}

	return nil
}

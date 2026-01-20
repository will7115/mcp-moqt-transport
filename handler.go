// Copyright 2025 The MCP-MOQT Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package mcpmoqt

import (
	"context"
	"sync"

	"github.com/mengelbart/moqtransport"
)

// MCPSubscribeHandler handles MOQT subscribe messages for MCP tracks.
type MCPSubscribeHandler struct {
	// SessionID is the MCP session identifier
	SessionID string

	// ServerConn is the server connection that will receive messages
	ServerConn *moqtServerConn

	mu sync.Mutex
}

// HandleSubscribe implements moqtransport.SubscribeHandler.
func (h *MCPSubscribeHandler) HandleSubscribe(rw *moqtransport.SubscribeResponseWriter, msg *moqtransport.SubscribeMessage) {
	// Check if this is a control track subscription
	namespace := msg.Namespace
	trackName := msg.Track

	// Expected namespace: ["mcp", sessionID, "control"]
	// Expected track names: "client-to-server" or "server-to-client"
	if len(namespace) >= 3 && namespace[0] == "mcp" && namespace[2] == "control" {
		if trackName == "client-to-server" {
			// Client wants to send messages to server
			h.mu.Lock()
			h.ServerConn.clientToServerPublisher = rw
			h.mu.Unlock()

			// Accept the subscription
			if err := rw.Accept(); err != nil {
				return
			}

			// Start reading from this track
			go h.readFromClientToServerTrack(context.Background(), rw)
		} else if trackName == "server-to-client" {
			// Server wants to send messages to client
			h.mu.Lock()
			h.ServerConn.serverToClientPublisher = rw
			h.mu.Unlock()

			// Accept the subscription
			if err := rw.Accept(); err != nil {
				return
			}
		} else {
			// Unknown track name
			rw.Reject(moqtransport.ErrorCodeSubscribeTrackDoesNotExist, "unknown track")
			return
		}
	} else {
		// Unknown namespace
		rw.Reject(moqtransport.ErrorCodeSubscribeTrackDoesNotExist, "unknown namespace")
		return
	}
}

// readFromClientToServerTrack reads objects from the client-to-server track.
func (h *MCPSubscribeHandler) readFromClientToServerTrack(ctx context.Context, publisher moqtransport.Publisher) {
	// Subscribe to the track to read from it
	// Note: This is a simplified approach - in a full implementation,
	// we would need to properly handle the bidirectional nature of tracks
	// For v0.1.0, we'll use a placeholder
}

// MCPHandler handles general MOQT messages for MCP.
type MCPHandler struct {
	// SessionID is the MCP session identifier
	SessionID string

	// ServerConn is the server connection
	ServerConn *moqtServerConn
}

// Handle implements moqtransport.Handler.
func (h *MCPHandler) Handle(rw moqtransport.ResponseWriter, msg *moqtransport.Message) {
	// Handle various message types
	switch msg.Method {
	case moqtransport.MessageSubscribe:
		// Handled by SubscribeHandler
	default:
		rw.Reject(0, "unsupported message type")
	}
}

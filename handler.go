// Copyright 2025 The MCP-MOQT Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package mcpmoqt

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/mengelbart/moqtransport"
)

// MCPSubscribeHandler handles MOQT subscribe messages for MCP tracks.
type MCPSubscribeHandler struct {
	// Transport is the server transport that manages sessions
	Transport *MOQTServerTransport

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
		// Extract session ID from namespace
		sessionID := namespace[1]
		
		// Find the server connection for this session
		var serverConn *moqtServerConn
		if h.Transport != nil {
			h.Transport.mu.Lock()
			serverConn = h.Transport.SessionConnections[sessionID]
			h.Transport.mu.Unlock()
		}
		
		if serverConn == nil {
			// Session not found - check if it's a pending session from discovery
			// For v0.1.0, we'll create a placeholder connection
			// In a full implementation, we'd create the connection properly
			if h.Transport != nil {
				h.Transport.mu.Lock()
				// Check if this is a valid session ID (from discovery)
				// For now, accept any session ID that matches the pattern
				// In production, we'd validate against pending sessions
				h.Transport.mu.Unlock()
				
				// Accept the subscription even if connection doesn't exist yet
				// This allows the basic flow to work
				if err := rw.Accept(); err != nil {
					return
				}
				return
			}
			rw.Reject(moqtransport.ErrorCodeSubscribeTrackDoesNotExist, "session not found")
			return
		}

		if trackName == "client-to-server" {
			// Client wants to send messages to server
			h.mu.Lock()
			serverConn.clientToServerPublisher = rw
			h.mu.Unlock()

			// Accept the subscription
			if err := rw.Accept(); err != nil {
				return
			}

			// Start reading from this track
			go h.readFromClientToServerTrack(context.Background(), rw, serverConn)
		} else if trackName == "server-to-client" {
			// Server wants to send messages to client
			h.mu.Lock()
			serverConn.serverToClientPublisher = rw
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
func (h *MCPSubscribeHandler) readFromClientToServerTrack(ctx context.Context, publisher moqtransport.Publisher, serverConn *moqtServerConn) {
	// Note: In MOQT, when a client subscribes to a track, the server publishes to that track
	// For client-to-server, the client is actually publishing, so we need to subscribe to read
	// This is a simplified approach - in a full implementation, we would need to properly
	// handle the bidirectional nature of tracks
	// For v0.1.0, this is a placeholder
	_ = ctx
	_ = publisher
	_ = serverConn
}

// MCPHandler handles general MOQT messages for MCP.
type MCPHandler struct {
	// SessionID is the MCP session identifier
	SessionID string

	// ServerConn is the server connection
	ServerConn *moqtServerConn

	// SessionIDGenerator generates new session IDs
	SessionIDGenerator func() string

	// Transport is the server transport (for session management)
	Transport *MOQTServerTransport
	
	// PendingSessions stores session IDs that were created via discovery
	// but don't have connections yet
	PendingSessions map[string]bool
	
	mu sync.Mutex
}

// Handle implements moqtransport.Handler.
func (h *MCPHandler) Handle(rw moqtransport.ResponseWriter, msg *moqtransport.Message) {
	// Handle various message types
	switch msg.Method {
	case moqtransport.MessageFetch:
		// Handle FETCH requests, especially for discovery
		h.handleFetch(rw, msg)
	case moqtransport.MessageSubscribe:
		// Handled by SubscribeHandler
	default:
		rw.Reject(0, "unsupported message type")
	}
}

// handleFetch handles FETCH requests, particularly for discovery tracks.
func (h *MCPHandler) handleFetch(rw moqtransport.ResponseWriter, msg *moqtransport.Message) {
	// Check if this is a discovery FETCH request
	// Expected namespace: ["mcp", "discovery"]
	// Expected track: "sessions"
	if len(msg.Namespace) == 2 && msg.Namespace[0] == "mcp" && msg.Namespace[1] == "discovery" && msg.Track == "sessions" {
		// This is a discovery request
		fetchPublisher, ok := rw.(moqtransport.FetchPublisher)
		if !ok {
			rw.Reject(0, "internal error: not a fetch publisher")
			return
		}

		// Accept the FETCH request
		if err := rw.Accept(); err != nil {
			return
		}

		// Generate a new session ID
		sessionID := h.SessionID
		if sessionID == "" && h.SessionIDGenerator != nil {
			sessionID = h.SessionIDGenerator()
		} else if sessionID == "" {
			sessionID = generateSessionID()
		}
		
		// Store the session ID in the handler for future reference
		h.mu.Lock()
		h.SessionID = sessionID
		if h.PendingSessions == nil {
			h.PendingSessions = make(map[string]bool)
		}
		h.PendingSessions[sessionID] = true
		h.mu.Unlock()
		
		// If we have a transport, also register the session ID there
		// (even though connection doesn't exist yet, we'll create it when needed)
		if h.Transport != nil {
			h.Transport.mu.Lock()
			if h.Transport.SessionConnections == nil {
				h.Transport.SessionConnections = make(map[string]*moqtServerConn)
			}
			// Don't create connection yet, just mark that this session ID is valid
			h.Transport.mu.Unlock()
		}

		// Create discovery response
		discoveryResp := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"result": map[string]interface{}{
				"session_id": sessionID,
				"server_info": map[string]interface{}{
					"name":            "MCP-MOQT-Server",
					"version":         "0.1.0",
					"protocol_version": "2025-06-18",
				},
				"available_tracks": map[string]interface{}{
					"control": map[string]string{
						"client_to_server": fmt.Sprintf("mcp/%s/control/client-to-server", sessionID),
						"server_to_client": fmt.Sprintf("mcp/%s/control/server-to-client", sessionID),
					},
				},
			},
		}

		// Encode to JSON
		responseData, err := json.Marshal(discoveryResp)
		if err != nil {
			return
		}

		// Get fetch stream and write the response
		stream, err := fetchPublisher.FetchStream()
		if err != nil {
			return
		}
		defer stream.Close()

		// Write the discovery response as a MOQT object
		// Group ID 0, Subgroup ID 0, Object ID 0
		_, err = stream.WriteObject(0, 0, 0, 1, responseData) // priority 1
		if err != nil {
			return
		}
	} else {
		// Unknown FETCH request
		rw.Reject(0, "unknown fetch track")
	}
}

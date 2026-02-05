// Copyright 2025 The MCP-MOQT Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package mcpmoqt

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mengelbart/moqtransport"
)

// MOQTClientTransport implements the client side of MCP over MOQT transport.
type MOQTClientTransport struct {
	cfg *transportConfig
}

// NewMOQTClientTransport creates a new client transport.
func NewMOQTClientTransport(opts ...Option) (*MOQTClientTransport, error) {
	cfg, err := applyOptions(roleClient, opts)
	if err != nil {
		return nil, err
	}
	return &MOQTClientTransport{cfg: cfg}, nil
}

// Draft: draft-jennings-mcp-over-moqt-00 §2
// MCP messages are mapped onto MOQT objects via control tracks.
// NOTE: This implementation targets moqtransport (draft-11, moq-00).
// It is not wire-compatible with draft-16 stream/datagram encodings.
// Connect implements the Transport interface for client.
func (t *MOQTClientTransport) Connect(ctx context.Context) (Connection, error) {
	moqConn, _, err := dialMOQT(ctx, t.cfg)
	if err != nil {
		return nil, err
	}

	// First, discover the session ID from the server
	sessionID, session, sendSlot, err := t.bootstrapSession(ctx, moqConn)
	if err != nil {
		return nil, fmt.Errorf("failed to discover session ID: %w", err)
	}

	// Subscribe to recv track (server-to-client).
	recv, err := session.Subscribe(ctx, controlNamespace(sessionID), trackServerToClient)
	if err != nil {
		return nil, err
	}
	return newControlConn(moqConn, sessionID, recv, sendSlot), nil
}

// discoverSessionID discovers the session ID from the server using the discovery track.
// According to the draft, clients should use FETCH (not Subscribe) for discovery.
func (t *MOQTClientTransport) discoverSessionID(ctx context.Context, session *moqtransport.Session) (string, error) {
	// Use FETCH to access discovery track (as per draft specification)
	namespace := []string{"mcp", "discovery"}
	track, err := session.Fetch(ctx, namespace, "sessions")
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

func (t *MOQTClientTransport) bootstrapSession(ctx context.Context, conn moqtransport.Connection) (sessionID string, session *moqtransport.Session, sendSlot *publisherSlot, _ error) {
	// Client must accept subscribe to its send track (client-to-server), so server can obtain a Publisher to receive client messages.
	sendSlot = newPublisherSlot()
	session = &moqtransport.Session{
		Handler:             noOpHandler{},
		SubscribeHandler:    &subscribeHandler{sessionID: "", sendTrack: trackClientToServer, sendSlot: sendSlot},
		InitialMaxRequestID: 100,
	}
	if err := runSession(ctx, session, conn); err != nil {
		return "", nil, nil, err
	}

	// Discover session ID.
	sid, err := t.discoverSessionID(ctx, session)
	if err != nil {
		return "", nil, nil, err
	}

	// Now that we know the session ID, update the subscribe handler to accept peer subscription to our send track.
	// NOTE: moqtransport.Session expects stable handler pointers; we mutate handler fields, not replace handler.
	sh := session.SubscribeHandler.(*subscribeHandler)
	sh.sessionID = sid
	sh.sendTrack = trackClientToServer

	// Server needs a RemoteTrack for client-to-server; it will subscribe, and we will setPublisher via our SubscribeHandler.
	// Nothing else to do here; Connect will subscribe to recv track.
	return sid, session, sendSlot, nil
}

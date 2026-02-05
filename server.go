// Copyright 2025 The MCP-MOQT Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package mcpmoqt

import (
	"context"

	"github.com/mengelbart/moqtransport"
)

// MOQTServerTransport implements the server side of MCP over MOQT transport.
type MOQTServerTransport struct {
	cfg       *transportConfig
	sessionID string
}

// NewMOQTServerTransport creates a new server transport.
func NewMOQTServerTransport(opts ...Option) (*MOQTServerTransport, error) {
	cfg, err := applyOptions(roleServer, opts)
	if err != nil {
		return nil, err
	}
	return &MOQTServerTransport{cfg: cfg, sessionID: generateSessionID()}, nil
}

// Draft: draft-jennings-mcp-over-moqt-00 §2
// MCP messages are mapped onto MOQT objects via control tracks.
// NOTE: This implementation targets moqtransport (draft-11, moq-00).
// It is not wire-compatible with draft-16 stream/datagram encodings.
// Connect implements the Transport interface for server.
func (t *MOQTServerTransport) Connect(ctx context.Context) (Connection, error) {
	moqConn, _, _, err := listenAndAcceptMOQT(ctx, t.cfg)
	if err != nil {
		return nil, err
	}
	session, sendSlot, err := t.bootstrapSession(ctx, moqConn)
	if err != nil {
		return nil, err
	}

	// Server subscribes to recv track (client-to-server) so it can read requests.
	recv, err := session.Subscribe(ctx, controlNamespace(t.sessionID), trackClientToServer)
	if err != nil {
		return nil, err
	}

	return newControlConn(moqConn, t.sessionID, recv, sendSlot), nil
}

func (t *MOQTServerTransport) bootstrapSession(ctx context.Context, conn moqtransport.Connection) (*moqtransport.Session, *publisherSlot, error) {
	sendSlot := newPublisherSlot()
	session := &moqtransport.Session{
		Handler:             &discoveryHandler{sessionID: t.sessionID},
		SubscribeHandler:    &subscribeHandler{sessionID: t.sessionID, sendTrack: trackServerToClient, sendSlot: sendSlot},
		InitialMaxRequestID: 100,
	}
	if err := runSession(ctx, session, conn); err != nil {
		return nil, nil, err
	}
	// Announce the "mcp" namespace so peers may subscribe (best-effort).
	_ = session.Announce(context.Background(), []string{"mcp"})
	return session, sendSlot, nil
}

// Copyright 2025 The MCP-MOQT Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"crypto/tls"
	"log"
	"time"

	"github.com/mcp-moqt/mcp-moqt-transport"
	"github.com/mengelbart/moqtransport"
	"github.com/mengelbart/moqtransport/quicmoq"
	"github.com/quic-go/quic-go"
)

func main() {
	// Create a QUIC listener
	listener, err := quic.ListenAddr("localhost:8080", generateTLSConfig(), &quic.Config{})
	if err != nil {
		log.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	log.Println("MCP over MOQT server listening on localhost:8080")

	for {
		conn, err := listener.Accept(context.Background())
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}

		// Handle each connection in a goroutine
		go handleConnection(conn)
	}
}

func handleConnection(conn quic.Connection) {
	defer conn.CloseWithError(0, "")

	// Create MOQT connection
	moqtConn := quicmoq.NewConnection(conn, moqtransport.PerspectiveServer)

	// Create MOQT session
	session := &moqtransport.Session{
		Handler:                &mcpmoqt.MCPHandler{},
		SubscribeHandler:       &mcpmoqt.MCPSubscribeHandler{},
		InitialMaxRequestID:    100,
	}

	// Run the session
	if err := session.Run(moqtConn); err != nil {
		log.Printf("Session error: %v", err)
		return
	}

	// Create MCP over MOQT server transport
	transport := mcpmoqt.NewMOQTServerTransport(session)

	// Connect
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mcpConn, err := transport.Connect(ctx)
	if err != nil {
		log.Printf("Failed to connect: %v", err)
		return
	}
	defer mcpConn.Close()

	log.Printf("MCP connection established, Session ID: %s", mcpConn.SessionID())

	// Server logic would go here
	// For now, just keep the connection alive
	select {
	case <-ctx.Done():
		return
	}
}

// generateTLSConfig generates a basic TLS config for testing.
// In production, use proper certificates.
func generateTLSConfig() *tls.Config {
	// This is a placeholder - implement proper TLS config
	// For testing, you would generate self-signed certificates
	return &tls.Config{
		InsecureSkipVerify: true, // Only for testing!
	}
}

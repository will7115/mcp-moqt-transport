// Copyright 2025 The MCP-MOQT Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package mcpmoqt

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

func TestMCPServerClient_RunAndPing(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	// Pick an ephemeral port by binding UDP first.
	udpAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	require.NoError(t, err)
	udpConn, err := net.ListenUDP("udp", udpAddr)
	require.NoError(t, err)
	port := udpConn.LocalAddr().(*net.UDPAddr).Port
	_ = udpConn.Close()

	addr := fmt.Sprintf("127.0.0.1:%d", port)

	serverTransport, err := NewMoqTransport(
		RoleServer,
		WithAddr(addr),
	)
	require.NoError(t, err)

	clientTransport, err := NewMoqTransport(
		RoleClient,
		WithAddr(addr),
	)
	require.NoError(t, err)

	server := mcp.NewServer(&mcp.Implementation{Name: "server", Version: "v0.1.2"}, nil)
	client := mcp.NewClient(&mcp.Implementation{Name: "client", Version: "v0.1.2"}, nil)

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.Run(ctx, serverTransport)
	}()

	// Give the server a moment to start listening.
	time.Sleep(150 * time.Millisecond)

	cs, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	defer cs.Close()

	// Basic round-trip: Ping.
	require.NoError(t, cs.Ping(ctx, nil))

	select {
	case err := <-serverErr:
		// Server should normally still be running; if it exited, surface the error.
		require.NoError(t, err)
	default:
	}
}

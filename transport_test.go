// Copyright 2025 The MCP-MOQT Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package mcpmoqt

import (
	"testing"
)

// TestTransportInterface tests that our transport implements the interface correctly.
func TestTransportInterface(t *testing.T) {
	// This is a basic interface test
	// In a full implementation, we would test with a mock MOQT session
	t.Skip("Requires mock MOQT session")
}

// TestConnectionInterface tests that our connection implements the interface correctly.
func TestConnectionInterface(t *testing.T) {
	// This is a basic interface test
	// In a full implementation, we would test with a mock MOQT session
	t.Skip("Requires mock MOQT session")
}

// TestSessionIDGeneration tests session ID generation.
func TestSessionIDGeneration(t *testing.T) {
	sessionID1 := generateSessionID()
	sessionID2 := generateSessionID()

	if sessionID1 == sessionID2 {
		t.Error("Session IDs should be unique")
	}

	if len(sessionID1) == 0 {
		t.Error("Session ID should not be empty")
	}
}

// TestConnectionClose tests that connections can be closed multiple times.
func TestConnectionClose(t *testing.T) {
	// This test would require a mock connection
	t.Skip("Requires mock connection")
}

// TestReadWrite tests basic read/write operations.
func TestReadWrite(t *testing.T) {
	// This test would require a full MOQT session setup
	t.Skip("Requires full MOQT session setup")
}

func TestNewMoqTransportRoles(t *testing.T) {
	serverTransport, err := NewMoqTransport(RoleServer)
	if err != nil {
		t.Fatalf("new server transport: %v", err)
	}
	if serverTransport == nil {
		t.Fatal("server transport is nil")
	}
// === 测试客户端角色 (RoleClient) ===
	clientTransport, err := NewMoqTransport(RoleClient)
	if err != nil {
		t.Fatalf("new client transport: %v", err)
	}
	if clientTransport == nil {
		t.Fatal("client transport is nil")
	}

	if _, err := NewMoqTransport(Role(99)); err == nil {
		t.Fatal("expected error for unknown role")
	}
}

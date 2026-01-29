// Copyright 2025 The MCP-MOQT Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"flag"
	"log"

	mcpmoqt "github.com/mcp-moqt/mcp-moqt-transport"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8080", "listen address")
	flag.Parse()

	tlsCfg, err := mcpmoqt.SelfSignedTLSServerConfig()
	if err != nil {
		log.Fatalf("tls config: %v", err)
	}

	transport, err := mcpmoqt.NewMOQTServerTransport(
		mcpmoqt.WithAddr(*addr),
		mcpmoqt.WithTLSServerConfig(tlsCfg),
	)
	if err != nil {
		log.Fatalf("transport: %v", err)
	}

	server := mcp.NewServer(&mcp.Implementation{Name: "moqt-mcp-server", Version: "v0.0.1"}, nil)
	log.Printf("listening on %s (MOQT/QUIC)", *addr)
	if err := server.Run(context.Background(), transport); err != nil {
		log.Fatalf("server run: %v", err)
	}
}

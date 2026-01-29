// Example client implementation for MCP over MOQT.
package main

import (
	"context"
	"flag"
	"log"
	"time"

	mcpmoqt "github.com/mcp-moqt/mcp-moqt-transport"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8080", "server address")
	timeout := flag.Duration("timeout", 5*time.Second, "connect timeout")
	flag.Parse()

	transport, err := mcpmoqt.NewMOQTClientTransport(
		mcpmoqt.WithAddr(*addr),
	)
	if err != nil {
		log.Fatalf("transport: %v", err)
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "moqt-mcp-client", Version: "v0.0.1"}, nil)

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	cs, err := client.Connect(ctx, transport, nil)
	if err != nil {
		log.Fatalf("client connect: %v", err)
	}
	defer cs.Close()

	if err := cs.Ping(ctx, nil); err != nil {
		log.Fatalf("ping: %v", err)
	}

	log.Printf("connected to %s; ping ok", *addr)
}


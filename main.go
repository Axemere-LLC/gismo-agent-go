// Command gismo-agent-go is a runnable Gismo competitor agent: an MCP
// server exposing get_state/submit_orders/surrender directly to the
// referee (game-and-protocol.md#match-protocol-mcp-tools), mounted at
// /v1. Fork this repo, implement your own agent.Strategy, and wire it in
// below in place of agent.HoldStrategy{} — everything else (the state
// cache, the tool surface, wire encoding) is handled by the agent package.
// When you cut a new agent version, add another agent.Mount rather than
// replacing this one — see agent.VersionedHandler and
// gismo-agent-hosting/docs/serving-multiple-versions.md.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Axemere-LLC/gismo-agent-go/agent"
)

func main() {
	addr := flag.String("addr", agent.DefaultAddr(":8080"), "address to listen on for the agent's MCP endpoint")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s runs a Gismo competitor agent's MCP server.\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Usage:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// HoldStrategy is the unmodified template's stub: every own tank holds
	// its current heading/speed and does not fire. Replace it with your
	// own agent.Strategy implementation.
	handler, err := agent.VersionedHandler(agent.Mount{Path: "/v1", Strategy: agent.HoldStrategy{}})
	if err != nil {
		log.Fatalf("versioned handler: %v", err)
	}

	if key := os.Getenv("MCP_OUTBOUND_KEY"); key != "" {
		handler = agent.BearerAuth(key, handler)
		log.Print("MCP_OUTBOUND_KEY set: requiring a matching Authorization: Bearer header")
	} else {
		log.Print("MCP_OUTBOUND_KEY not set: endpoint is unauthenticated")
	}

	log.Printf("%s %s listening on %s at /v1", agent.Name, agent.Version, *addr)

	if err := agent.ServeHandler(ctx, *addr, handler); err != nil {
		log.Fatalf("serve: %v", err)
	}
}

// Command heuristic runs the gismo-agent-go heuristic reference agent: an
// MCP server that engages the nearest visible enemy and otherwise seeks
// cover in the nearest Forest cell.
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
	"github.com/Axemere-LLC/gismo-agent-go/examples/heuristic"
)

func main() {
	addr := flag.String("addr", agent.DefaultAddr(":8082"), "address to listen on for the agent's MCP endpoint")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s runs the gismo-agent-go heuristic reference agent's MCP server.\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Usage:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	handler, err := agent.VersionedHandler(agent.Mount{Path: "/v1", Strategy: heuristic.Strategy{}})
	if err != nil {
		log.Fatalf("versioned handler: %v", err)
	}

	if key := os.Getenv("MCP_OUTBOUND_KEY"); key != "" {
		handler = agent.BearerAuth(key, handler)
		log.Print("MCP_OUTBOUND_KEY set: requiring a matching Authorization: Bearer header")
	} else {
		log.Print("MCP_OUTBOUND_KEY not set: endpoint is unauthenticated")
	}

	log.Printf("heuristic reference agent listening on %s at /v1", *addr)

	if err := agent.ServeHandler(ctx, *addr, handler); err != nil {
		log.Fatalf("serve: %v", err)
	}
}

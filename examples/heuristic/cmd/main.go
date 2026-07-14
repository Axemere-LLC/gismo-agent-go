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
	addr := flag.String("addr", ":8082", "address to listen on for the agent's MCP endpoint")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s runs the gismo-agent-go heuristic reference agent's MCP server.\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Usage:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Printf("heuristic reference agent listening on %s", *addr)

	if err := agent.Serve(ctx, *addr, heuristic.Strategy{}); err != nil {
		log.Fatalf("serve: %v", err)
	}
}

// Command random runs the gismo-agent-go random reference agent: an MCP
// server that plays every own tank with a random legal order each impulse.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Axemere-LLC/gismo-agent-go/agent"
	"github.com/Axemere-LLC/gismo-agent-go/examples/random"
)

func main() {
	addr := flag.String("addr", agent.DefaultAddr(":8081"), "address to listen on for the agent's MCP endpoint")
	seed := flag.Uint64("seed", 1, "seed for the random strategy's deterministic order sequence")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s runs the gismo-agent-go random reference agent's MCP server.\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Usage:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Printf("random reference agent (seed %d) listening on %s at /v1", *seed, *addr)

	handler := agent.NewHandler(random.New(*seed), agent.WithVersion("v1"))
	mux := http.NewServeMux()
	mux.Handle("/v1", handler)
	mux.Handle("/v1/", handler) // avoid a 301 redirect on the trailing-slash form

	if err := agent.ServeHandler(ctx, *addr, mux); err != nil {
		log.Fatalf("serve: %v", err)
	}
}

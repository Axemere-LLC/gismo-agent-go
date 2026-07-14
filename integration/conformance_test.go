// Package integration drives real gismo-agent-go MCP servers — the
// unmodified template and both bundled reference agents — through
// gismo-contracts' conformance harness over real network (HTTP) transport,
// the same transport agent.Serve exposes to the referee in production. This
// is the "run the conformance harness end to end against both bundled
// reference agents" check repos-and-cicd.md's CI row describes.
package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Axemere-LLC/gismo-contracts/conformance/mockreferee"
	"github.com/Axemere-LLC/gismo-contracts/conformance/schema"

	"github.com/Axemere-LLC/gismo-agent-go/agent"
	"github.com/Axemere-LLC/gismo-agent-go/examples/heuristic"
	"github.com/Axemere-LLC/gismo-agent-go/examples/random"
)

// connectAgent starts strategy's MCP server as a real HTTP listener
// (mirroring agent.Serve) and returns a client session connected to it over
// that HTTP connection — a real network round-trip, not an in-memory pipe.
func connectAgent(t *testing.T, ctx context.Context, strategy agent.Strategy) *mcp.ClientSession {
	t.Helper()

	server := agent.NewServer(strategy)
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, nil)
	httpServer := httptest.NewServer(handler)
	t.Cleanup(httpServer.Close)

	client := mcp.NewClient(&mcp.Implementation{Name: "gismo-agent-go-conformance-test", Version: "test"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: httpServer.URL}, nil)
	if err != nil {
		t.Fatalf("connect to %s over HTTP: %v", httpServer.URL, err)
	}
	t.Cleanup(func() { _ = session.Close() })

	return session
}

func TestConformance_BundledAgentsPassTheHarness(t *testing.T) {
	registry, err := schema.NewRegistry()
	if err != nil {
		t.Fatalf("schema.NewRegistry: %v", err)
	}

	tests := []struct {
		name     string
		strategy agent.Strategy
	}{
		// The unmodified template a competitor forks, before they implement
		// their own Strategy — Stage 4's exit criterion requires this to
		// pass the harness as-is.
		{"unmodified template (HoldStrategy)", agent.HoldStrategy{}},
		{"random reference agent", random.New(1)},
		{"heuristic reference agent", heuristic.Strategy{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			session := connectAgent(t, ctx, tt.strategy)

			if err := mockreferee.Run(ctx, session, registry, mockreferee.Scenario("conformance-test-match", 1)); err != nil {
				t.Errorf("Run() = %v, want nil", err)
			}
		})
	}
}

// Package agent provides the reusable MCP server plumbing every Gismo
// competitor agent needs: the get_state/submit_orders/surrender tool
// surface (game-and-protocol.md#match-protocol-mcp-tools), the match-ID-
// scoped state cache submit_orders depends on, and the one-method Strategy
// hook a competitor implements. It is built on gismo-sdk-go's generated
// wire types, never gismo-platform's private ones — this repo talks MCP
// directly to the referee (bypassing the control plane), so the SDK is the
// only contract dependency.
package agent

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	mcpsdk "github.com/Axemere-LLC/gismo-sdk-go/mcp"
)

// Name and Version identify this agent server to the MCP client (the
// referee) during the initialize handshake.
const Name = "gismo-agent-go"

// Version is this template's own version, distinct from any competitor's
// agent-version registered with the platform.
const Version = "0.1.0"

// config holds the settings an Option can override.
type config struct {
	version string
}

// Option customizes the server built by NewServer (and, through them,
// NewHandler and Serve).
type Option func(*config)

// WithVersion overrides the version this agent reports in serverInfo during
// the MCP initialize handshake. Set it to the version_label the platform
// assigned your registered agent (e.g. "v2") so the referee can tell which
// revision played a match. An empty version is ignored, keeping the template
// default.
func WithVersion(version string) Option {
	return func(cfg *config) {
		if version != "" {
			cfg.version = version
		}
	}
}

// NewServer builds an MCP server implementing get_state, submit_orders, and
// surrender, deciding submit_orders' response by calling strategy.Decide
// against the most recently cached get_state view for that match. A nil
// strategy defaults to HoldStrategy.
func NewServer(strategy Strategy, opts ...Option) *mcp.Server {
	if strategy == nil {
		strategy = HoldStrategy{}
	}
	cache := newStateCache()

	cfg := config{version: Version}
	for _, opt := range opts {
		opt(&cfg)
	}

	server := mcp.NewServer(&mcp.Implementation{Name: Name, Version: cfg.version}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_state",
		Description: "Receive the current battlefield view for a match; echoed back unchanged and cached for the next submit_orders call.",
	}, func(_ context.Context, req *mcp.CallToolRequest, in mcpsdk.StateView) (*mcp.CallToolResult, mcpsdk.StateView, error) {
		cache.store(req.Session, in.MatchId, in)
		return nil, in, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "submit_orders",
		Description: "Return this agent's orders for the given match and impulse, decided from the last cached get_state view.",
	}, func(_ context.Context, req *mcp.CallToolRequest, in mcpsdk.SubmitOrdersRequest) (*mcp.CallToolResult, mcpsdk.SubmitOrdersResponse, error) {
		orders := []mcpsdk.TankOrder{}
		if view, ok := cache.load(req.Session, in.MatchId); ok {
			if decided := strategy.Decide(view); decided != nil {
				orders = decided
			}
		}
		// No cached view yet (e.g. submit_orders arrived before this
		// match's first get_state) falls back to the empty order list
		// above: a safe, always-legal no-op, since an un-ordered tank
		// simply holds its current heading/speed and does not fire.
		return nil, mcpsdk.SubmitOrdersResponse{Impulse: in.Impulse, Orders: orders}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "surrender",
		Description: "Report whether this agent surrenders the given match now.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in mcpsdk.SurrenderRequest) (*mcp.CallToolResult, mcpsdk.SurrenderResponse, error) {
		// The template never surrenders on its own; a competitor's
		// Strategy can extend this behavior later if they want their
		// agent to concede under some condition.
		return nil, mcpsdk.SurrenderResponse{Surrendered: false}, nil
	})

	return server
}

package agent

import (
	"context"
	"errors"
	"net/http"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// DefaultAddr returns ":"+$PORT when PORT is set (Cloud Run and similar
// platforms inject it and expect the process to listen there), or fallback
// otherwise. Intended as a flag.String default so -addr can still override it.
func DefaultAddr(fallback string) string {
	if port := os.Getenv("PORT"); port != "" {
		return ":" + port
	}
	return fallback
}

// NewHandler builds the MCP HTTP handler for strategy, without starting a
// server — for callers that need to wrap it (e.g. with BearerAuth) before
// serving it themselves. Serve is the same handler with no wrapping. Any
// opts are passed through to NewServer.
func NewHandler(strategy Strategy, opts ...Option) http.Handler {
	server := NewServer(strategy, opts...)
	return mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return server
	}, nil)
}

// Serve runs an MCP server built by NewServer(strategy) as an HTTP
// (streamable) endpoint listening on addr, blocking until ctx is canceled
// or the listener fails. This is the transport game-and-protocol.md
// describes an agent exposing to the referee over TLS in production; addr
// is plaintext here — terminate TLS in front of this process (e.g. a load
// balancer or reverse proxy) rather than inside it. Any opts are passed
// through to NewServer.
func Serve(ctx context.Context, addr string, strategy Strategy, opts ...Option) error {
	return ServeHandler(ctx, addr, NewHandler(strategy, opts...))
}

// ServeHandler runs handler as an HTTP server listening on addr, blocking
// until ctx is canceled or the listener fails. Exposed separately from
// Serve so a caller can wrap NewHandler's output (e.g. with BearerAuth)
// before serving it.
func ServeHandler(ctx context.Context, addr string, handler http.Handler) error {
	httpServer := &http.Server{Addr: addr, Handler: handler}

	errCh := make(chan error, 1)
	go func() { errCh <- httpServer.ListenAndServe() }()

	select {
	case <-ctx.Done():
		return httpServer.Shutdown(context.Background())
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

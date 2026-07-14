package agent

import (
	"context"
	"errors"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Serve runs an MCP server built by NewServer(strategy) as an HTTP
// (streamable) endpoint listening on addr, blocking until ctx is canceled
// or the listener fails. This is the transport game-and-protocol.md
// describes an agent exposing to the referee over TLS in production; addr
// is plaintext here — terminate TLS in front of this process (e.g. a load
// balancer or reverse proxy) rather than inside it.
func Serve(ctx context.Context, addr string, strategy Strategy) error {
	server := NewServer(strategy)
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return server
	}, nil)

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

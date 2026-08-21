package agent

import (
	"fmt"
	"net/http"
	"strings"
)

// Mount pairs a URL path with the Strategy that should serve it, for use
// with VersionedHandler. Path is the agent version's URL path, e.g. "/v1";
// the version label reported to the referee in the MCP initialize handshake
// is derived from it (agent.WithVersion(strings.TrimPrefix(Path, "/"))), so
// "/v3" reports "v3" with nothing to keep in sync by hand.
type Mount struct {
	Path     string
	Strategy Strategy
}

// VersionedHandler builds an http.Handler that serves one or more immutable
// agent generations from a single process: each Mount's Strategy is served,
// isolated, at its own Path. This is the shape a competitor's fork grows
// into once it has more than one generation registered with the platform —
// see gismo-agent-hosting/docs/serving-multiple-versions.md.
//
// Each mount gets its own MCP server and state cache (via NewHandler), so
// two mounts never share match state even if they wrap the same Strategy
// value. The returned handler does not apply authentication; wrap it with
// BearerAuth if the deployment requires one.
func VersionedHandler(mounts ...Mount) (http.Handler, error) {
	if len(mounts) == 0 {
		return nil, fmt.Errorf("agent: no mounts given")
	}
	mux := http.NewServeMux()
	seen := make(map[string]bool, len(mounts))
	for _, m := range mounts {
		if !strings.HasPrefix(m.Path, "/") || m.Path == "/" {
			return nil, fmt.Errorf("agent: mount path %q must start with / and not be the bare root", m.Path)
		}
		if seen[m.Path] {
			return nil, fmt.Errorf("agent: duplicate mount path %q", m.Path)
		}
		seen[m.Path] = true

		handler := NewHandler(m.Strategy, WithVersion(strings.TrimPrefix(m.Path, "/")))
		mux.Handle(m.Path, handler)
		mux.Handle(m.Path+"/", handler) // avoid a 301 redirect on the trailing-slash form
	}
	return mux, nil
}

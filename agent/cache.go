package agent

import (
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	mcpsdk "github.com/Axemere-LLC/gismo-sdk-go/mcp"
)

// stateKey scopes a cached view by both the MCP session that stored it and
// the match ID. Scoping by matchID alone would let two sessions serving
// opposite sides of the same self-play match (same agent process, same
// matchID, two distinct sessions from the referee's two Dial calls)
// overwrite each other's cached view, feeding one side's submit_orders the
// other side's OwnTanks list.
type stateKey struct {
	session *mcp.ServerSession
	matchID string
}

// stateCache holds the most recent get_state view per (session, match ID).
//
// submit_orders carries only matchId and impulse — no state
// (game-and-protocol.md#match-protocol-mcp-tools) — so an agent must
// remember the last view it was handed for a match to know what to respond
// with. The referee delivers that view by calling get_state with the view
// as arguments and expecting it echoed back unchanged; get_state is this
// cache's only write path.
type stateCache struct {
	mu    sync.Mutex
	views map[stateKey]mcpsdk.StateView
}

func newStateCache() *stateCache {
	return &stateCache{views: make(map[stateKey]mcpsdk.StateView)}
}

// store records view as the latest state for matchID, scoped to session.
func (c *stateCache) store(session *mcp.ServerSession, matchID string, view mcpsdk.StateView) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.views[stateKey{session, matchID}] = view
}

// load returns the latest state stored for matchID under session, if any.
func (c *stateCache) load(session *mcp.ServerSession, matchID string) (mcpsdk.StateView, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	view, ok := c.views[stateKey{session, matchID}]
	return view, ok
}

// forget discards any cached state for matchID under session, e.g. once a
// match has surrendered or ended, so a long-lived agent process doesn't
// accumulate state for matches that will never be queried again.
func (c *stateCache) forget(session *mcp.ServerSession, matchID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.views, stateKey{session, matchID})
}

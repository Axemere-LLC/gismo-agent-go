package agent

import (
	"sync"

	mcpsdk "github.com/Axemere-LLC/gismo-sdk-go/mcp"
)

// stateCache holds the most recent get_state view per match ID.
//
// submit_orders carries only matchId and impulse — no state
// (game-and-protocol.md#match-protocol-mcp-tools) — so an agent must
// remember the last view it was handed for a match to know what to respond
// with. The referee delivers that view by calling get_state with the view
// as arguments and expecting it echoed back unchanged; get_state is this
// cache's only write path.
type stateCache struct {
	mu    sync.Mutex
	views map[string]mcpsdk.StateView
}

func newStateCache() *stateCache {
	return &stateCache{views: make(map[string]mcpsdk.StateView)}
}

// store records view as the latest state for matchID.
func (c *stateCache) store(matchID string, view mcpsdk.StateView) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.views[matchID] = view
}

// load returns the latest state stored for matchID, if any.
func (c *stateCache) load(matchID string) (mcpsdk.StateView, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	view, ok := c.views[matchID]
	return view, ok
}

// forget discards any cached state for matchID, e.g. once a match has
// surrendered or ended, so a long-lived agent process doesn't accumulate
// state for matches that will never be queried again.
func (c *stateCache) forget(matchID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.views, matchID)
}

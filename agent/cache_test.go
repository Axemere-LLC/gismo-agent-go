package agent

import (
	"reflect"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	mcpsdk "github.com/Axemere-LLC/gismo-sdk-go/mcp"
)

func TestStateCache_LoadMissReturnsFalse(t *testing.T) {
	cache := newStateCache()
	session := &mcp.ServerSession{}

	if _, ok := cache.load(session, "no-such-match"); ok {
		t.Fatal("load() on an empty cache reported ok=true, want false")
	}
}

func TestStateCache_StoreThenLoadRoundTrips(t *testing.T) {
	cache := newStateCache()
	session := &mcp.ServerSession{}
	want := mcpsdk.StateView{MatchId: "m1", Impulse: 3}

	cache.store(session, "m1", want)

	got, ok := cache.load(session, "m1")
	if !ok {
		t.Fatal("load() after store() reported ok=false, want true")
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("load() = %+v, want %+v", got, want)
	}
}

func TestStateCache_ScopedByMatchID(t *testing.T) {
	cache := newStateCache()
	session := &mcp.ServerSession{}
	cache.store(session, "match-a", mcpsdk.StateView{MatchId: "match-a", Impulse: 1})
	cache.store(session, "match-b", mcpsdk.StateView{MatchId: "match-b", Impulse: 9})

	a, ok := cache.load(session, "match-a")
	if !ok || a.Impulse != 1 {
		t.Errorf("load(match-a) = %+v, ok=%v, want Impulse=1, ok=true", a, ok)
	}
	b, ok := cache.load(session, "match-b")
	if !ok || b.Impulse != 9 {
		t.Errorf("load(match-b) = %+v, ok=%v, want Impulse=9, ok=true", b, ok)
	}
}

func TestStateCache_StoreOverwritesPreviousImpulse(t *testing.T) {
	cache := newStateCache()
	session := &mcp.ServerSession{}
	cache.store(session, "m1", mcpsdk.StateView{MatchId: "m1", Impulse: 1})
	cache.store(session, "m1", mcpsdk.StateView{MatchId: "m1", Impulse: 2})

	got, ok := cache.load(session, "m1")
	if !ok || got.Impulse != 2 {
		t.Errorf("load() after two stores = %+v, ok=%v, want Impulse=2, ok=true", got, ok)
	}
}

func TestStateCache_ForgetRemovesMatch(t *testing.T) {
	cache := newStateCache()
	session := &mcp.ServerSession{}
	cache.store(session, "m1", mcpsdk.StateView{MatchId: "m1"})

	cache.forget(session, "m1")

	if _, ok := cache.load(session, "m1"); ok {
		t.Error("load() after forget() reported ok=true, want false")
	}
}

// TestStateCache_ScopedBySession is the regression test for the self-play
// collision bug: two sessions (e.g. one per side, both against the same
// agent process) using the identical matchID must not see each other's
// cached view.
func TestStateCache_ScopedBySession(t *testing.T) {
	cache := newStateCache()
	sideA := &mcp.ServerSession{}
	sideB := &mcp.ServerSession{}

	cache.store(sideA, "m1", mcpsdk.StateView{MatchId: "m1", Impulse: 1, OwnTanks: []mcpsdk.TankView{{Id: 1}}})
	cache.store(sideB, "m1", mcpsdk.StateView{MatchId: "m1", Impulse: 1, OwnTanks: []mcpsdk.TankView{{Id: 8}}})

	a, ok := cache.load(sideA, "m1")
	if !ok || len(a.OwnTanks) != 1 || a.OwnTanks[0].Id != 1 {
		t.Errorf("load(sideA) = %+v, ok=%v, want OwnTanks=[{Id:1}], ok=true", a, ok)
	}
	b, ok := cache.load(sideB, "m1")
	if !ok || len(b.OwnTanks) != 1 || b.OwnTanks[0].Id != 8 {
		t.Errorf("load(sideB) = %+v, ok=%v, want OwnTanks=[{Id:8}], ok=true", b, ok)
	}
}

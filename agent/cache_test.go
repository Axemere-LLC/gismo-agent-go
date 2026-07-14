package agent

import (
	"reflect"
	"testing"

	mcpsdk "github.com/Axemere-LLC/gismo-sdk-go/mcp"
)

func TestStateCache_LoadMissReturnsFalse(t *testing.T) {
	cache := newStateCache()

	if _, ok := cache.load("no-such-match"); ok {
		t.Fatal("load() on an empty cache reported ok=true, want false")
	}
}

func TestStateCache_StoreThenLoadRoundTrips(t *testing.T) {
	cache := newStateCache()
	want := mcpsdk.StateView{MatchId: "m1", Impulse: 3}

	cache.store("m1", want)

	got, ok := cache.load("m1")
	if !ok {
		t.Fatal("load() after store() reported ok=false, want true")
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("load() = %+v, want %+v", got, want)
	}
}

func TestStateCache_ScopedByMatchID(t *testing.T) {
	cache := newStateCache()
	cache.store("match-a", mcpsdk.StateView{MatchId: "match-a", Impulse: 1})
	cache.store("match-b", mcpsdk.StateView{MatchId: "match-b", Impulse: 9})

	a, ok := cache.load("match-a")
	if !ok || a.Impulse != 1 {
		t.Errorf("load(match-a) = %+v, ok=%v, want Impulse=1, ok=true", a, ok)
	}
	b, ok := cache.load("match-b")
	if !ok || b.Impulse != 9 {
		t.Errorf("load(match-b) = %+v, ok=%v, want Impulse=9, ok=true", b, ok)
	}
}

func TestStateCache_StoreOverwritesPreviousImpulse(t *testing.T) {
	cache := newStateCache()
	cache.store("m1", mcpsdk.StateView{MatchId: "m1", Impulse: 1})
	cache.store("m1", mcpsdk.StateView{MatchId: "m1", Impulse: 2})

	got, ok := cache.load("m1")
	if !ok || got.Impulse != 2 {
		t.Errorf("load() after two stores = %+v, ok=%v, want Impulse=2, ok=true", got, ok)
	}
}

func TestStateCache_ForgetRemovesMatch(t *testing.T) {
	cache := newStateCache()
	cache.store("m1", mcpsdk.StateView{MatchId: "m1"})

	cache.forget("m1")

	if _, ok := cache.load("m1"); ok {
		t.Error("load() after forget() reported ok=true, want false")
	}
}

package random

import (
	"testing"

	mcpsdk "github.com/Axemere-LLC/gismo-sdk-go/mcp"

	"github.com/Axemere-LLC/gismo-agent-go/agent"
)

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// fixtureView returns a StateView with own tanks at every combination of
// heading (0-7) and speed (0-3), so every order-legality check below runs
// against every possible starting state a tank could be in.
func fixtureView() mcpsdk.StateView {
	var ownTanks []mcpsdk.TankView
	id := 1
	for heading := 0; heading < 8; heading++ {
		for speed := 0; speed < 4; speed++ {
			ownTanks = append(ownTanks, mcpsdk.TankView{
				Id: id, Heading: heading, Speed: speed, TurretHeading: heading,
			})
			id++
		}
	}
	return mcpsdk.StateView{
		MatchId:  "m1",
		Impulse:  1,
		OwnTanks: ownTanks,
		VisibleTanks: []mcpsdk.TankView{
			{Id: 100, Side: 1, X: 5, Y: 5},
		},
	}
}

func TestStrategy_EveryOrderIsLegal(t *testing.T) {
	view := fixtureView()
	strategy := New(42)

	for impulse := 0; impulse < 20; impulse++ {
		orders := strategy.Decide(view)
		if len(orders) != len(view.OwnTanks) {
			t.Fatalf("impulse %d: len(orders) = %d, want %d", impulse, len(orders), len(view.OwnTanks))
		}
		byID := make(map[int]mcpsdk.TankOrder, len(orders))
		for _, o := range orders {
			byID[o.TankId] = o
		}
		for _, tank := range view.OwnTanks {
			order, ok := byID[tank.Id]
			if !ok {
				t.Errorf("impulse %d: no order for tank %d", impulse, tank.Id)
				continue
			}
			if speedDiff := abs(order.Speed - tank.Speed); speedDiff > 1 {
				t.Errorf("impulse %d, tank %d: speed %d -> %d, diff %d exceeds 1", impulse, tank.Id, tank.Speed, order.Speed, speedDiff)
			}
			allowance := agent.TurnAllowance(order.Speed)
			if turned := agent.TurnDistance(tank.Heading, order.Heading); turned > allowance {
				t.Errorf("impulse %d, tank %d: heading %d -> %d turns %d steps, exceeds allowance %d for ordered speed %d",
					impulse, tank.Id, tank.Heading, order.Heading, turned, allowance, order.Speed)
			}
		}
	}
}

func TestStrategy_NoVisibleEnemiesNeverFires(t *testing.T) {
	view := fixtureView()
	view.VisibleTanks = nil
	strategy := New(1)

	for impulse := 0; impulse < 20; impulse++ {
		for _, order := range strategy.Decide(view) {
			if order.Fire {
				t.Fatalf("impulse %d: order.Fire = true with no visible enemies", impulse)
			}
			if !order.TurretHold {
				t.Fatalf("impulse %d: order.TurretHold = false with no visible enemies", impulse)
			}
		}
	}
}

func TestStrategy_SameSeedIsDeterministic(t *testing.T) {
	view := fixtureView()
	a := New(7)
	b := New(7)

	for impulse := 0; impulse < 10; impulse++ {
		ordersA := a.Decide(view)
		ordersB := b.Decide(view)
		if len(ordersA) != len(ordersB) {
			t.Fatalf("impulse %d: len mismatch %d vs %d", impulse, len(ordersA), len(ordersB))
		}
		for i := range ordersA {
			if ordersA[i] != ordersB[i] {
				t.Errorf("impulse %d, order %d: %+v != %+v for the same seed", impulse, i, ordersA[i], ordersB[i])
			}
		}
	}
}

func TestStrategy_EmptyOwnTanksReturnsEmptyNonNilSlice(t *testing.T) {
	strategy := New(1)
	orders := strategy.Decide(mcpsdk.StateView{})

	if orders == nil {
		t.Fatal("Decide with no own tanks returned a nil slice, want a non-nil empty slice")
	}
	if len(orders) != 0 {
		t.Errorf("len(orders) = %d, want 0", len(orders))
	}
}

package heuristic

import (
	"testing"

	mcpsdk "github.com/Axemere-LLC/gismo-sdk-go/mcp"

	"github.com/Axemere-LLC/gismo-agent-go/agent"
)

func TestStrategy_NoEnemiesNoTerrainHolds(t *testing.T) {
	tank := mcpsdk.TankView{Id: 1, X: 5, Y: 5, Heading: 3, Speed: 2}
	view := mcpsdk.StateView{OwnTanks: []mcpsdk.TankView{tank}}

	orders := Strategy{}.Decide(view)

	if len(orders) != 1 {
		t.Fatalf("len(orders) = %d, want 1", len(orders))
	}
	want := agent.HoldOrders([]mcpsdk.TankView{tank})[0]
	if orders[0] != want {
		t.Errorf("Decide() = %+v, want hold order %+v", orders[0], want)
	}
}

func TestStrategy_EngagesNearestVisibleEnemy(t *testing.T) {
	tank := mcpsdk.TankView{Id: 1, X: 0, Y: 0, Heading: 0, Speed: 2, TurretHeading: 0}
	near := mcpsdk.TankView{Id: 10, X: 5, Y: 0} // due east: heading 2
	far := mcpsdk.TankView{Id: 11, X: 50, Y: 0}
	view := mcpsdk.StateView{
		OwnTanks:     []mcpsdk.TankView{tank},
		VisibleTanks: []mcpsdk.TankView{far, near},
	}

	orders := Strategy{}.Decide(view)
	if len(orders) != 1 {
		t.Fatalf("len(orders) = %d, want 1", len(orders))
	}
	order := orders[0]

	if order.TargetX != near.X || order.TargetY != near.Y {
		t.Errorf("targeted (%d,%d), want the nearer enemy at (%d,%d)", order.TargetX, order.TargetY, near.X, near.Y)
	}
	if order.TurretHold {
		t.Error("TurretHold = true while engaging a visible enemy, want false")
	}
	wantTurretHeading := 2 // East
	if order.TurretHeading != wantTurretHeading {
		t.Errorf("TurretHeading = %d, want %d (East, toward the target)", order.TurretHeading, wantTurretHeading)
	}
}

func TestStrategy_FiresOnlyWhenAlignedInRangeAndArmed(t *testing.T) {
	base := mcpsdk.TankView{Id: 1, X: 0, Y: 0, Heading: 2, Speed: 1, TurretHeading: 2, Ammo: 10}
	enemyInRangeAligned := mcpsdk.TankView{Id: 10, X: 5, Y: 0} // due east, matches TurretHeading 2

	tests := []struct {
		name     string
		tank     mcpsdk.TankView
		enemy    mcpsdk.TankView
		wantFire bool
	}{
		{"aligned, in range, armed", base, enemyInRangeAligned, true},
		{"not aligned yet", func() mcpsdk.TankView { t := base; t.TurretHeading = 6; /* West */ return t }(), enemyInRangeAligned, false},
		{"out of range", base, mcpsdk.TankView{Id: 10, X: 1000, Y: 0}, false},
		{"out of ammo", func() mcpsdk.TankView { t := base; t.Ammo = 0; return t }(), enemyInRangeAligned, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := mcpsdk.StateView{
				OwnTanks:     []mcpsdk.TankView{tt.tank},
				VisibleTanks: []mcpsdk.TankView{tt.enemy},
			}
			orders := Strategy{}.Decide(view)
			if orders[0].Fire != tt.wantFire {
				t.Errorf("Fire = %v, want %v", orders[0].Fire, tt.wantFire)
			}
		})
	}
}

func TestStrategy_SeeksNearestForestWhenNoEnemyVisible(t *testing.T) {
	tank := mcpsdk.TankView{Id: 1, X: 0, Y: 0, Heading: 0, Speed: 1}
	far := mcpsdk.TerrainView{X: 20, Y: 20, Type: forestTerrain}
	near := mcpsdk.TerrainView{X: 3, Y: 0, Type: forestTerrain}
	water := mcpsdk.TerrainView{X: 1, Y: 0, Type: 2} // closer than either forest cell, but not forest
	view := mcpsdk.StateView{
		OwnTanks: []mcpsdk.TankView{tank},
		Terrain:  []mcpsdk.TerrainView{far, water, near},
	}

	orders := Strategy{}.Decide(view)
	if len(orders) != 1 {
		t.Fatalf("len(orders) = %d, want 1", len(orders))
	}
	order := orders[0]
	if !order.TurretHold {
		t.Error("TurretHold = false while seeking cover with no enemy, want true")
	}
	// The tank starts Halted and accelerates to AheadHalf to close on cover,
	// so TurnAllowance permits only a 1-step turn this impulse (see
	// agent.TurnAllowance) — it can't jump straight from North (0) to the
	// target heading East (2), only one step toward it, to Northeast (1).
	wantHeading := 1
	if order.Heading != wantHeading {
		t.Errorf("Heading = %d, want %d (one legal step toward the nearest forest cell)", order.Heading, wantHeading)
	}
}

func TestStrategy_HoldsAtForestCellItAlreadyOccupies(t *testing.T) {
	tank := mcpsdk.TankView{Id: 1, X: 3, Y: 0, Heading: 5, Speed: 1}
	view := mcpsdk.StateView{
		OwnTanks: []mcpsdk.TankView{tank},
		Terrain:  []mcpsdk.TerrainView{{X: 3, Y: 0, Type: forestTerrain}},
	}

	orders := Strategy{}.Decide(view)
	want := agent.HoldOrders([]mcpsdk.TankView{tank})[0]
	if orders[0] != want {
		t.Errorf("Decide() at own forest cell = %+v, want hold order %+v", orders[0], want)
	}
}

func TestStrategy_EveryOrderIsLegal(t *testing.T) {
	tanks := []mcpsdk.TankView{
		{Id: 1, X: 0, Y: 0, Heading: 0, Speed: 0, TurretHeading: 0},
		{Id: 2, X: 10, Y: 10, Heading: 5, Speed: 3, TurretHeading: 5},
	}
	view := mcpsdk.StateView{
		OwnTanks:     tanks,
		VisibleTanks: []mcpsdk.TankView{{Id: 100, X: 7, Y: 3}},
		Terrain:      []mcpsdk.TerrainView{{X: 2, Y: 2, Type: forestTerrain}},
	}

	orders := Strategy{}.Decide(view)
	for i, order := range orders {
		tank := tanks[i]
		if diff := order.Speed - tank.Speed; diff > 1 || diff < -1 {
			t.Errorf("tank %d: speed %d -> %d, illegal diff %d", tank.Id, tank.Speed, order.Speed, diff)
		}
		if turned := agent.TurnDistance(tank.Heading, order.Heading); turned > agent.TurnAllowance(order.Speed) {
			t.Errorf("tank %d: heading %d -> %d turns %d steps, exceeds allowance for ordered speed %d", tank.Id, tank.Heading, order.Heading, turned, order.Speed)
		}
	}
}

package agent

import (
	"testing"

	mcpsdk "github.com/Axemere-LLC/gismo-sdk-go/mcp"
)

func TestHoldOrders_OneOrderPerOwnTank(t *testing.T) {
	ownTanks := []mcpsdk.TankView{
		{Id: 1, Heading: 2, Speed: 3},
		{Id: 2, Heading: 5, Speed: 1},
	}

	orders := HoldOrders(ownTanks)

	if len(orders) != len(ownTanks) {
		t.Fatalf("len(orders) = %d, want %d", len(orders), len(ownTanks))
	}
	for i, tank := range ownTanks {
		order := orders[i]
		if order.TankId != tank.Id {
			t.Errorf("orders[%d].TankId = %d, want %d", i, order.TankId, tank.Id)
		}
		if order.Heading != tank.Heading {
			t.Errorf("orders[%d].Heading = %d, want unchanged %d", i, order.Heading, tank.Heading)
		}
		if order.Speed != tank.Speed {
			t.Errorf("orders[%d].Speed = %d, want unchanged %d", i, order.Speed, tank.Speed)
		}
		if !order.TurretHold {
			t.Errorf("orders[%d].TurretHold = false, want true", i)
		}
		if order.Fire {
			t.Errorf("orders[%d].Fire = true, want false", i)
		}
	}
}

func TestHoldOrders_EmptyOwnTanksReturnsEmptyNonNilSlice(t *testing.T) {
	orders := HoldOrders(nil)

	if orders == nil {
		t.Fatal("HoldOrders(nil) returned a nil slice, want a non-nil empty slice (must marshal to [], not null)")
	}
	if len(orders) != 0 {
		t.Errorf("len(orders) = %d, want 0", len(orders))
	}
}

func TestHoldStrategy_DecideMatchesHoldOrders(t *testing.T) {
	view := mcpsdk.StateView{
		OwnTanks: []mcpsdk.TankView{{Id: 7, Heading: 4, Speed: 2}},
	}

	got := HoldStrategy{}.Decide(view)
	want := HoldOrders(view.OwnTanks)

	if len(got) != len(want) || got[0] != want[0] {
		t.Errorf("HoldStrategy{}.Decide(view) = %+v, want %+v", got, want)
	}
}

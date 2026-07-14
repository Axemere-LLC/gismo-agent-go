package agent

import (
	mcpsdk "github.com/Axemere-LLC/gismo-sdk-go/mcp"
)

// Strategy is the single hook a competitor implements: given the agent's
// current view of the battlefield for a match, decide what orders to
// submit for the current impulse. Everything else — the MCP server, the
// match-ID-scoped state cache, wire encoding/decoding — is handled by this
// package.
type Strategy interface {
	// Decide returns the orders to submit for view's impulse. It may
	// return an order for any subset of view.OwnTanks (or none); a tank
	// with no order simply holds its current heading/speed and does not
	// fire (../gismo-platform's referee applies this default, since agent
	// orders are untrusted input it validates rather than corrects).
	Decide(view mcpsdk.StateView) []mcpsdk.TankOrder
}

// HoldStrategy is the default, always-legal strategy: every own tank keeps
// its current heading and speed and holds its turret, firing at nothing.
// It is what the unmodified template plays, so a competitor who hasn't
// implemented their own Strategy yet still fields a legal (if inert) agent.
type HoldStrategy struct{}

// Decide implements Strategy.
func (HoldStrategy) Decide(view mcpsdk.StateView) []mcpsdk.TankOrder {
	return HoldOrders(view.OwnTanks)
}

// HoldOrders returns a legal "no-op" order for each tank in ownTanks: same
// heading, same speed (always a legal change per game.Speed.CanChangeTo —
// the diff is zero), turret held. Reference agents can use this as a
// starting point for tanks they choose not to act on this impulse.
func HoldOrders(ownTanks []mcpsdk.TankView) []mcpsdk.TankOrder {
	orders := make([]mcpsdk.TankOrder, len(ownTanks))
	for i, t := range ownTanks {
		orders[i] = mcpsdk.TankOrder{
			TankId:     t.Id,
			Speed:      t.Speed,
			Heading:    t.Heading,
			TurretHold: true,
		}
	}
	return orders
}

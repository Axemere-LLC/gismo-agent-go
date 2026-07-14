// Package heuristic implements a deterministic, no-randomness Gismo player:
// each own tank engages the nearest visible enemy (turning hull and turret
// toward it, firing once aligned and in range), or — with no enemy in
// sight — advances toward the nearest Forest cell for concealment, using
// the terrain map get_state now delivers in full every impulse. It is a
// demonstration of what a Strategy can do with the terrain and
// visible-tanks fields, not a tuned competitive player.
package heuristic

import (
	mcpsdk "github.com/Axemere-LLC/gismo-sdk-go/mcp"

	"github.com/Axemere-LLC/gismo-agent-go/agent"
)

const (
	speedHalted    = 1
	speedAheadHalf = 2

	forestTerrain = 1

	// effectiveRangeCells is the tank gun's documented effective range
	// (GISMO_Specification.md's Weapons section: "effective range of 100
	// grid squares"); beyond it a shot has no chance of hitting a target.
	effectiveRangeCells = 100

	// closeEngagementCells is a heuristic distance, not a documented rule:
	// inside it this strategy halts to steady its aim rather than
	// continuing to close, since a stopped tank turns its hull twice as
	// fast per impulse (agent.TurnAllowance).
	closeEngagementCells = 20
)

// Strategy is agent.Strategy's heuristic reference implementation.
type Strategy struct{}

// Decide implements agent.Strategy.
func (Strategy) Decide(view mcpsdk.StateView) []mcpsdk.TankOrder {
	orders := make([]mcpsdk.TankOrder, len(view.OwnTanks))
	for i, tank := range view.OwnTanks {
		orders[i] = orderFor(tank, view.VisibleTanks, view.Terrain)
	}
	return orders
}

func orderFor(tank mcpsdk.TankView, visible []mcpsdk.TankView, terrain []mcpsdk.TerrainView) mcpsdk.TankOrder {
	if enemy, ok := nearestTank(tank, visible); ok {
		return engage(tank, enemy)
	}
	if cover, ok := nearestForest(tank, terrain); ok {
		return seekCover(tank, cover)
	}
	return agent.HoldOrders([]mcpsdk.TankView{tank})[0]
}

// engage turns tank's hull and turret toward enemy, halting to steady its
// aim once close, and fires when the turret is already aligned, the target
// is in range, and ammo remains.
func engage(tank, enemy mcpsdk.TankView) mcpsdk.TankOrder {
	dx, dy := enemy.X-tank.X, enemy.Y-tank.Y
	targetHeading := agent.HeadingToward(dx, dy, tank.Heading)

	desiredSpeed := speedAheadHalf
	if distanceSquared(dx, dy) <= closeEngagementCells*closeEngagementCells {
		desiredSpeed = speedHalted
	}
	speed := agent.StepSpeedToward(tank.Speed, desiredSpeed)
	heading := agent.StepHeadingToward(tank.Heading, targetHeading, agent.TurnAllowance(speed))

	aligned := agent.TurnDistance(tank.TurretHeading, targetHeading) == 0
	inRange := distanceSquared(dx, dy) <= effectiveRangeCells*effectiveRangeCells

	return mcpsdk.TankOrder{
		TankId:        tank.Id,
		Speed:         speed,
		Heading:       heading,
		TurretHold:    false,
		TurretHeading: targetHeading,
		Fire:          aligned && inRange && tank.Ammo > 0,
		TargetX:       enemy.X,
		TargetY:       enemy.Y,
	}
}

// seekCover advances tank toward cover, turret held since there is nothing
// to aim at.
func seekCover(tank mcpsdk.TankView, cover mcpsdk.TerrainView) mcpsdk.TankOrder {
	dx, dy := cover.X-tank.X, cover.Y-tank.Y
	if dx == 0 && dy == 0 {
		return agent.HoldOrders([]mcpsdk.TankView{tank})[0]
	}

	targetHeading := agent.HeadingToward(dx, dy, tank.Heading)
	speed := agent.StepSpeedToward(tank.Speed, speedAheadHalf)
	heading := agent.StepHeadingToward(tank.Heading, targetHeading, agent.TurnAllowance(speed))

	return mcpsdk.TankOrder{
		TankId:     tank.Id,
		Speed:      speed,
		Heading:    heading,
		TurretHold: true,
	}
}

// nearestTank returns the candidate closest to from (squared Euclidean
// distance, ties broken by lowest Id for determinism).
func nearestTank(from mcpsdk.TankView, candidates []mcpsdk.TankView) (mcpsdk.TankView, bool) {
	var best mcpsdk.TankView
	bestDist := 0
	found := false
	for _, c := range candidates {
		dist := distanceSquared(c.X-from.X, c.Y-from.Y)
		if !found || dist < bestDist || (dist == bestDist && c.Id < best.Id) {
			best, bestDist, found = c, dist, true
		}
	}
	return best, found
}

// nearestForest returns the closest Forest cell to from (squared Euclidean
// distance, ties broken by lowest Y then lowest X for determinism).
func nearestForest(from mcpsdk.TankView, terrain []mcpsdk.TerrainView) (mcpsdk.TerrainView, bool) {
	var best mcpsdk.TerrainView
	bestDist := 0
	found := false
	for _, cell := range terrain {
		if cell.Type != forestTerrain {
			continue
		}
		dist := distanceSquared(cell.X-from.X, cell.Y-from.Y)
		if !found || dist < bestDist || (dist == bestDist && (cell.Y < best.Y || (cell.Y == best.Y && cell.X < best.X))) {
			best, bestDist, found = cell, dist, true
		}
	}
	return best, found
}

func distanceSquared(dx, dy int) int {
	return dx*dx + dy*dy
}

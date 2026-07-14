// Package random implements the simplest legal Gismo player: every impulse,
// each own tank picks a random legal heading/speed step and, if an enemy is
// visible, sometimes fires at one. It exists to give competitors (and the
// conformance harness) a deterministic, always-legal opponent that isn't
// just holding still — not to play well.
package random

import (
	"math/rand/v2"
	"sync"

	mcpsdk "github.com/Axemere-LLC/gismo-sdk-go/mcp"

	"github.com/Axemere-LLC/gismo-agent-go/agent"
)

// numHeadings and numSpeeds are the sizes of the wire's compass and speed
// scales (see agent.TurnDistance's doc comment for the encoding).
const (
	numHeadings = 8
	numSpeeds   = 4

	// fireProbabilityPercent is the chance, per own tank per impulse, that
	// this strategy fires at a random visible enemy rather than just
	// maneuvering. It's a difficulty knob, not a rule from the spec.
	fireProbabilityPercent = 50
)

// Strategy is agent.Strategy's random reference implementation.
type Strategy struct {
	mu  sync.Mutex
	rng *rand.Rand
}

// New returns a random Strategy seeded deterministically from seed: the
// same seed always produces the same sequence of orders, which is what
// keeps this agent reproducible for the conformance harness and for CI.
func New(seed uint64) *Strategy {
	return &Strategy{rng: rand.New(rand.NewPCG(seed, seed))}
}

// Decide implements agent.Strategy.
func (s *Strategy) Decide(view mcpsdk.StateView) []mcpsdk.TankOrder {
	s.mu.Lock()
	defer s.mu.Unlock()

	orders := make([]mcpsdk.TankOrder, len(view.OwnTanks))
	for i, tank := range view.OwnTanks {
		orders[i] = s.orderFor(tank, view.VisibleTanks)
	}
	return orders
}

// orderFor picks a random legal heading/speed step for tank, and — with
// probability fireProbabilityPercent — turns the turret toward a random
// visible enemy and fires at it.
func (s *Strategy) orderFor(tank mcpsdk.TankView, visible []mcpsdk.TankView) mcpsdk.TankOrder {
	targetSpeed := s.rng.IntN(numSpeeds)
	speed := agent.StepSpeedToward(tank.Speed, targetSpeed)

	targetHeading := s.rng.IntN(numHeadings)
	heading := agent.StepHeadingToward(tank.Heading, targetHeading, agent.TurnAllowance(speed))

	order := mcpsdk.TankOrder{
		TankId:     tank.Id,
		Speed:      speed,
		Heading:    heading,
		TurretHold: true,
	}

	if len(visible) == 0 || s.rng.IntN(100) >= fireProbabilityPercent {
		return order
	}

	target := visible[s.rng.IntN(len(visible))]
	order.TurretHold = false
	order.TurretHeading = agent.HeadingToward(target.X-tank.X, target.Y-tank.Y, tank.TurretHeading)
	order.Fire = true
	order.TargetX = target.X
	order.TargetY = target.Y
	return order
}

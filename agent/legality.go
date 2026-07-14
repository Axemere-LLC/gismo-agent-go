package agent

import "math"

// Package-level helpers for building orders that respect the referee's
// per-impulse turn-rate and speed-change limits
// (game-and-protocol.md#match-protocol-mcp-tools). They operate purely on
// the wire's integer encodings — heading is the 8-point compass (0=North,
// clockwise, ..., 7=Northwest) and speed is the 4-step scale (0=BackHalf,
// 1=Halted, 2=AheadHalf, 3=AheadFull) — so any Strategy can use them without
// depending on gismo-platform's internal game package. An order that turns
// or accelerates faster than these functions allow is not corrected by the
// referee: it is rejected outright and the tank holds instead, so getting
// this arithmetic right is what keeps a Strategy's orders legal.
const numHeadings = 8

// TurnDistance returns the minimum number of 45-degree compass steps
// between two headings (each in [0,7]), in [0,4].
func TurnDistance(a, b int) int {
	d := ((a-b)%numHeadings + numHeadings) % numHeadings
	if d > numHeadings/2 {
		d = numHeadings - d
	}
	return d
}

// TurnAllowance returns how many compass steps a tank may turn its hull in
// one impulse, given the speed it is ordered to hold for that impulse: 2 if
// Halted (1), 1 otherwise. This mirrors the referee's own turn-rate rule,
// which keys off the ordered speed, not the tank's speed before the order.
func TurnAllowance(orderedSpeed int) int {
	const halted = 1
	if orderedSpeed == halted {
		return 2
	}
	return 1
}

// StepHeadingToward returns the heading reached by turning current at most
// allowance compass steps toward target, choosing whichever rotation
// direction is shorter (a tie — target directly opposite current — turns
// clockwise). Returns current unchanged if already at target or allowance
// is 0.
func StepHeadingToward(current, target, allowance int) int {
	if allowance <= 0 {
		return current
	}
	diff := ((target-current)%numHeadings + numHeadings) % numHeadings
	if diff > numHeadings/2 {
		diff -= numHeadings // now in (-4, 4]; negative means counter-clockwise is shorter
	}
	step := diff
	if step > allowance {
		step = allowance
	} else if step < -allowance {
		step = -allowance
	}
	return ((current+step)%numHeadings + numHeadings) % numHeadings
}

// StepSpeedToward returns the speed reached by changing current by at most
// one step toward target (each in [0,3]); the referee only permits a diff
// of -1, 0, or 1 per impulse.
func StepSpeedToward(current, target int) int {
	switch {
	case target > current:
		return current + 1
	case target < current:
		return current - 1
	default:
		return current
	}
}

// HeadingToward returns the 8-point compass heading that best points from
// (0,0) toward (dx,dy), rounding to the nearest 45-degree sector (Y
// increases southward, matching the grid's row-major convention). If dx and
// dy are both zero, it returns current unchanged since there is no
// direction to point toward.
func HeadingToward(dx, dy, current int) int {
	if dx == 0 && dy == 0 {
		return current
	}
	bearing := math.Atan2(float64(dx), float64(-dy)) * 180 / math.Pi
	if bearing < 0 {
		bearing += 360
	}
	return int(math.Round(bearing/45)) % numHeadings
}

package agent

import "testing"

func TestTurnDistance(t *testing.T) {
	tests := []struct {
		name string
		a, b int
		want int
	}{
		{"same heading", 0, 0, 0},
		{"one step clockwise", 0, 1, 1},
		{"one step counter-clockwise", 1, 0, 1},
		{"opposite headings", 0, 4, 4},
		{"wraps past north", 7, 1, 2},
		{"wraps the other way", 1, 7, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TurnDistance(tt.a, tt.b); got != tt.want {
				t.Errorf("TurnDistance(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestTurnAllowance(t *testing.T) {
	tests := []struct {
		name         string
		orderedSpeed int
		want         int
	}{
		{"back half", 0, 1},
		{"halted gets a wider turn", 1, 2},
		{"ahead half", 2, 1},
		{"ahead full", 3, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TurnAllowance(tt.orderedSpeed); got != tt.want {
				t.Errorf("TurnAllowance(%d) = %d, want %d", tt.orderedSpeed, got, tt.want)
			}
		})
	}
}

func TestStepHeadingToward(t *testing.T) {
	tests := []struct {
		name                    string
		current, target, budget int
		want                    int
	}{
		{"already at target", 3, 3, 1, 3},
		{"zero allowance holds", 0, 4, 0, 0},
		{"reaches target within budget", 0, 1, 1, 1},
		{"clamped short of target, clockwise", 0, 3, 2, 2},
		{"clamped short of target, counter-clockwise", 0, 5, 1, 7},
		{"opposite target ties clockwise", 0, 4, 1, 1},
		{"wraps north going clockwise", 7, 1, 2, 1},
		{"halted allowance covers a 2-step turn", 0, 2, 2, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StepHeadingToward(tt.current, tt.target, tt.budget)
			if got != tt.want {
				t.Errorf("StepHeadingToward(%d, %d, %d) = %d, want %d", tt.current, tt.target, tt.budget, got, tt.want)
			}
			// Whatever this returns must itself have been a legal turn.
			if d := TurnDistance(tt.current, got); d > tt.budget {
				t.Errorf("StepHeadingToward(%d, %d, %d) = %d turns %d steps, exceeds budget %d", tt.current, tt.target, tt.budget, got, d, tt.budget)
			}
		})
	}
}

func TestStepSpeedToward(t *testing.T) {
	tests := []struct {
		name            string
		current, target int
		want            int
	}{
		{"already at target", 2, 2, 2},
		{"accelerate one step", 1, 3, 2},
		{"decelerate one step", 3, 0, 2},
		{"large jump clamped to one step up", 0, 3, 1},
		{"large jump clamped to one step down", 3, 0, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StepSpeedToward(tt.current, tt.target); got != tt.want {
				t.Errorf("StepSpeedToward(%d, %d) = %d, want %d", tt.current, tt.target, got, tt.want)
			}
		})
	}
}

func TestHeadingToward(t *testing.T) {
	tests := []struct {
		name    string
		dx, dy  int
		current int
		want    int
	}{
		{"no delta returns current unchanged", 0, 0, 5, 5},
		{"due north", 0, -1, 0, 0},
		{"due northeast", 1, -1, 0, 1},
		{"due east", 1, 0, 0, 2},
		{"due southeast", 1, 1, 0, 3},
		{"due south", 0, 1, 0, 4},
		{"due southwest", -1, 1, 0, 5},
		{"due west", -1, 0, 0, 6},
		{"due northwest", -1, -1, 0, 7},
		{"scaled delta resolves the same as its unit vector", 5, -5, 0, 1},
		{"mostly-north nudge rounds to north", 1, -8, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HeadingToward(tt.dx, tt.dy, tt.current); got != tt.want {
				t.Errorf("HeadingToward(%d, %d, %d) = %d, want %d", tt.dx, tt.dy, tt.current, got, tt.want)
			}
		})
	}
}

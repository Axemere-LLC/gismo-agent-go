package agent_test

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"testing"

	"github.com/Axemere-LLC/gismo-agent-go/agent"
	"github.com/Axemere-LLC/gismo-agent-go/examples/heuristic"
	"github.com/Axemere-LLC/gismo-agent-go/examples/random"
)

// update regenerates the golden files in fixtures/expected instead of
// comparing against them. Run: go test ./agent/... -run TestFixtures -update
var update = flag.Bool("update", false, "update golden fixture files instead of comparing against them")

// TestFixtures locks each of this template's mounted generations to a
// recorded set of orders for a fixed scenario corpus, so an edit to a
// shared helper (agent.StepHeadingToward, agent.HeadingToward, ...) that
// silently changes an already-shipped generation's behavior fails a test
// instead of shipping unnoticed.
func TestFixtures(t *testing.T) {
	scenarios, err := agent.LoadScenarios("../fixtures/scenarios.json")
	if err != nil {
		t.Fatalf("LoadScenarios: %v", err)
	}

	cases := []struct {
		name     string
		strategy agent.Strategy
		golden   string
	}{
		{"v1", agent.HoldStrategy{}, "../fixtures/expected/v1.json"},
		{"random-v1", random.New(1), "../fixtures/expected/random-v1.json"},
		{"heuristic-v1", heuristic.Strategy{}, "../fixtures/expected/heuristic-v1.json"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			replies := agent.Replay(tc.strategy, scenarios)

			got, err := json.MarshalIndent(replies, "", "  ")
			if err != nil {
				t.Fatalf("marshal replies: %v", err)
			}
			got = append(got, '\n')

			if *update {
				if err := os.WriteFile(tc.golden, got, 0o644); err != nil {
					t.Fatalf("write golden %s: %v", tc.golden, err)
				}
				return
			}

			want, err := os.ReadFile(tc.golden)
			if err != nil {
				t.Fatalf("read golden %s: %v (run with -update to generate it)", tc.golden, err)
			}
			if !bytes.Equal(got, want) {
				t.Errorf("%s replay drifted from %s\ngot:\n%s\nwant:\n%s\n(run with -update if this drift is intentional)",
					tc.name, tc.golden, got, want)
			}
		})
	}
}

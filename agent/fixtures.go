package agent

import (
	"encoding/json"
	"fmt"
	"os"

	mcpsdk "github.com/Axemere-LLC/gismo-sdk-go/mcp"
)

// Scenario is one StateView input in a fixture corpus, paired with a short
// name identifying it in test output and golden files.
type Scenario struct {
	Name string           `json:"name"`
	View mcpsdk.StateView `json:"view"`
}

// Reply is one Scenario's recorded output: the orders a Strategy returned
// for it, carrying the same Name as its Scenario so a golden file reads
// standalone.
type Reply struct {
	Name   string             `json:"name"`
	Orders []mcpsdk.TankOrder `json:"orders"`
}

// LoadScenarios reads a JSON-encoded fixture corpus (a []Scenario) from
// path.
func LoadScenarios(path string) ([]Scenario, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("agent: load scenarios: %w", err)
	}
	var scenarios []Scenario
	if err := json.Unmarshal(data, &scenarios); err != nil {
		return nil, fmt.Errorf("agent: decode scenarios %s: %w", path, err)
	}
	return scenarios, nil
}

// Replay runs strategy.Decide against every scenario in order and returns
// one Reply per scenario. For a stateful Strategy (e.g. a seeded PRNG),
// scenario order therefore affects the result — replay a fresh Strategy
// instance against the same scenario order every time to get a
// reproducible golden comparison.
func Replay(strategy Strategy, scenarios []Scenario) []Reply {
	replies := make([]Reply, len(scenarios))
	for i, s := range scenarios {
		replies[i] = Reply{Name: s.Name, Orders: strategy.Decide(s.View)}
	}
	return replies
}

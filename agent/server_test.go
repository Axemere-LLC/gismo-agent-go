package agent

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	mcpsdk "github.com/Axemere-LLC/gismo-sdk-go/mcp"
)

// connect starts server in-process (no real network involved, mirroring
// gismo-platform's internal/referee mock-agent test pattern) and returns a
// connected client session.
func connect(t *testing.T, ctx context.Context, server *mcp.Server) *mcp.ClientSession {
	t.Helper()

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	if _, err := server.Connect(ctx, serverTransport, nil); err != nil {
		t.Fatalf("connect server: %v", err)
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "gismo-agent-go-test", Version: "test"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("connect client: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })

	return session
}

func callTool[Out any](t *testing.T, ctx context.Context, session *mcp.ClientSession, name string, args any) Out {
	t.Helper()

	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("call %s: %v", name, err)
	}
	if result.IsError {
		t.Fatalf("call %s returned an error result: %+v", name, result.Content)
	}

	// StructuredContent round-trips through the wire as a generic
	// map[string]any (it's decoded into an `any` field); re-marshal and
	// decode into the concrete Out type to compare against expectations.
	raw, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatalf("marshal %s structured content: %v", name, err)
	}
	var out Out
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("decode %s result: %v", name, err)
	}
	return out
}

func TestNewServer_GetStateCachesAndEchoesView(t *testing.T) {
	ctx := context.Background()
	session := connect(t, ctx, NewServer(HoldStrategy{}))

	in := mcpsdk.StateView{
		MatchId:  "m1",
		Impulse:  3,
		OwnTanks: []mcpsdk.TankView{{Id: 1, Heading: 2, Speed: 1}},
	}

	got := callTool[mcpsdk.StateView](t, ctx, session, "get_state", in)
	if !reflect.DeepEqual(got, in) {
		t.Errorf("get_state echoed %+v, want %+v", got, in)
	}
}

func TestNewServer_SubmitOrdersUsesCachedView(t *testing.T) {
	ctx := context.Background()
	session := connect(t, ctx, NewServer(HoldStrategy{}))

	view := mcpsdk.StateView{
		MatchId:  "m1",
		Impulse:  3,
		OwnTanks: []mcpsdk.TankView{{Id: 1, Heading: 2, Speed: 1}},
	}
	callTool[mcpsdk.StateView](t, ctx, session, "get_state", view)

	got := callTool[mcpsdk.SubmitOrdersResponse](t, ctx, session, "submit_orders", mcpsdk.SubmitOrdersRequest{MatchId: "m1", Impulse: 3})

	want := HoldOrders(view.OwnTanks)
	if got.Impulse != 3 || len(got.Orders) != len(want) || got.Orders[0] != want[0] {
		t.Errorf("submit_orders = %+v, want Impulse=3 Orders=%+v", got, want)
	}
}

func TestNewServer_SubmitOrdersWithNoCachedViewReturnsEmptyOrders(t *testing.T) {
	ctx := context.Background()
	session := connect(t, ctx, NewServer(HoldStrategy{}))

	got := callTool[mcpsdk.SubmitOrdersResponse](t, ctx, session, "submit_orders", mcpsdk.SubmitOrdersRequest{MatchId: "never-seen", Impulse: 1})

	if got.Impulse != 1 {
		t.Errorf("submit_orders.Impulse = %d, want 1", got.Impulse)
	}
	if len(got.Orders) != 0 {
		t.Errorf("submit_orders.Orders = %+v, want empty", got.Orders)
	}
}

func TestNewServer_SubmitOrdersIsScopedByMatchID(t *testing.T) {
	ctx := context.Background()
	session := connect(t, ctx, NewServer(HoldStrategy{}))

	callTool[mcpsdk.StateView](t, ctx, session, "get_state", mcpsdk.StateView{
		MatchId:  "match-a",
		OwnTanks: []mcpsdk.TankView{{Id: 1, Heading: 0, Speed: 1}},
	})

	// submit_orders for a different, never-seen match must not see
	// match-a's cached view.
	got := callTool[mcpsdk.SubmitOrdersResponse](t, ctx, session, "submit_orders", mcpsdk.SubmitOrdersRequest{MatchId: "match-b", Impulse: 1})

	if len(got.Orders) != 0 {
		t.Errorf("submit_orders for an unrelated match = %+v, want empty (no cross-match leakage)", got.Orders)
	}
}

// TestNewServer_SelfPlaySessionsDoNotShareCachedView is the end-to-end
// regression test for the self-play collision bug: when the same agent
// process serves both sides of a match (identical matchID, two distinct
// MCP sessions — one per side, exactly as the referee's two Dial calls
// produce), each session's submit_orders must be decided from its own
// get_state view, never the other session's.
func TestNewServer_SelfPlaySessionsDoNotShareCachedView(t *testing.T) {
	ctx := context.Background()
	server := NewServer(HoldStrategy{})

	sideA := connect(t, ctx, server)
	sideB := connect(t, ctx, server)

	viewA := mcpsdk.StateView{MatchId: "m1", Impulse: 1, OwnTanks: []mcpsdk.TankView{{Id: 1, Heading: 0, Speed: 1}}}
	viewB := mcpsdk.StateView{MatchId: "m1", Impulse: 1, OwnTanks: []mcpsdk.TankView{{Id: 8, Heading: 0, Speed: 1}}}
	callTool[mcpsdk.StateView](t, ctx, sideA, "get_state", viewA)
	callTool[mcpsdk.StateView](t, ctx, sideB, "get_state", viewB)

	gotA := callTool[mcpsdk.SubmitOrdersResponse](t, ctx, sideA, "submit_orders", mcpsdk.SubmitOrdersRequest{MatchId: "m1", Impulse: 1})
	gotB := callTool[mcpsdk.SubmitOrdersResponse](t, ctx, sideB, "submit_orders", mcpsdk.SubmitOrdersRequest{MatchId: "m1", Impulse: 1})

	wantA := HoldOrders(viewA.OwnTanks)
	wantB := HoldOrders(viewB.OwnTanks)
	if len(gotA.Orders) != 1 || gotA.Orders[0].TankId != wantA[0].TankId {
		t.Errorf("side A submit_orders = %+v, want orders for tank %d (its own), got tank %d", gotA.Orders, wantA[0].TankId, gotA.Orders[0].TankId)
	}
	if len(gotB.Orders) != 1 || gotB.Orders[0].TankId != wantB[0].TankId {
		t.Errorf("side B submit_orders = %+v, want orders for tank %d (its own), got tank %d", gotB.Orders, wantB[0].TankId, gotB.Orders[0].TankId)
	}
}

func TestNewServer_SurrenderReportsFalse(t *testing.T) {
	ctx := context.Background()
	session := connect(t, ctx, NewServer(HoldStrategy{}))

	got := callTool[mcpsdk.SurrenderResponse](t, ctx, session, "surrender", mcpsdk.SurrenderRequest{MatchId: "m1"})

	if got.Surrendered {
		t.Error("surrender = true, want false (the template never surrenders on its own)")
	}
}

// TestNewServer_ReportedVersion covers the serverInfo.version an agent
// reports during the initialize handshake: the template default unless the
// competitor overrides it with the version_label the platform assigned their
// registered agent.
func TestNewServer_ReportedVersion(t *testing.T) {
	tests := []struct {
		name string
		opts []Option
		want string
	}{
		{name: "no options reports the template version", want: Version},
		{name: "WithVersion overrides it", opts: []Option{WithVersion("v2")}, want: "v2"},
		{name: "empty WithVersion keeps the default", opts: []Option{WithVersion("")}, want: Version},
		{name: "last WithVersion wins", opts: []Option{WithVersion("v2"), WithVersion("v3")}, want: "v3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			session := connect(t, ctx, NewServer(HoldStrategy{}, tt.opts...))

			info := session.InitializeResult().ServerInfo
			if info.Version != tt.want {
				t.Errorf("serverInfo.version = %q, want %q", info.Version, tt.want)
			}
			if info.Name != Name {
				t.Errorf("serverInfo.name = %q, want %q (the version knob must not affect the name)", info.Name, Name)
			}
		})
	}
}

func TestNewServer_NilStrategyDefaultsToHold(t *testing.T) {
	ctx := context.Background()
	session := connect(t, ctx, NewServer(nil))

	view := mcpsdk.StateView{
		MatchId:  "m1",
		OwnTanks: []mcpsdk.TankView{{Id: 9, Heading: 6, Speed: 2}},
	}
	callTool[mcpsdk.StateView](t, ctx, session, "get_state", view)

	got := callTool[mcpsdk.SubmitOrdersResponse](t, ctx, session, "submit_orders", mcpsdk.SubmitOrdersRequest{MatchId: "m1", Impulse: 1})

	want := HoldOrders(view.OwnTanks)
	if len(got.Orders) != len(want) || got.Orders[0] != want[0] {
		t.Errorf("NewServer(nil) submit_orders = %+v, want the HoldStrategy default %+v", got.Orders, want)
	}
}

package agent

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	mcpsdk "github.com/Axemere-LLC/gismo-sdk-go/mcp"
)

// connectHTTP connects to a live HTTP endpoint, unlike connect (in
// server_test.go), which only exercises the in-memory transport. Routing
// tests need a real request to go through net/http's mux to be meaningful.
//
// Unlike connect, this does not register the session's Close via
// t.Cleanup: the streamable HTTP transport holds a long-lived GET
// connection open for server-to-client pushes, and httptest.Server.Close
// blocks until every connection it served is idle. t.Cleanup callbacks run
// only after the test function (and its own defers, including a
// `defer server.Close()`) has already returned, so relying on it here would
// have server.Close() wait forever on a connection the session hasn't
// closed yet. Callers must close the session themselves — with a `defer`
// ordered before the server's — while the test is still running.
func connectHTTP(t *testing.T, ctx context.Context, endpoint string) *mcp.ClientSession {
	t.Helper()

	client := mcp.NewClient(&mcp.Implementation{Name: "gismo-agent-go-test", Version: "test"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: endpoint}, nil)
	if err != nil {
		t.Fatalf("connect %s: %v", endpoint, err)
	}
	return session
}

func TestVersionedHandler_RejectsInvalidMountLists(t *testing.T) {
	tests := []struct {
		name   string
		mounts []Mount
	}{
		{"no mounts", nil},
		{"path missing leading slash", []Mount{{Path: "v1", Strategy: HoldStrategy{}}}},
		{"bare root path", []Mount{{Path: "/", Strategy: HoldStrategy{}}}},
		{"duplicate path", []Mount{
			{Path: "/v1", Strategy: HoldStrategy{}},
			{Path: "/v1", Strategy: HoldStrategy{}},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := VersionedHandler(tt.mounts...); err == nil {
				t.Errorf("VersionedHandler(%+v) = nil error, want an error", tt.mounts)
			}
		})
	}
}

func TestVersionedHandler_UnknownPathIs404(t *testing.T) {
	handler, err := VersionedHandler(Mount{Path: "/v1", Strategy: HoldStrategy{}})
	if err != nil {
		t.Fatalf("VersionedHandler: %v", err)
	}
	server := httptest.NewServer(handler)
	defer server.Close()

	resp, err := http.Post(server.URL+"/v3", "application/json", strings.NewReader("{}"))
	if err != nil {
		t.Fatalf("POST /v3: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("POST /v3 = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestVersionedHandler_ExactAndTrailingSlashPathsServeWithoutRedirect(t *testing.T) {
	handler, err := VersionedHandler(Mount{Path: "/v1", Strategy: HoldStrategy{}})
	if err != nil {
		t.Fatalf("VersionedHandler: %v", err)
	}
	server := httptest.NewServer(handler)
	defer server.Close()

	// Registering only the "/v1/" subtree pattern would make ServeMux
	// 301-redirect a bare "/v1" request; an MCP client shouldn't have to
	// follow a redirect to reach the endpoint it was given. Disable
	// automatic redirect-following so a 3xx here fails the test loudly
	// rather than resolving into whatever the redirect target answers.
	client := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	for _, path := range []string{"/v1", "/v1/"} {
		resp, err := client.Post(server.URL+path, "application/json", strings.NewReader("{}"))
		if err != nil {
			t.Fatalf("POST %s: %v", path, err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode >= 300 && resp.StatusCode < 400 {
			t.Errorf("POST %s = %d, want no redirect", path, resp.StatusCode)
		}
	}
}

// TestVersionedHandler_ReportsVersionLabelMatchingPath guards against a
// mount reporting a fixed default version (e.g. agent.Version) in the MCP
// initialize handshake regardless of which path a client connected to — the
// referee compares this against the agent_versions.version_label it
// expects, so a mismatch here would silently misattribute match results.
func TestVersionedHandler_ReportsVersionLabelMatchingPath(t *testing.T) {
	handler, err := VersionedHandler(
		Mount{Path: "/v1", Strategy: HoldStrategy{}},
		Mount{Path: "/v3", Strategy: HoldStrategy{}},
	)
	if err != nil {
		t.Fatalf("VersionedHandler: %v", err)
	}
	server := httptest.NewServer(handler)
	defer server.Close()

	ctx := context.Background()
	for _, path := range []string{"/v1", "/v3"} {
		session := connectHTTP(t, ctx, server.URL+path)
		got := session.InitializeResult().ServerInfo.Version
		_ = session.Close()

		want := strings.TrimPrefix(path, "/")
		if got != want {
			t.Errorf("path %s: reported version = %q, want %q", path, got, want)
		}
	}
}

// TestVersionedHandler_MountsHaveIndependentState guards against two mounts
// sharing a state cache: if they did, a get_state seen by one mount would
// leak into another mount's submit_orders for the same match ID.
func TestVersionedHandler_MountsHaveIndependentState(t *testing.T) {
	handler, err := VersionedHandler(
		Mount{Path: "/v1", Strategy: HoldStrategy{}},
		Mount{Path: "/v2", Strategy: HoldStrategy{}},
	)
	if err != nil {
		t.Fatalf("VersionedHandler: %v", err)
	}
	server := httptest.NewServer(handler)
	defer server.Close()

	ctx := context.Background()
	const matchID = "shared-match"

	v1 := connectHTTP(t, ctx, server.URL+"/v1")
	defer func() { _ = v1.Close() }()
	callTool[mcpsdk.StateView](t, ctx, v1, "get_state", mcpsdk.StateView{
		MatchId:  matchID,
		Impulse:  1,
		OwnTanks: []mcpsdk.TankView{{Id: 1, Heading: 0, Speed: 1}},
	})

	// Same match ID, but the /v2 mount never saw a get_state for it: if
	// mounts shared a cache, this would return /v1's cached orders instead
	// of an empty list.
	v2 := connectHTTP(t, ctx, server.URL+"/v2")
	defer func() { _ = v2.Close() }()
	got := callTool[mcpsdk.SubmitOrdersResponse](t, ctx, v2, "submit_orders", mcpsdk.SubmitOrdersRequest{MatchId: matchID, Impulse: 1})

	if len(got.Orders) != 0 {
		t.Errorf("/v2 submit_orders for a match only /v1 saw = %+v, want empty (mounts must not share state)", got.Orders)
	}
}

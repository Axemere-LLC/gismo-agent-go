# gismo-agent-go

**Go starter template for a Gismo competitor agent — clone it, implement one interface, and you have
a legal, playable MCP server.**

![version](https://img.shields.io/badge/go-v1.0.0-blue)
![license](https://img.shields.io/badge/license-Apache--2.0-blue)
![CI](https://github.com/Axemere-LLC/gismo-agent-go/actions/workflows/ci.yml/badge.svg)

## What is Gismo 2026?

Gismo 2026 is a cloud platform where AI agents compete head-to-head in GISMO, a tank-battle game
originally defined in 1991. Organizations register agents instead of humans; the platform pairs
agents against each other over the Model Context Protocol (MCP), adjudicates every move through a
referee, rates the results, and makes every match replayable afterward.

This repo is an MCP server that talks directly to the referee (`get_state` / `submit_orders` /
`surrender`), with exactly one function left as a stub for you to fill in. It also hosts two
runnable reference agents under `examples/`.

## Table of Contents

- [Install](#install)
- [Quickstart](#quickstart)
- [Auth](#auth)
- [The `Strategy` interface](#the-strategy-interface)
- [Serving multiple versions](#serving-multiple-versions)
- [Observability model](#observability-model)
- [Wire encodings](#wire-encodings)
- [Reference agents](#reference-agents)
- [Versioning & compatibility](#versioning--compatibility)
- [Reporting your agent's version](#reporting-your-agents-version)
- [Deploy it](#deploy-it)
- [Related repos](#related-repos)
- [Testing](#testing)
- [Repository layout](#repository-layout)
- [License](#license)

## Install

There is no `create-gismo-agent` scaffolder yet — use this repo directly as a GitHub template, or
fork it:

```sh
git clone https://github.com/Axemere-LLC/gismo-agent-go.git my-agent
cd my-agent && go build ./...
```

## Quickstart

```sh
go run . -addr :8080
```

`-addr` is the address the agent's MCP endpoint listens on. The template mounts its `Strategy` at
`/v1` (see [Serving multiple versions](#serving-multiple-versions)) — point the referee (or the
conformance harness) at `http://<host>:8080/v1` for this match. The endpoint speaks the MCP
Streamable HTTP transport in plaintext — terminate TLS in front of this process (a load balancer or
reverse proxy) rather than inside it, per `game-and-protocol.md`'s Secure Transport Requirements.

Run `go run . -h` for the full flag list.

## Auth

This agent's MCP endpoint is a server, not a caller — it doesn't itself hold a Personal API Token or
JWT. It's the *referee* that authenticates to your endpoint when a match starts (via a match-scoped
credential passed at agent registration), and your endpoint that authenticates to the platform's REST
API — for registering agent versions, checking match history, and similar — using a PAT or JWT
exactly as described in [`gismo-sdk-go`](https://github.com/Axemere-LLC/gismo-sdk-go#auth), which
this template imports.

## The `Strategy` interface

```go
type Strategy interface {
    // Decide returns the orders to submit for view's impulse. It may
    // return an order for any subset of view.OwnTanks (or none); a tank
    // with no order simply holds its current heading/speed and does not
    // fire.
    Decide(view mcpsdk.StateView) []mcpsdk.TankOrder
}
```

This is the only method you implement. Everything else — the MCP tool surface, the match-ID-scoped
state cache, wire encoding/decoding — is handled by the `agent` package. `main.go` wires
`agent.HoldStrategy{}` (hold heading/speed, never fire) into a `/v1` mount; replace that one
`Strategy` value with your own and your agent is playable:

```go
handler, err := agent.VersionedHandler(agent.Mount{Path: "/v1", Strategy: yourpkg.Strategy{}})
if err != nil {
    log.Fatalf("versioned handler: %v", err)
}
if err := agent.ServeHandler(ctx, *addr, handler); err != nil {
    log.Fatalf("serve: %v", err)
}
```

## Serving multiple versions

Gismo has two independent versioning axes: this repo's **code version** (semver, `go.mod`, git tags)
and your agent's **generation** (a flat integer, one immutable URL path `/vN`, rated independently by
the platform from the moment it's registered). A code release doesn't create a new generation — only
adding another `agent.Mount` does.

`agent.VersionedHandler` mounts one or more immutable generations, each with its own `Strategy` and
its own isolated match-state cache, in a single process:

```go
handler, err := agent.VersionedHandler(
    agent.Mount{Path: "/v1", Strategy: v1.Strategy{}}, // frozen: never change what /v1 serves
    agent.Mount{Path: "/v2", Strategy: v2.Strategy{}}, // your current, still-evolving generation
)
```

Register `/v1` and `/v2` as separate agent versions with the platform, each with its own
`version_label` (`"v1"`, `"v2"`); the referee compares that label against `serverInfo.version` from
each mount's MCP `initialize` handshake, which `VersionedHandler` derives automatically from the
mount's path (`"/v3"` reports `"v3"`) — there's no separate version string to keep in sync by hand.
Once a generation is rated, treat its `Strategy` as frozen: fix a bug or improve behavior by adding a
new `Mount` at a new path, not by editing the old one in place — see
[Fixture drift lock](#testing) for a test that catches an accidental edit to a shared helper (like
`agent/legality.go`) silently changing an already-shipped generation's behavior.

`agent.VersionedHandler` returns a bare `http.Handler` with no auth applied — wrap it yourself, as
`main.go` does with `agent.BearerAuth`, before passing it to `agent.ServeHandler`.

## Observability model

Every `get_state` call returns a `StateView` — your agent's complete view of the battlefield for one
match, for one impulse:

| Data | What you get |
|---|---|
| Terrain | The **complete** static map, identical for both sides, every impulse. Never gated. |
| Own tanks | Always, in full. |
| Enemy tanks | Only the ones currently Line-of-Sight-visible to one of your tanks. |
| Blockhouses | Yours always; the enemy's only when Line-of-Sight-visible. |

This mirrors the original 1991 design: terrain was delivered once at match start (it never changes), and
everything else is fog-of-war'd except your own units. In the 2026 protocol the terrain array just rides
along on every `get_state` response instead of a separate startup step — repeating it is harmless, since
it's static.

```mermaid
sequenceDiagram
    participant Referee
    participant Agent as Your agent (this repo)

    Referee->>Agent: get_state(matchId, impulse)
    Agent-->>Referee: StateView (terrain, own tanks,<br>visible enemies, blockhouses)
    Note over Agent: cache StateView by matchId
    Referee->>Agent: submit_orders(matchId, impulse)
    Agent-->>Referee: orders, decided from the<br>cached StateView
```

**Figure 1.** One impulse of the match loop, from the agent's side. Alt text: a sequence diagram showing
the referee calling `get_state`, the agent caching the returned view, then the referee calling
`submit_orders` and the agent replying with orders decided from that cached view.

`submit_orders` requests carry only a match ID and impulse number — no state. That's why every agent
built on this package caches the most recent `StateView` per match ID (see `agent/cache.go`) and decides
orders from that cache; a `submit_orders` call that arrives before any `get_state` for its match falls
back to an empty, always-legal order list (every un-ordered tank simply holds).

## Wire encodings

All enums on the wire are plain integers, matching `gismo-sdk-go/mcp`'s generated types:

| Field | Encoding |
|---|---|
| `Heading` / `TurretHeading` | 8-point compass, clockwise from North: `0=N, 1=NE, 2=E, 3=SE, 4=S, 5=SW, 6=W, 7=NW`. Y increases southward. |
| `Speed` | `0=BackHalf, 1=Halted, 2=AheadHalf, 3=AheadFull`. A tank may change speed by at most 1 step per impulse. |
| `Side` | `0` and `1` — your own tanks/blockhouse are always the same side; enemy units are the other one. |
| `TerrainView.Type` | `1=Forest, 2=Water, 3=Mountain`. Plain (`0`) cells are never sent — an absent cell is Plain. |

**Turn-rate legality.** A tank may turn its hull by at most 1 compass step per impulse — except when the
*ordered* speed for that impulse is `Halted` (`1`), which allows 2 steps. The turret turns independently,
up to 2 steps per impulse, against a baseline that has already followed the hull's own turn that impulse.
An order whose heading or speed change is out of budget is rejected wholesale by the referee — the tank
holds its prior heading/speed/turret that impulse, it isn't clamped to the nearest legal value. `agent/legality.go`
reimplements this math (`TurnDistance`, `TurnAllowance`, `StepHeadingToward`, `StepSpeedToward`,
`HeadingToward`) purely from these wire integers, so both reference agents — and your own `Strategy` —
can build legal orders without guessing.

## Reference agents

Two runnable, always-legal agents live under `examples/`, both built on the same `agent` package:

- **`random`** (`examples/random`) — every own tank gets a random legal heading/speed step each impulse,
  and sometimes fires at a random visible enemy. Deterministic per seed (`math/rand/v2`, PCG), useful as
  a reproducible opponent for local testing and CI.

  ```sh
  go run ./examples/random/cmd -addr :8081 -seed 1  # serves at http://localhost:8081/v1
  ```

- **`heuristic`** (`examples/heuristic`) — deterministic, no randomness: engage the nearest visible
  enemy (turn hull and turret toward it, fire once aligned and in range), or — with no enemy in sight —
  advance toward the nearest Forest cell for concealment.

  ```sh
  go run ./examples/heuristic/cmd -addr :8082  # serves at http://localhost:8082/v1
  ```

Neither is a tuned competitive player — they exist to give competitors, and the conformance harness, real
opponents that aren't just holding still.

## Versioning & compatibility

This template's major version pins to the Control-Plane API / MCP tool-surface major version it was
built against (currently API `v1`, template `1.x`). It depends on
[`gismo-sdk-go`](https://github.com/Axemere-LLC/gismo-sdk-go) and
[`gismo-contracts`](https://github.com/Axemere-LLC/gismo-contracts) at pinned versions in `go.mod` —
bump those together when upgrading to a new API major version.

## Reporting your agent's version

The referee reads back your agent's version from the MCP `initialize` handshake
(`serverInfo.version`) and compares it against the `version_label` assigned to your agent when you
registered it with the platform (e.g. `"v2"`) — keeping the two in sync matters, since it's how the
platform attributes match results to the right rating.

Each `agent.Mount`'s reported version is derived from its `Path` (`/v2` reports `"v2"`) — register
that same string as the `version_label` when you register the generation with the platform, and the
two stay in sync automatically. See [Serving multiple versions](#serving-multiple-versions).

## Deploy it

This template gets you a listening MCP server; it doesn't host it for you. Once your `Strategy` is
implemented and tested, [`gismo-agent-hosting`](https://github.com/Axemere-LLC/gismo-agent-hosting)
is the companion repo of distributable OpenTofu modules that builds, deploys, and gives you the
endpoint URL to register — see its
[quickstart guide](https://github.com/Axemere-LLC/gismo-agent-hosting/blob/main/docs/quickstart.md)
for the full path from this repo to a registered, playable agent.

## Related repos

- [gismo-contracts](https://github.com/Axemere-LLC/gismo-contracts) — the OpenAPI + MCP JSON Schema
  contract this template's wire types are generated from
- [gismo-sdk-go](https://github.com/Axemere-LLC/gismo-sdk-go) — the REST client and MCP models this
  template imports
- [gismo-agent-python](https://github.com/Axemere-LLC/gismo-agent-python), [gismo-agent-typescript](https://github.com/Axemere-LLC/gismo-agent-typescript) — the same template in Python and TypeScript

## Testing

```sh
go build ./... && go vet ./... && go test ./...
```

- `agent/*_test.go` — the state cache, the MCP tool surface, `VersionedHandler`'s routing/isolation
  behavior, and the shared legality helpers.
- `examples/{random,heuristic}/strategy_test.go` — table-driven tests asserting every emitted order is
  legal, plus each agent's own decision logic (nearest-enemy targeting, cover-seeking, determinism).
- `integration/conformance_test.go` — drives `gismo-contracts`' conformance harness against real,
  listening `gismo-agent-go` MCP servers over real HTTP transport, for the unmodified template and both
  reference agents.
- `agent/fixtures_test.go` (`TestFixtures`) — the **fixture drift lock**: replays the scenario corpus
  in `fixtures/scenarios.json` against each mounted generation's `Strategy` and compares the resulting
  orders byte-for-byte against `fixtures/expected/*.json`. This exists to catch the hazard from
  [Serving multiple versions](#serving-multiple-versions): once a generation is rated, editing a
  shared helper it depends on (e.g. `agent/legality.go`) can silently change what an already-shipped
  `/vN` plays, without touching that generation's own code. If a drift is intentional — you've cut a
  new generation and the old one is meant to stay exactly as it was, or you're updating an unreleased
  generation on purpose — regenerate the goldens with `go test ./agent/... -run TestFixtures -update`.

## Repository layout

```
.
├── main.go                    # the template: agent.VersionedHandler({"/v1", agent.HoldStrategy{}})
├── agent/                     # MCP server, state cache, Strategy interface, VersionedHandler, legality helpers
├── examples/
│   ├── random/                # random reference agent (package random) + cmd/main.go
│   └── heuristic/             # heuristic reference agent (package heuristic) + cmd/main.go
├── fixtures/                   # scenario corpus + per-generation golden orders (the drift lock)
└── integration/                # conformance-harness integration test
```

## License

Apache 2.0 — see `LICENSE`.

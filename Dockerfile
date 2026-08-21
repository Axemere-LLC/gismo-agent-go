# syntax=docker/dockerfile:1
#
# Builds an MCP server binary: by default, this repo's own root main.go (a
# fork's real agent, once its Strategy is filled in); pass --build-arg
# EXAMPLE=random or EXAMPLE=heuristic to build one of the reference agents
# under examples/ instead. Build with (from this repo's root — see
# gismo-platform's Makefile's docker-build-agents / docker-push-agents):
#
#   docker buildx build --platform linux/amd64 --build-arg EXAMPLE=random \
#     -f Dockerfile -t <tag> .
#
# Single-repo build context, like cmd/referee-server/Dockerfile in
# gismo-platform. go.mod resolves gismo-sdk-go and gismo-contracts from the
# Go module proxy (both are public and tagged), so no sibling-repo context
# or filesystem `replace` directives are needed.

ARG EXAMPLE=

FROM golang:1.26.5@sha256:3aff6657219a4d9c14e27fb1d8976c49c29fddb70ba835014f477e1c70636647 AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG EXAMPLE
RUN set -eu; \
    if [ -n "$EXAMPLE" ]; then target="./examples/$EXAMPLE/cmd"; else target="."; fi; \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /out/agent "$target"

FROM gcr.io/distroless/static-debian12:nonroot@sha256:f5b485ea962d9bd1186b2f6b3a061191539b905b82ec395de78cbfae51f20e35
COPY --from=builder /out/agent /agent
USER nonroot:nonroot
ENTRYPOINT ["/agent"]

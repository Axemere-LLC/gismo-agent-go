# syntax=docker/dockerfile:1
#
# Builds either reference agent (random or heuristic) as a standalone MCP
# server binary. Build with (from the gismo-platform repo, this repo's
# sibling — see its Makefile's docker-build-agents / docker-push-agents):
#
#   docker buildx build --platform linux/amd64 --build-arg EXAMPLE=random \
#     -f ../gismo-agent-go/Dockerfile -t <tag> ..
#
# The build context must be the *parent* directory of this repo
# (one level above every sibling repo under gismo/), because go.mod uses
# filesystem `replace` directives to ../gismo-sdk-go and ../gismo-contracts
# (which itself replaces ../gismo-contracts a second level down) — none of
# the three modules are published yet, so a context rooted at this repo
# alone can't resolve them. See Dockerfile.dockerignore for what's excluded
# from that wider context. This whole workaround goes away once the SDK and
# contracts repos are public and tagged; at that point this can drop back to
# a normal single-repo build like cmd/referee-server/Dockerfile in
# gismo-platform.

ARG EXAMPLE=random

FROM golang:1.26.5@sha256:3aff6657219a4d9c14e27fb1d8976c49c29fddb70ba835014f477e1c70636647 AS builder
WORKDIR /src

COPY gismo-contracts/go.mod gismo-contracts/go.sum ./gismo-contracts/
COPY gismo-sdk-go/go.mod gismo-sdk-go/go.sum ./gismo-sdk-go/
COPY gismo-agent-go/go.mod gismo-agent-go/go.sum ./gismo-agent-go/
RUN cd gismo-agent-go && go mod download

COPY gismo-contracts ./gismo-contracts
COPY gismo-sdk-go ./gismo-sdk-go
COPY gismo-agent-go ./gismo-agent-go

ARG EXAMPLE
RUN cd gismo-agent-go && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /out/agent ./examples/${EXAMPLE}/cmd

FROM gcr.io/distroless/static-debian12:nonroot@sha256:f5b485ea962d9bd1186b2f6b3a061191539b905b82ec395de78cbfae51f20e35
COPY --from=builder /out/agent /agent
USER nonroot:nonroot
ENTRYPOINT ["/agent"]

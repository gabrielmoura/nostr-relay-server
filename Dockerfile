# syntax=docker/dockerfile:1.7

ARG GO_VERSION=1.26.2
ARG NODE_VERSION=22.19.0
ARG ALPINE_VERSION=3.23

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS go-deps
WORKDIR /src
ENV GOMODCACHE=/gomodcache
COPY go.mod go.sum ./
RUN go mod download

FROM --platform=$BUILDPLATFORM node:${NODE_VERSION}-alpine AS frontend-build
WORKDIR /src/infra/dash
COPY infra/dash/package.json infra/dash/pnpm-lock.yaml ./
RUN corepack enable && pnpm install --frozen-lockfile
COPY infra/dash/ ./
RUN pnpm run build

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS backend-build
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src
RUN apk add --no-cache git
COPY go.mod go.sum ./
COPY --from=go-deps /gomodcache /go/pkg/mod
COPY . .
COPY --from=frontend-build /src/infra/dash/dist ./infra/dash/dist
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} GOAMD64=v2 \
    go build -trimpath -ldflags="-w -s" -o /out/nrserver ./cmd/nrserver

FROM alpine:${ALPINE_VERSION} AS runtime-prep
RUN apk add --no-cache ca-certificates && update-ca-certificates
RUN addgroup -S -g 10001 appuser \
    && adduser -S -D -H -h /app -s /sbin/nologin -u 10001 -G appuser appuser

FROM scratch

WORKDIR /app

COPY --from=backend-build /out/nrserver /app/nrserver
COPY --from=runtime-prep /etc/passwd /etc/passwd
COPY --from=runtime-prep /etc/group /etc/group
COPY --from=runtime-prep /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
VOLUME ["/app/conf.yaml"]

USER appuser:appuser

ARG BUILD_DATE="2026-04-17T00:00:00Z"
ARG VERSION=0.1.0

LABEL org.opencontainers.image.source="https://github.com/gabrielmoura/nostr-relay-server"
LABEL org.opencontainers.image.title="nostr-relay-server"
LABEL org.opencontainers.image.description="nostr-relay-server is a simple relay server for Nostr"
LABEL org.opencontainers.image.created=$BUILD_DATE
LABEL org.opencontainers.image.version=$VERSION
LABEL org.opencontainers.image.authors="Gabriel Moura <gmouradev96@gmail.com>"

ENV PORT=9090
EXPOSE 9090
EXPOSE 9091

ENTRYPOINT ["/app/nrserver", "server"]

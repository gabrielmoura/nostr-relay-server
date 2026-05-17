# syntax=docker/dockerfile:1.7

ARG GO_VERSION=1.26.2
ARG NODE_VERSION=22.19.0
ARG ALPINE_VERSION=3.23

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS go-deps
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

FROM --platform=$BUILDPLATFORM node:${NODE_VERSION}-alpine AS frontend-deps
ENV PNPM_HOME=/pnpm
ENV PATH=${PNPM_HOME}:${PATH}
ENV NPM_CONFIG_FETCH_RETRIES=5
WORKDIR /src/infra/dash
COPY infra/dash/package.json infra/dash/pnpm-lock.yaml ./
RUN for attempt in 1 2 3 4 5; do npm install -g pnpm@9.15.9 && break; sleep 10; done
RUN --mount=type=cache,target=/pnpm/store \
    pnpm config set store-dir /pnpm/store && \
    pnpm config set fetch-retries 5 && \
    pnpm install --frozen-lockfile

FROM frontend-deps AS frontend-build
COPY infra/dash/ ./
COPY ref/nips /src/ref/nips
RUN --mount=type=cache,target=/pnpm/store \
    pnpm run build

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS backend-build
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src
COPY --from=go-deps /go/pkg/mod /go/pkg/mod
COPY . .
COPY --from=frontend-build /src/infra/dash/dist ./infra/dash/dist
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
    go build -buildvcs=false -trimpath -ldflags="-s -w" -o /out/nrserver ./cmd/nrserver

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=backend-build /out/nrserver /app/nrserver
COPY --from=backend-build /src/nostr.png /app/nostr.png

ARG BUILD_DATE="2026-04-17T00:00:00Z"
ARG VERSION=0.1.0

LABEL org.opencontainers.image.source="https://github.com/gabrielmoura/nostr-relay-server"
LABEL org.opencontainers.image.title="nostr-relay-server"
LABEL org.opencontainers.image.description="Nostr relay server with embedded admin dashboard"
LABEL org.opencontainers.image.created=$BUILD_DATE
LABEL org.opencontainers.image.version=$VERSION
LABEL org.opencontainers.image.authors="Gabriel Moura <gmouradev96@gmail.com>"

EXPOSE 9090 9091

ENTRYPOINT ["/app/nrserver"]
CMD ["server"]

FROM golang:1.23.3-alpine3.20 AS prepare
RUN apk add --no-cache git ca-certificates && update-ca-certificates

ENV USER=appuser
ENV UID=10001

RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "/app" \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid "${UID}" \
    "${USER}"

FROM golang:1.23.3-alpine3.20 AS builder

WORKDIR /app

COPY . .

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64  go build -a -installsuffix cgo -ldflags="-w -s" -o /app/nrserver /app/cmd/nrserver/main.go

FROM scratch

WORKDIR /app

COPY --from=builder /app/nrserver /app/nrserver
COPY --from=prepare /etc/passwd /etc/passwd
COPY --from=prepare /etc/group /etc/group
COPY --from=prepare /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
VOLUME /app/conf.yml


USER appuser:appuser

ARG BUILD_DATE="2024-01-01T00:00:00Z"
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

ENTRYPOINT ["/app/nrserver","server"]
FROM golang:1.26-bookworm@sha256:6c5605ab3a9a9fb3c4eafe5b3d63cdbf3881caf113262b67862547b54a9db599 AS builder

WORKDIR /source
COPY go.mod go.sum ./
RUN go mod download
COPY agent ./agent
COPY protocol ./protocol
COPY server ./server
RUN CGO_ENABLED=1 go build -trimpath -ldflags="-s -w" -o /out/tempo-server ./server/cmd/tempo-server

FROM debian:13-slim@sha256:3a39a0592364683e6bab97937b72cad5a8fa6dcbbee90edb3bb48c7f8e94f258

RUN apt-get update \
    && apt-get install --yes --no-install-recommends ca-certificates curl \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd --system tempo-server \
    && useradd --system --uid 10001 --gid tempo-server --home-dir /nonexistent --shell /usr/sbin/nologin tempo-server \
    && install -d -o tempo-server -g tempo-server -m 0700 /var/lib/tempo-server \
    && install -d -o root -g tempo-server -m 0750 /etc/tempo-server

COPY --from=builder --chown=root:root /out/tempo-server /usr/local/bin/tempo-server
COPY --chown=root:tempo-server server/config.docker.toml /etc/tempo-server/config.toml

USER tempo-server:tempo-server
EXPOSE 8080
HEALTHCHECK --interval=15s --timeout=3s --start-period=10s --retries=3 \
  CMD curl --fail --silent http://127.0.0.1:8080/healthz || exit 1
ENTRYPOINT ["/usr/local/bin/tempo-server", "-config", "/etc/tempo-server/config.toml"]

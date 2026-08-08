FROM golang:1.26-trixie AS builder

WORKDIR /source
COPY go.mod go.sum ./
RUN go mod download
COPY agent ./agent
COPY protocol ./protocol
COPY server ./server
RUN CGO_ENABLED=1 go build -trimpath -ldflags="-s -w" -o /out/tempo-server ./server/cmd/tempo-server

FROM debian:13-slim

RUN groupadd --system tempo-server \
    && useradd --system --gid tempo-server --home-dir /nonexistent --shell /usr/sbin/nologin tempo-server \
    && install -d -o tempo-server -g tempo-server -m 0700 /var/lib/tempo-server \
    && install -d -o root -g tempo-server -m 0750 /etc/tempo-server

COPY --from=builder --chown=root:root /out/tempo-server /usr/local/bin/tempo-server
COPY --chown=root:tempo-server server/config.docker.toml /etc/tempo-server/config.toml
COPY --chown=root:root server/web/templates /usr/share/tempo-server/web/templates
COPY --chown=root:root server/web/static /usr/share/tempo-server/web/static

USER tempo-server:tempo-server
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/tempo-server", "-config", "/etc/tempo-server/config.toml"]

# clash-gateway

`clash-gateway` is a Docker-first traffic gateway that lets app containers opt into a specific Mihomo/Clash gateway instance through labels.

## What It Does

- Runs one logical gateway per container instance
- Supports same-host multi-gateway deployments
- Supports either subscription mode or mounted config file mode
- Supports scheduled subscription refresh or manual refresh only
- Exposes a bundled Clash UI by default, with optional HTTP proxy and SOCKS5 proxy ports when explicitly enabled
- Lets app containers target a gateway with:
  - `clash-gateway.gateway=<name>`
  - `clash-gateway.allow-attach=true`
  - `clash-gateway.disable=true`

## Current Shape

The current implementation includes:

- `gatewayd` for config loading, Mihomo process supervision, state persistence, Docker snapshot polling, Docker event-triggered refresh, and UI static serving
- `gatewayctl` for `status`, `validate-config`, and `refresh`
- Docker CLI based container discovery and bridge attach operations
- State files under `/data/state/status.json`

The transparent `TCP + DNS` routing layer is scaffolded as a dedicated `internal/netns` package, but the live iptables/netns mutation path is not yet wired into the runtime loop.

## Configuration

### Required

- `GATEWAY_NAME`
- `CONFIG_MODE=subscription|file`

### Subscription Mode

- `SUBSCRIPTION_URL`
- `AUTO_UPDATE=true|false`
- `UPDATE_INTERVAL` or `UPDATE_CRON`

### File Mode

- `CONFIG_FILE_PATH`

### Ports

- `HTTP_PROXY_PORT`
- `SOCKS_PROXY_PORT`
- `EXTERNAL_CONTROLLER_PORT`
- `UI_PORT`
- `EXTERNAL_CONTROLLER_SECRET`

### Docker/Gateway

- `MANAGED_NETWORK_NAME`
- `AUTO_ATTACH_CONTAINERS=true|false`
- `DATA_DIR`
- `LOG_LEVEL`

## Commands

```bash
gatewayctl validate-config
gatewayctl status
gatewayctl refresh
```

## Quick Deploy

Image:

```bash
ghcr.io/monlor/clash-gateway:latest
```

### Subscription Mode

Only the required variables:

```bash
docker run -d \
  --name clash-gateway \
  --restart unless-stopped \
  --privileged \
  --pid host \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v $PWD/data:/data \
  -e GATEWAY_NAME=main \
  -e CONFIG_MODE=subscription \
  -e SUBSCRIPTION_URL='https://example.com/subscribe' \
  -e EXTERNAL_CONTROLLER_SECRET='change-me' \
  -p 9080:9080 \
  ghcr.io/monlor/clash-gateway:latest
```

### Local Config File Mode

```bash
docker run -d \
  --name clash-gateway \
  --restart unless-stopped \
  --privileged \
  --pid host \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v $PWD/data:/data \
  -v $PWD/mihomo.yaml:/etc/clash-gateway/config.yaml:ro \
  -e GATEWAY_NAME=main \
  -e CONFIG_MODE=file \
  -e CONFIG_FILE_PATH=/etc/clash-gateway/config.yaml \
  -e EXTERNAL_CONTROLLER_SECRET='change-me' \
  -p 9080:9080 \
  ghcr.io/monlor/clash-gateway:latest
```

Default ports if you do not override them:

- UI: `9080`
- internal external-controller: `9090`
- HTTP proxy: disabled by default
- SOCKS5 proxy: disabled by default

After the container starts:

- Open `http://<host>:9080`
- The bundled UI proxies requests to the internal Mihomo controller automatically
- Other containers join this gateway with:
  - `clash-gateway.gateway=main`
  - `clash-gateway.allow-attach=true`

### Optional: expose host-side HTTP/SOCKS proxy ports

If you want to use the container as a host-level proxy too, explicitly enable and publish them:

```bash
docker run -d \
  --name clash-gateway \
  --restart unless-stopped \
  --privileged \
  --pid host \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v $PWD/data:/data \
  -e GATEWAY_NAME=main \
  -e CONFIG_MODE=subscription \
  -e SUBSCRIPTION_URL='https://example.com/subscribe' \
  -e EXTERNAL_CONTROLLER_SECRET='change-me' \
  -e HTTP_PROXY_PORT=7890 \
  -e SOCKS_PROXY_PORT=7891 \
  -p 7890:7890 \
  -p 7891:7891 \
  -p 9080:9080 \
  ghcr.io/monlor/clash-gateway:latest
```

## Build From Source

```bash
make test
make build
docker build -t monlor/clash-gateway:dev .
```

## Example

See [deploy/docker-compose.yml](deploy/docker-compose.yml) for a two-gateway example:

- `gateway-hk`
- `gateway-us`
- `app-a` routed to `hk`
- `app-b` routed to `us`

## UI

The image bundles MetaCubeXD assets from the official `gh-pages` branch. GitHub shows `v1.245.1` as the latest MetaCubeXD release as of 2026-04-21:

- https://github.com/MetaCubeX/metacubexd/releases/tag/v1.245.1

## Runtime Requirements

- Docker socket mounted into the container
- `privileged: true`
- `pid: host`
- A base image that provides the `mihomo` binary

## GitHub Actions

The repo includes [`.github/workflows/docker.yml`](.github/workflows/docker.yml) to build multi-arch images for:

- `linux/amd64`
- `linux/arm64`

It publishes to:

```bash
ghcr.io/monlor/clash-gateway
```

Trigger behavior:

- push to `main`: build and push branch tags plus `latest`
- tag `v*`: build and push tag image
- pull request: build only, no push
- manual dispatch: build and push

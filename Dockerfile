ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH
ARG MIHOMO_BASE_IMAGE=docker.io/metacubex/mihomo:Alpha

FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS build
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} go build -o /out/gatewayd ./cmd/gatewayd && \
    CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} go build -o /out/gatewayctl ./cmd/gatewayctl

FROM ${MIHOMO_BASE_IMAGE}
RUN apk add --no-cache bash ca-certificates curl docker-cli git iproute2 iptables jq && \
    git clone --depth 1 --branch gh-pages https://github.com/MetaCubeX/metacubexd.git /opt/metacubexd
COPY --from=build /out/gatewayd /usr/local/bin/gatewayd
COPY --from=build /out/gatewayctl /usr/local/bin/gatewayctl
COPY scripts/entrypoint.sh /entrypoint.sh
RUN chmod +x /usr/local/bin/gatewayd /usr/local/bin/gatewayctl /entrypoint.sh
ENV DATA_DIR=/data
VOLUME ["/data"]
ENTRYPOINT ["/entrypoint.sh"]

GO ?= go
BINDIR ?= bin

.PHONY: test build clean image

test:
	$(GO) test ./...

build:
	mkdir -p $(BINDIR)
	$(GO) build -o $(BINDIR)/gatewayd ./cmd/gatewayd
	$(GO) build -o $(BINDIR)/gatewayctl ./cmd/gatewayctl

clean:
	rm -rf $(BINDIR)

image:
	docker build -t monlor/clash-gateway:dev .

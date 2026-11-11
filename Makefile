# Prompt Fighter build. `make` builds the web app and bin/pf.
#
#   make            web app + bin/pf (version stamped from git)
#   make test       go vet, go test (includes the web tests)
#   make release    cross-platform archives + SHA256SUMS in dist/
#   make smoke      MCP smoke test against a temporary pf serve
#   make assets     re-encode audio and the cover (needs ffmpeg)
#   make clean
#
# pf embeds web/dist, which is not committed, so every Go target builds the
# web app first.

VERSION   ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS   := -s -w -X main.version=$(VERSION)
PLATFORMS ?= linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64
GO        ?= go

.PHONY: all web build test release smoke assets clean

all: build

web/node_modules/.package-lock.json: web/package-lock.json
	cd web && npm ci

web: web/node_modules/.package-lock.json
	cd web && npm run build

build: web
	mkdir -p bin
	CGO_ENABLED=0 $(GO) build -trimpath -ldflags '$(LDFLAGS)' -o bin/pf ./cmd/pf

test: web
	$(GO) vet ./...
	$(GO) test ./...

release: web
	rm -rf dist && mkdir -p dist
	set -e; for p in $(PLATFORMS); do \
		os=$${p%/*}; arch=$${p#*/}; ext=; [ "$$os" = windows ] && ext=.exe; \
		name=pf-$(VERSION)-$$os-$$arch; mkdir -p dist/$$name; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch $(GO) build -trimpath -ldflags '$(LDFLAGS)' -o dist/$$name/pf$$ext ./cmd/pf; \
		cp README.md dist/$$name/; \
		if [ "$$os" = windows ]; then (cd dist && zip -qr $$name.zip $$name); else tar -C dist -czf dist/$$name.tar.gz $$name; fi; \
		rm -rf dist/$$name; \
	done
	cd dist && if command -v sha256sum >/dev/null; then sha256sum pf-* > SHA256SUMS; else shasum -a 256 pf-* > SHA256SUMS; fi

smoke: build
	set -e; data=$$(mktemp -d); port=18765; \
	bin/pf serve --addr 127.0.0.1:$$port --data $$data > $$data/serve.log 2>&1 & pid=$$!; \
	trap 'kill $$pid 2>/dev/null; rm -rf $$data' EXIT; \
	for i in $$(seq 30); do curl -sf http://127.0.0.1:$$port/healthz >/dev/null && break; sleep 1; done; \
	.github/scripts/mcp-smoke.sh bin/pf http://127.0.0.1:$$port

assets:
	cd web && npm run assets

clean:
	rm -rf bin dist web/dist

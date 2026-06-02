.PHONY: build test verify goreleaser release-check release-snapshot release clean

BIN := bin/puppt
GORELEASER_VERSION ?= v2.16.0
GORELEASER_BIN := .tools/goreleaser/$(GORELEASER_VERSION)/goreleaser
GORELEASER ?= $(GORELEASER_BIN)

build:
	go build -trimpath -ldflags "-s -w" -o $(BIN) ./cmd/puppt

test:
	go test -count=1 ./...

verify: test
	git diff --check
	go run ./cmd/puppt --help >/dev/null

goreleaser: $(GORELEASER_BIN)

$(GORELEASER_BIN):
	set -eu; \
	os=$$(uname -s); \
	arch=$$(uname -m); \
	case "$$os" in \
		Darwin|Linux) gore_os="$$os" ;; \
		*) echo "unsupported GoReleaser OS: $$os" >&2; exit 1 ;; \
	esac; \
	case "$$arch" in \
		arm64|aarch64) gore_arch="arm64" ;; \
		x86_64|amd64) gore_arch="x86_64" ;; \
		*) echo "unsupported GoReleaser arch: $$arch" >&2; exit 1 ;; \
	esac; \
	tmp=$$(mktemp -d); \
	trap 'rm -rf "$$tmp"' EXIT; \
	asset="goreleaser_$${gore_os}_$${gore_arch}.tar.gz"; \
	base="https://github.com/goreleaser/goreleaser/releases/download/$(GORELEASER_VERSION)"; \
	curl -fsSLo "$$tmp/checksums.txt" "$$base/checksums.txt"; \
	curl -fsSLo "$$tmp/$$asset" "$$base/$$asset"; \
	grep "  $$asset$$" "$$tmp/checksums.txt" > "$$tmp/checksums.filtered"; \
	(cd "$$tmp" && shasum -a 256 -c checksums.filtered); \
	mkdir -p "$(dir $(GORELEASER_BIN))"; \
	tar -xzf "$$tmp/$$asset" -C "$(dir $(GORELEASER_BIN))" goreleaser; \
	chmod +x "$(GORELEASER_BIN)"

release-check: goreleaser
	$(GORELEASER) check

release-snapshot: goreleaser
	$(GORELEASER) release --snapshot --clean

release: goreleaser
	$(GORELEASER) release --clean

clean:
	rm -rf bin dist .tools

.PHONY: build test verify clean

BIN := bin/puppt

build:
	go build -trimpath -ldflags "-s -w" -o $(BIN) ./cmd/puppt

test:
	go test -count=1 ./...

verify: test
	git diff --check
	go run ./cmd/puppt --help >/dev/null

clean:
	rm -rf bin

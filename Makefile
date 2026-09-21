# pwprofiler — authorized password-audit wordlist generator
BIN := pwprofiler
PKG := ./cmd/pwprofiler

.PHONY: build test fmt vet run clean

build:
	go build -o $(BIN) $(PKG)

test:
	go test ./...

fmt:
	gofmt -w .

vet:
	go vet ./...

run: build
	./$(BIN) generate --config testdata/sample.yaml

clean:
	rm -f $(BIN)

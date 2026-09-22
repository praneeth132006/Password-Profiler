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

.PHONY: install deb
PREFIX ?= /usr/local

install: build
	install -d "$(DESTDIR)$(PREFIX)/bin"
	install -m 755 $(BIN) "$(DESTDIR)$(PREFIX)/bin/$(BIN)"

deb:
	./scripts/build-deb.sh

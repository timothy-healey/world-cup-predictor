.PHONY: build test test-verbose run install clean

build:
	go build -o bin/wcp ./cmd/wcp

test:
	go test ./...

test-verbose:
	go test -v ./...

run: build
	./bin/wcp

install: build
	@mkdir -p $(HOME)/bin
	cp bin/wcp $(HOME)/bin/wcp
	@echo "installed to $(HOME)/bin/wcp"

clean:
	rm -rf bin/

.PHONY: build test test-verbose run install clean build-frontend

FRONTEND := ../frontend

build-frontend:
	npm --prefix $(FRONTEND) ci
	npm --prefix $(FRONTEND) run build
	rm -rf internal/server/dist
	mkdir -p internal/server/dist
	cp -R $(FRONTEND)/dist/. internal/server/dist/
	touch internal/server/dist/.gitkeep

build: build-frontend
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
	rm -rf bin/ internal/server/dist
	mkdir -p internal/server/dist
	touch internal/server/dist/.gitkeep

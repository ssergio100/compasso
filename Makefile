.PHONY: all build build-agent build-server fmt fmt-check lint test test-go test-ui test-migrations test-security clean

all: test

build: build-agent build-server
	go build ./...

build-agent:
	mkdir -p bin
	go build -o bin/tempo-agent ./agent/cmd/tempo-agent
	go build -o bin/tempo-pam-check ./agent/cmd/tempo-pam-check
	go build -o bin/tempo-pam-setup ./agent/cmd/tempo-pam-setup

build-server:
	mkdir -p bin
	go build -o bin/tempo-server ./server/cmd/tempo-server

fmt:
	gofmt -w $$(find agent server protocol -type f -name '*.go' 2>/dev/null)

fmt-check:
	@unformatted="$$(gofmt -l $$(find agent server protocol -type f -name '*.go' 2>/dev/null))"; \
	if [ -n "$$unformatted" ]; then \
		echo "Arquivos Go sem formatação:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi

lint: fmt-check
	go vet ./...

test-go:
	go test ./...

test-ui:
	cd local-ui && python3 -m unittest discover -v

test-migrations:
	./scripts/test-migrations.sh

test-security:
	./scripts/test-security-packaging.sh

test: lint test-go test-ui test-migrations test-security build

clean:
	rm -rf ./bin ./dist

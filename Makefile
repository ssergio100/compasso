.PHONY: all build fmt fmt-check lint test test-go test-migrations clean

all: test

build:
	go build ./...

fmt:
	gofmt -w $$(find agent server -type f -name '*.go' 2>/dev/null)

fmt-check:
	@unformatted="$$(gofmt -l $$(find agent server -type f -name '*.go' 2>/dev/null))"; \
	if [ -n "$$unformatted" ]; then \
		echo "Arquivos Go sem formatação:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi

lint: fmt-check
	go vet ./...

test-go:
	go test ./...

test-migrations:
	./scripts/test-migrations.sh

test: lint test-go test-migrations build

clean:
	rm -rf ./bin ./dist

.PHONY: all build build-agent build-agent-portable build-server admin-ui-dev package-client package-client-tar package-deb package-server test-deb test-server-package fmt fmt-check lint test test-go test-ui test-admin-ui test-migrations test-security clean

all: test

build: build-agent build-server
	go build ./...

build-agent:
	mkdir -p bin
	go build -o bin/tempo-agent ./agent/cmd/tempo-agent
	go build -o bin/tempo-agent-configure ./agent/cmd/tempo-agent-configure
	go build -o bin/compasso-session-logout ./agent/cmd/compasso-session-logout
	go build -o bin/tempo-pam-check ./agent/cmd/tempo-pam-check
	go build -o bin/tempo-pam-setup ./agent/cmd/tempo-pam-setup

build-agent-portable:
	./scripts/build-portable-client-binaries.sh

build-server:
	mkdir -p bin
	go build -o bin/tempo-server ./server/cmd/tempo-server

admin-ui-dev:
	python3 -m http.server 8182 --directory admin-ui

package-client: package-deb

package-client-tar: build-agent-portable
	./scripts/build-client-package.sh

package-deb: build-agent-portable
	./scripts/build-debian-package.sh

package-server:
	./scripts/build-server-package.sh

test-deb: package-deb
	./scripts/test-debian-package.sh

test-server-package:
	./scripts/test-server-package.sh

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

test-admin-ui:
	node --check admin-ui/api-client.js
	node --check admin-ui/app.js
	node --test admin-ui/*.test.js

test-migrations:
	./scripts/test-migrations.sh

test-security:
	./scripts/test-security-packaging.sh

test: lint test-go test-ui test-admin-ui test-migrations test-security build

clean:
	rm -rf ./bin ./dist

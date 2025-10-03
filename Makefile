# Makefile for nexuscli project

.PHONY: build test test-e2e test-short generate-docs

build:
	goreleaser release --snapshot --clean --skip=publish

test:
	cd nexuscli-go && go test -v

test-short:
	cd nexuscli-go && go test -v -short

test-e2e:
	cd nexuscli-go && go test -v -run TestEndToEndUploadDownload -timeout 15m

generate-docs:
	@echo "Generating API documentation from OpenAPI spec..."
	@command -v widdershins >/dev/null 2>&1 || { echo >&2 "widdershins is not installed. Installing..."; npm install -g widdershins; }
	@mkdir -p .github
	widdershins --omitHeader --resolve docs/sonatype-nexus-repository-api.json -o .github/nexus-api-reference.md
	@echo "Documentation generated at .github/nexus-api-reference.md"

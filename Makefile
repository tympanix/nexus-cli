# Makefile for nexuscli project

.PHONY: build test test-e2e test-short

build:
	goreleaser release --snapshot --clean --skip=publish

# Run unit tests only
test:
	go test -v -short ./...

# Run end-to-end tests only
test-e2e:
	go test -v -run TestEndToEnd -timeout 15m ./internal/nexus

# Run complete test suite including unit and integration tests
test-all:
	go test -v ./...

# Makefile for nexuscli project

.PHONY: build test test-e2e test-short format format-check

build:
	goreleaser release --snapshot --clean --skip=publish

test:
	go test -v ./...

test-short:
	go test -v -short ./...

test-e2e:
	go test -v -run TestEndToEndUploadDownload -timeout 15m ./internal/nexus

format:
	gofmt -w .

format-check:
	@if [ -n "$$(gofmt -l .)" ]; then \
		echo "The following files are not formatted correctly:"; \
		gofmt -l .; \
		exit 1; \
	fi

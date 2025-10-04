# Makefile for nexuscli project

.PHONY: build test test-e2e test-short

build:
	goreleaser release --snapshot --clean --skip=publish

test:
	go test -v ./...

test-short:
	go test -v -short ./...

test-e2e:
	go test -v -run TestEndToEndUploadDownload -timeout 15m ./internal/nexus

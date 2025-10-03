# Makefile for nexuscli project

.PHONY: build test

build:
	goreleaser release --snapshot --clean --skip=publish

test:
	go test -v ./...

# Makefile for nexuscli project

.PHONY: build test

build:
	goreleaser release --snapshot --clean --skip=publish

test:
	cd nexuscli-go && go test -v

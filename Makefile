# Makefile for nexuscli project

.PHONY: build

build:
	goreleaser release --snapshot --clean --skip=publish

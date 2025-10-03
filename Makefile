# Makefile for nexuscli project

.PHONY: build test test-e2e test-short

build:
	goreleaser release --snapshot --clean --skip=publish

test:
	cd nexuscli-go && go test -v

test-short:
	cd nexuscli-go && go test -v -short

test-e2e:
	cd nexuscli-go && go test -v -run TestEndToEndUploadDownload -timeout 15m

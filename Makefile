# Makefile for nexuscli project

.PHONY: venv build

venv:
	python3 -m venv venv
	. venv/bin/activate && pip install --upgrade pip
	. venv/bin/activate && pip install .

build:
	goreleaser release --snapshot --clean --skip=publish

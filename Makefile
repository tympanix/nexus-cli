# Makefile for nexuscli project

.PHONY: venv

venv:
	python3 -m venv venv
	. venv/bin/activate && pip install --upgrade pip
	. venv/bin/activate && pip install .

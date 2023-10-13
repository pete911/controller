#!make
SHELL := /bin/bash
.SHELLFLAGS := -ec

test:
	$(info *** [go test] ***)
	go clean -testcache && go test -cover ./...
.PHONY:test

e2e-test:
	$(info *** [end to end tests] ***)
	./e2e/e2e
.PHONY:e2e-test

build: test
	$(info *** [go build] ***)
	go build
.PHONY:build

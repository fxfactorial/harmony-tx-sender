SHELL := /bin/bash
version := $(shell git rev-list --count HEAD)
commit := $(shell git describe --always --long --dirty)
built_at := $(shell date +%FT%T%z)
built_by := ${USER}

flags := -gcflags="all=-N -l -c 2"
ldflags := -X main.version=v${version} -X main.commit=${commit}
ldflags += -X main.builtAt=${built_at} -X main.builtBy=${built_by}

dist := ./dist/harmony-tx-sender
env := GO111MODULE=on
DIR := ${CURDIR}

all:
	$(env) go build -o $(dist) -ldflags="$(ldflags)" main.go

static:
	$(env) go build -o $(dist) -ldflags="$(ldflags) -w -extldflags \"-static\"" main.go

debug:
	$(env) go build $(flags) -o $(dist) -ldflags="$(ldflags)" main.go

.PHONY:clean

clean:
	@rm -f $(dist)
	@rm -rf ./dist

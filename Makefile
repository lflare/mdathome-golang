# ########################################################## #
# Makefile for Golang Project
# Includes cross-compiling, installation, cleanup
# ########################################################## #

# Check for required command tools to build or stop immediately
EXECUTABLES = git go find pwd
K := $(foreach exec,$(EXECUTABLES),\
        $(if $(shell which $(exec)),some string,$(error "No $(exec) in PATH)))

ROOT_DIR := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
MAKEFILE := $(lastword $(MAKEFILE_LIST))

BINARY = mdathome-golang
VERSION = `git describe --tag`
BUILD = `git rev-parse HEAD`
PLATFORMS = linux windows
ARCHITECTURES = 386 amd64 arm arm64


LDFLAGS = -ldflags "-X main.Version=${VERSION} -X main.Build=${BUILD}"

default:
	export GO111MODULE=on
	export CGO_ENABLED=0
	go build -trimpath -ldflags '-s -w' .

local:
	export GO111MODULE=off
	export CGO_ENABLED=0
	go build -trimpath -ldflags '-s -w' .

snapshot:
	goreleaser build --rm-dist --snapshot

all:
	goreleaser build --rm-dist

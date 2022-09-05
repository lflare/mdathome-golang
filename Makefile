# ########################################################## #
# Makefile for Golang Project
# Includes cross-compiling, installation, cleanup
# ########################################################## #

# Check for required command tools to build or stop immediately
EXECUTABLES = git go find pwd upx
K := $(foreach exec,$(EXECUTABLES),\
        $(if $(shell which $(exec)),some string,$(error "No $(exec) in PATH)))

ROOT_DIR := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
MAKEFILE := $(lastword $(MAKEFILE_LIST))

BINARY = mdathome-golang
VERSION = $(shell git describe --tag | cut -d '-' -f -2 | tr '-' '.')
BUILD = $(shell git rev-parse HEAD)
PLATFORMS = linux windows
ARCHITECTURES = 386 amd64 arm arm64


LDFLAGS = "-X github.com/lflare/mdathome-golang/internal/mdathome.ClientVersion=${VERSION} -X mdathome.Build=${BUILD}"

default:
	CGO_ENABLED=0 go build -o ./mdathome-golang -tags netgo -trimpath -ldflags=${LDFLAGS} .
	upx -qq mdathome-golang

snapshot:
	LDFLAGS=${LDFLAGS} goreleaser build --rm-dist --snapshot
	find 'build/' -name 'mdathome-*' | grep -v 'darwin' | xargs -n 1 upx -qq

all:
	LDFLAGS=${LDFLAGS} goreleaser build --rm-dist
	find 'build/' -name 'mdathome-*' | grep -v 'darwin' | xargs -n 1 upx -qq

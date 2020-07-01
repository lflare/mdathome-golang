# ########################################################## #
# Makefile for Golang Project
# Includes cross-compiling, installation, cleanup
# ########################################################## #

# Check for required command tools to build or stop immediately
EXECUTABLES = git go find pwd
K := $(foreach exec,$(EXECUTABLES),\
        $(if $(shell which $(exec)),some string,$(error "No $(exec) in PATH)))

ROOT_DIR:=$(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))

BINARY=mdathome-golang
VERSION=`git describe --tag`
BUILD=`git rev-parse HEAD`
PLATFORMS=linux windows
ARCHITECTURES=386 amd64 arm arm64

LDFLAGS=-ldflags "-X main.Version=${VERSION} -X main.Build=${BUILD}"

default: all

all: clean buildall

clean:
	rm -r build/ || exit 0

buildall:
	$(foreach GOOS, $(PLATFORMS), \
	$(foreach GOARCH, $(ARCHITECTURES), \
		$(eval EXT := $(if $(filter $(GOOS),windows), ".exe", "")) \
		$(shell export GOOS=$(GOOS); \
			    export GOARCH=$(GOARCH); \
			    if [[ $(GOOS) == "windows" ]]; then \
			        export EXT=.exe; \
			    fi; \
			    go build -o build/$(BINARY)-$(VERSION)-$(GOOS)-$(GOARCH)$(EXT))))

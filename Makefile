GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
GOARM ?=
DIST ?= dist

EXT :=
ifeq ($(GOOS),windows)
EXT := .exe
endif

ARCH := $(GOARCH)
ifneq ($(GOARM),)
ARCH := $(GOARCH)v$(GOARM)
endif

OUTPUT := $(DIST)/leoxy-$(GOOS)-$(ARCH)$(EXT)

.PHONY: build test run

build:
	go build -trimpath -o $(OUTPUT) ./app/cli

test:
	go test ./...

run:
	go run ./app/cli

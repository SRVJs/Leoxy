GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
DIST ?= dist

EXT :=
ifeq ($(GOOS),windows)
EXT := .exe
endif

OUTPUT := $(DIST)/leoxy-$(GOOS)-$(GOARCH)$(EXT)

.PHONY: build run test clean

build:
	go build -trimpath -o $(OUTPUT) ./app/cli

run:
	go run ./app/cli

test:
	go test ./...

clean:
	go clean

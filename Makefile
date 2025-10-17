
BIN_DIR := bin

ifeq ($(OS),Windows_NT)
  EXE   := .exe
  MKDIR := if not exist $(BIN_DIR) mkdir $(BIN_DIR)
  RM    := rmdir /S /Q $(BIN_DIR)
else
  EXE   :=
  MKDIR := mkdir -p $(BIN_DIR)
  RM    := rm -rf $(BIN_DIR)
endif

CONTROL := $(BIN_DIR)/xdp47-control$(EXE)
AGENT   := $(BIN_DIR)/xdp47-agent$(EXE)

GO ?= go

.PHONY: all build clean run-control run-agent deps
all: build

deps:
	$(GO) mod tidy

build: deps
	$(MKDIR)
	$(GO) build -o $(CONTROL) ./cmd/xdp47-control
	$(GO) build -o $(AGENT)   ./cmd/xdp47-agent

run-control: build
	$(CONTROL)

run-agent: build
	$(AGENT)

clean:
	-$(RM)

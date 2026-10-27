# 本文件提供后端常用开发命令。
GO ?= go
GOCACHE ?= $(CURDIR)/.gocache
GOMODCACHE ?= $(CURDIR)/.gomodcache

.PHONY: run test test-base tidy build fmt migrate-up migrate-down init-secret

run:
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) $(GO) run ./cmd/server

test:
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) $(GO) test ./...

test-base:
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) $(GO) test ./internal/base/...

tidy:
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) $(GO) mod tidy

build:
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) $(GO) build -o bin/wfagent-server ./cmd/server

migrate-up:
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) $(GO) run ./cmd/migrate up

migrate-down:
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) $(GO) run ./cmd/migrate down

fmt:
	$(GO) fmt ./...

init-secret:
	@echo JWT_ACCESS_SECRET=$$(openssl rand -hex 32) >> .env
	@echo JWT_REFRESH_SECRET=$$(openssl rand -hex 32) >> .env

.RECIPEPREFIX := >
SHELL := /usr/bin/env bash

GO ?= go
PYTHON ?= python3.12

.PHONY: verify-tools tree status fmt-check test test-race vet check

verify-tools:
> @printf '\n===== GO =====\n'
> @$(GO) version
> @printf '\n===== PYTHON =====\n'
> @$(PYTHON) --version
> @printf '\n===== DOCKER =====\n'
> @docker version --format 'Client={{.Client.Version}} Server={{.Server.Version}}'
> @docker compose version
> @printf '\n===== KUBERNETES =====\n'
> @kubectl version --client --output=yaml | sed -n '1,16p'
> @printf '\n===== UTILITIES =====\n'
> @gcc --version | head -n 1
> @jq --version

tree:
> @find . -maxdepth 3 -path './.git' -prune -o -print | sort

status:
> @git status --short

fmt-check:
> @unformatted="$$(gofmt -l $$(find . -type f -name '*.go' -not -path './vendor/*'))"; \
> if test -n "$$unformatted"; then \
>   printf '%s\n' "$$unformatted"; \
>   exit 1; \
> fi

test:
> @$(GO) test ./...

test-race:
> @CGO_ENABLED=1 $(GO) test -race ./...

vet:
> @$(GO) vet ./...

check: fmt-check vet test

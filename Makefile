.RECIPEPREFIX := >
SHELL := /usr/bin/env bash

GO ?= go
PYTHON ?= python3.12

.PHONY: verify-tools tree status

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

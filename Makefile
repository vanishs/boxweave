.PHONY: fmt fmt-check lint test test-race verify hooks

GO ?= go
PYTHON ?= python3

fmt:
	$(GO) fmt ./...

fmt-check:
	$(PYTHON) scripts/check_format.py
	$(GO) run scripts/check_yaml.go
	$(GO) mod tidy -diff

lint: fmt-check
	$(GO) vet ./...
	$(GO) mod verify

test:
	$(GO) test -count=1 ./...

test-race:
	CGO_ENABLED=1 $(GO) test -race -count=1 ./...

verify: lint test-race

hooks:
	sh scripts/install-hooks.sh

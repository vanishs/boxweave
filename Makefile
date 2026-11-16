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
	$(PYTHON) -m unittest discover -s scripts -p 'test_s4_raw_operation_witness.py'

hooks:
	sh scripts/install-hooks.sh

.PHONY: fmt fmt-check lint test test-race test-mcp-python test-uninstall test-stable-gate verify hooks

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

test-mcp-python:
	$(PYTHON) -m unittest discover -s scripts -p 'test_mcp_testkit.py' -q

test-uninstall:
	$(PYTHON) -m unittest discover -s scripts -p 'test_uninstall_core.py' -q

test-stable-gate:
	$(PYTHON) -m unittest discover -s scripts -p 'test_v1_stable_gate.py' -q

verify: lint test-race test-mcp-python test-uninstall test-stable-gate
	$(PYTHON) -m unittest discover -s scripts -p 'test_s4_raw_operation_witness.py'
	$(PYTHON) -m unittest discover -s scripts -p 'test_pr177_installed_recovery.py' -q

hooks:
	sh scripts/install-hooks.sh

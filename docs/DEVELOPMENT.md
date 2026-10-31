# Development and testing

This guide is for synthetic local development and does not validate a real Provider, external SSH access, network isolation, or model inference.

For Agent-driven iterations on existing or truly clean nested test Boxes, see the [Agent development/test workflow](process/AGENT_TEST_WORKFLOW.md). It covers auto-within-authorization and plan-first modes, evidence, failure preservation and escalation from fixtures to approved live validation. This fixture guide does not grant permission to modify a live environment.

## Isolated fixture

This is the local developer/demo and regression-test mode, **not a real installation**. It exercises the Web UI, project binding, and operation receipts using synthetic infrastructure without contacting or changing a real Incus/Containarium Provider.

On a prepared non-root Ubuntu 24.04 x86_64 machine with Python 3.12, Node/npm, browser test dependencies if needed, and no occupied port 18640:

```sh
python3 -m venv .venv
.venv/bin/pip install --require-hashes -r requirements-dev.lock
(cd web && npm ci --ignore-scripts --no-audit --no-fund && npm run build)
.venv/bin/python -m bwmini init-admin --config examples/fixture.json
.venv/bin/python -m bwmini worker --config examples/fixture.json
# In another terminal:
.venv/bin/python -m bwmini serve --config examples/fixture.json
```

Open `http://127.0.0.1:18640`, sign in using the administrator password entered locally, save a project binding and inspect a receipt. This is a **synthetic fixture**: working UI and simulated activity do not prove real Provider access or isolation. Processes are foreground-only and bind loopback. See [verification strategy](verification/TEST_STRATEGY.md).

## Routine checks

```sh
python scripts/check_knowledge.py
.venv/bin/python scripts/validate.py
.venv/bin/python scripts/validate.py --browser
```

The browser checks are fixture-only. The runtime lane requires an approved prepared non-root Linux environment. Record any additional real infrastructure validation separately in [the test queue](verification/REMEDIATION_TEST_QUEUE.md).

After an existing build, `scripts/validate.py --browser-only` runs just the seven fixture browser workflows with the same shared 600-second deadline and private reports. GitHub CI now includes this lane and publishes selected fixture logs/results/screenshots, plus a separately built application candidate; a configured workflow is not evidence of a successful run.

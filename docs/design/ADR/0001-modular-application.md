# ADR-001 — One application with explicit adapters

Status: selected for phase 4 validation, 2026-10-28. Inputs: SRC-026, REQ-001, REQ-008, REQ-010, REQ-011, REQ-014.

Validated selected boundaries through0.9.0/SRC-080..083: shared business services survive API/worker restart; Web-down independent MCP read passed; unprivileged staged app distribution preserves independent credential/state planes. User units run only the app API/worker. An external approved TLS frontend remains separate; installer never publishes a new listener beyond loopback. Current user-manager service evidence does not imply whole-host/LXC reboot readiness. See [release evidence](../../verification/TASK_012_013_RELEASE.md).

Problem: approved basic Web and prompt graph require UI, persistence and operations without two business-state implementations.

Options: Python/Flask/Waitress + SQLite + React/TypeScript; Go API + React; Node API/UI; distributed services. Select the Python modular application with a separate mutation worker, CLI/Web adapters and a fixed Containarium boundary. Python suits small process/config tools; React/XYFlow is justified by the included graph. Go would simplify some deployment, Node some UI sharing, but neither eliminates platform-specific enrollment. Distributed services add failure/deployment surfaces unsupported by single-operator demand.

Consequences: keep business services framework-independent; no huge app.py; use injected provider/process/clock seams. Web failure must not stop upstream SSH MCP. No direct reuse of legacy implementation. Python runtime and frontend build are deployment costs, handled by staged packaging. No framework version is silently pinned to an old vulnerable version.

Evidence: Flask documents Waitress as a single multithreaded process with buffered requests, so use bounded JSON and polling, not a presumed streaming server ([official docs](https://flask.palletsprojects.com/en/stable/deploying/waitress/), SRC-028).

Original phase3 validation plan: TASK-001, TASK-002, TASK-004 and TASK-005 run the minimal Web->binding->real MCP receipt slice and inspect shutdown/error isolation. If framework coupling prevents deterministic tests or HTTP threads block on provider phases, revise the ADR. Actual selected runtime results are now linked above; no performance benchmark is claimed.

# ADR-007 — Distinct host bootstrap and application release boundaries

Status: accepted **design direction**, implementation and real-host acceptance incomplete. Source: SRC-093 (owner 2026-10-29).

## Context
Legacy 0.5.1 combined root-level Incus/Containarium/SSH/host key/token/Web installation. New v2 distribution safely manages only a non-root, digest-verified app with an authorized independent MCP launcher. The owner expressly requires the useful integrated first-host installation experience back, without automatically migrating old data.

## Decision
Use two explicit execution boundaries: (1) separately approved, **pinned and locally reviewed** upstream root-level Provider/Incus provisioning, permitted only on an empty, inspected host; (2) app-user v2 distribution install/update/repair preserving backed-up database, exact config and service ownership. No implicit root download-execute, auto-Provider upgrade, SSH trust rewrite, credential transfer, TLS/public exposure or claim of host readiness from a plan. Keep a separate read-only management CLI.

The source `install.sh` implements a constrained coordinator, not the missing full end-to-end bootstrap: it currently requires an app user, systemd user bus, reviewed release archive/digest and separately prepared application MCP credentials/launcher. A complete reproducible fresh-host experience needs verified upstream contracts, authorized credential creation, bootstrap owner integration, rollback and reboot tests before acceptance.

## Consequences and verification
Operators must explicitly prepare privileged artifacts and authorization and retain previous host state; partial failures are not whole-host rollback. Tests need malformed inputs, existing-host refusal, isolation, first administrator, true MCP read, re-run/repair, reboot, file ownership, TLS/SSH responsibility and confirmation the old 0.5.1 services remain untouched. See [host guide](../../HOST_BOOTSTRAP.md), [R-02 queue](../../verification/REMEDIATION_TEST_QUEUE.md), and [risks](../../verification/OPEN_RISKS.md).

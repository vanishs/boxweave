# S2.5 incident: Exec UNKNOWN and nested Incus AppArmor denial

**Observed:** 2026-11-14 UTC, while preparing a disposable Ubuntu 24.04 Box under L1 RC4. Scope is newly created Boxes only. No historical UNKNOWN Box was retried, unlocked, or deleted. No BoxWeave controller was installed in the final disposable Box; Cloudflare Tunnel and OAuth setup were not reached.

## Findings

### Exec UNKNOWN and missing executable

An earlier L1 RC4 box_exec operation ran the shell pipeline curl -fsSL https://pkgs.zabbly.com/key.asc | gpg --show-keys --fingerprint --with-colons and ended UNKNOWN / POSTCONDITION_UNVERIFIED, with no usable saved output receipt. BoxWeave Operation ID: e38022d284ee5ecde41b9a040bb856f6; native Incus Operation ID: 9b606584-16c1-455a-b65e-5ae8a6376589; affected Box UUID: 44211d73-d9c3-4fed-acfa-1b4a1f94207a. Do not retry or relabel this historical operation.

In a fresh Ubuntu 24.04 minimal Box, command inventory showed gpg was absent. Installing the Ubuntu package gpg (2.4.4-2ubuntu17.6) made the same key inspection possible; it returned exit 0 and fingerprint 4EFC590696CB15B87C73A3AD82CC8797C838DCFD. This is strong evidence that a missing executable is a concrete bootstrap hazard. It does not prove whether the earlier UNKNOWN native operation dispatched, or that missing gpg alone caused that exact persisted UNKNOWN: the old operation has no usable output receipt and remains unresolved.

A separate bundled bootstrap attempt also ended UNKNOWN (BoxWeave operation d1e0f3c2b36a47391cd7781e00c3e1e1, native operation ba486088-4b10-4a45-a2d4-0ec91a6567a8, Box 150510e3-ebf3-415a-adba-63c310a44364), with no saved output. It bundled steps and is not suitable for localizing a single failing command.

**S2.5 hypothesis to test:** a missing executable detected before native Exec dispatch must produce a definite launch failure with a diagnostic, not an ambiguous postcondition UNKNOWN or a held resource claim. Preserve UNKNOWN whenever dispatch/effect actually may have occurred. Compare absent executable with an existing executable that exits nonzero, and retain original BoxWeave/native IDs and stdout/stderr. Never replay either historical operation.

### Inner Incus blocked by outer AppArmor policy

On a newly created running L1 child Box bw-rc4-stepwise-20261114-7b61, UUID 70657927-03d0-4af0-95f1-b555c77cf7a4, nesting=true, Ubuntu 24.04, installing Incus 7.6 succeeded. Package setup repeatedly emitted AppArmor profile replacement denials. The dedicated boxweave user (UID 1001, member of incus) used the unprivileged per-user Incus socket /var/lib/incus/unix.socket.user; first-use user Project/bridge initialization then timed out. The incus-user.service restart loop journal recorded:

    Failed to create network
    apparmor_parser -rWL /var/lib/incus/security/apparmor/cache /var/lib/incus/security/apparmor/profiles/incus_dnsmasq-incusbr0
    exit status 243
    Unable to replace "incus_dnsmasq-incusbr0_</var/lib/incus>".
    Access denied. You need policy admin privileges to manage profiles.

Admin-level incus project list remained only default; no restricted per-user Project was established. The per-user service was stopped to prevent a restart loop; the parent Incus service remained active. No AppArmor protection was disabled and no privileged Incus socket was substituted. This is a concrete host-policy block in this L1 environment, not proof all nested Incus deployments are impossible. The next investigation is whether an approved outer-host profile can safely support Incus profile loading and nested networking, or whether this environment must report nesting BLOCKED.

The official BoxWeave v1.0.0-rc.4 installer was downloaded and checksum-verified and its --plan was inspected. Apply was not run because required user Project/socket prerequisites could not be established. BoxWeave build-info, doctor, systemd, local MCP, tunnel and OAuth therefore remain unverified.

## S2.5 acceptance additions

See BH-018 and BH-019 in BRIDGEHEAD_ACCEPTANCE.md. They require one-command-at-a-time reproduction, exact operation evidence, honest pre-dispatch vs post-dispatch classification, preservation of UNKNOWN, AppArmor-enforced nested Incus readiness, and an explicit BLOCKED result when safe policy prerequisites are unavailable. These cases supplement BH-001–BH-017; they do not convert prior results to PASS or authorize physical-host changes.

## Next-Agent handoff prompt

> Work on BoxWeave v1 and read AGENTS.md, docs/development/EXECUTION_LOG.md#current-checkpoint, docs/development/phases/025_BRIDGEHEAD.md, docs/verification/BRIDGEHEAD_ACCEPTANCE.md, and docs/process/UNKNOWN_MANUAL_RECOVERY.md. Reproduce and resolve BH-018/BH-019 in an authorized disposable environment, then update evidence and S2.5 status.
>
> Use only the explicitly selected L1 RC4 MCP to create a fresh, uniquely named disposable Ubuntu 24.04 Box with nesting and internet enabled. Do not touch, replay, unlock, delete, or relabel historical UNKNOWN operations or existing Boxes. Capture each new Operation ID, native Incus Operation ID, Box UUID, raw state, output receipt and timestamp.
>
> Work in small stages. Before each command, inspect whether it exists (for example, command -v gpg) and whether inputs exist; execute one logical command at a time and record exit/output. Reproduce absent-executable behavior and compare it with an installed executable returning a nonzero guest exit. Determine whether absence is rejected before native dispatch. Fix RC4 classification/receipt handling so provably pre-dispatch launch failure is definite and diagnostic while uncertain dispatched effects remain UNKNOWN and are never retried. Add focused regression tests and retain exact original IDs.
>
> In the disposable environment, install supported Incus packages from the official repository with signature/checksum validation. Inspect the service identity, non-admin Incus group membership, unprivileged socket, daemon/project state, AppArmor status and exact incus-user.service journal. Reproduce apparmor_parser exit 243. Do not disable AppArmor, loosen global profiles, use the privileged Incus socket, fabricate a permissive Project, or mutate the physical host/other Boxes. Use L0 only for authorized read-only inspection of relevant outer policy unless the owner has approved a narrowly scoped policy change. If secure nested Incus prerequisites cannot be established, report BLOCKED and document exact prerequisites; do not claim nested PASS.
>
> Apply BoxWeave only after restricted Project and service socket prerequisites pass. Pin v1.0.0-rc.4 (or specifically approved candidate), verify official checksums, then validate build-info, doctor, systemd, local /mcp, project isolation and an actual next-level child lifecycle before declaring nesting PASS. Do not create deeper Boxes beyond the single acceptance child. Cloudflare and OAuth setup are out of scope unless separately requested.
>
> Update BH-018/BH-019 with exact source/binary SHA, environment, commands, outputs, native identities, failures, rollback and verifier. Run repository-required checks, preserve append-only history, and open a Draft PR against v1. Report PASS/PARTIAL/BLOCKED accurately; an installer plan or package installation alone is not success.

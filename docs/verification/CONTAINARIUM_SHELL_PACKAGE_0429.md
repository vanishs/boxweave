# Containarium host-hop wrapper and bounded repair — source 0.42.9

## Root cause and independent proof

`debug_container` previously diagnosed an existing backend host
user whose login shell was `/usr/local/bin/containarium-shell`
but the executable was absent. That prevents a host-hop SSH login.
Direct BoxWeave `connect(exec)` can still use managed guest IP
SSH; it is a distinct route.

The owner has now patched `dev` themselves. A fresh read-only
`debug_container(username="bwphase2-1102")` reports the host
login shell exists on Containarium v0.120.1. The backend advertises
no external SSH ingress. No actual host-hop login was exercised by
this development task.

## What the source change implements

- Package the exact official Containarium v0.120.1 wrapper (106 lines,
  identical to upstream `main@4be7f81a`) in
  `bwmini/containarium-shell`. Independent expected SHA-256:
  `2212651e2083b80d76363523bf794ec4d530f8eb74bbac0def0c347bb544caa6`.
- Include the wrapper in the source-commit-bound release manifest,
  standalone helper publication and installer downloads.
- For **BoxWeave-owned local Providers**, install only if absent and
  register `/usr/local/bin/containarium-shell` in `/etc/shells`
  when needed. Verify pinned digest, syntax, safe directory/registry,
  root ownership and file permission; refuse modified foreign or
  symlinked scripts. Record only a script actually installed by
  BoxWeave as a managed file.
- For **externally managed Providers**, observe without changes by
  default; opt-in to this narrow repair explicitly with
  `--repair-containarium-shell` on a published v0.42.9+ release.
- Do NOT invoke the complete upstream host-hop setup script: no
  global `ALL ALL NOPASSWD: incus` sudoers, system sshd, MOTD,
  host accounts, keys, routing, Provider or Incus version mutations.
- Record `host_hop_ssh_verified=false` in installer output. The
  stable version pin is intentionally unchanged.

## Executed source checks (isolated `bwphase2-1102` dev Box)

- `python3 -m unittest discover -s tests -p test_containarium_shell.py`:
  **14 PASS** (non-root mocked host ownership, no host writes).
- `python3 -m unittest discover -s tests -p test_online.py`:
  **18 PASS** (streamed-manifest, enrollment and SSH contract tests).
- `python3 -m unittest discover -s tests -p test_online_adoption.py`:
  **7 PASS** (existing Provider adoption boundaries).
- `python3 -m unittest discover -s tests -p test_arch0_boundary_guards.py -k root_helper`:
  **1 PASS**, checks release/public-helper source parity.
- `python3 scripts/check_knowledge.py`: **175 documents PASS**;
  `bash -n install.sh`, `python3 -m py_compile` and `git diff --check`: PASS.
- Complete standalone application import suite cannot be asserted from
  this bare dev Box: its Python environment lacks `werkzeug` and has no
  prepared venv/npm. CI must run the fully provisioned test lane.

## Tests and acceptance

Unit tests target pinned content and parser validity, missing file,
existing identical file, idempotence, explicit repair, foreign script,
unsafe registry, symlink and checksum refusal. The installer and
release inventories are tested independently.
Follow [runbook](../process/CONTAINARIUM_SHELL_RECOVERY.md)
for a separately authorized host-hop positive/negative test.

**Not proven:** a fresh privileged host install; end-to-end host-hop
SSH, sshpiper, actual per-account sudo policy, interactive terminal
and file-transfer interoperability, and cross-tenant denial. Direct
guest SSH is not host-hop evidence. No owner host file was written
by this source-development task. After publication, GitHub CI is
the source/fixture validation gate, not live infrastructure proof.

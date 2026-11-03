# Containarium shell real SSH hop and wake — 2026-11-03

**Status: LIVE PASS** for authenticated SSH command forwarding, wake from STOPPED, and unauthorized-key denial in an isolated nested test Box. **SECURITY FOLLOW-UP OPEN** for broad Incus sudo privileges. This is not an external client, sshpiper or cross-tenant certification.

[Upstream/BoxWeave decision](../design/CONTAINARIUM_UPSTREAM_MCP_FOLLOWUPS_2026-11-03.md) · [v0.42.10 release test](CONTAINARIUM_SHELL_PINNED_FETCH_04210_2026-11-03.md) · [Risk register](OPEN_RISKS.md)

## Exact environment

- Created a **new**, separate outer Box: `bwhop-1103-e2e-container`, 2 CPU, 4 GB RAM, 35 GB disk, own network Tenant. Applied nesting to that Box, restarted only that Box.
- Installed the published BoxWeave `v0.42.10` on it with the exact-version installer. State became `ready`. Local Containarium binaries stayed at `v0.120.1`.
- Installer output: `containarium_shell.status=installed_from_pinned_upstream_registered`, no error. The installed `/usr/local/bin/containarium-shell` SHA-256 was `2212651e2083b80d76363523bf794ec4d530f8eb74bbac0def0c347bb544caa6`, corresponding to upstream pinned commit `e5deb167c973033d9a847abdcb2420e917095546`.
- Created a **real inner Box**: `sudo containariumd create hopprobe --no-ssh-key --podman=false --cpu 1 --memory 1GB --disk 8GB`. Instance: `hopprobe-container`, RUNNING; guest account `hopprobe` existed. The local daemon created host account `hopprobe` with shell `/usr/local/bin/containarium-shell`. The plain `containarium create` command was client-only without a server option, so the host-local test used `containariumd`.
- Added a dedicated short-lived Ed25519 *public key* to the isolated host account, using a private test key held only on this test Box. No credential bytes were supplied to the outer host or GitHub.

## Actual TCP SSH tests

| Case | Observable evidence | Result |
| --- | --- | --- |
| Host loopback SSH forwarding | `ssh -T ... hopprobe@127.0.0.1 'printf "SSH_HOP_OK\n"; id -un; hostname; pwd'` returned **exit 0** and lines `SSH_HOP_OK`, `hopprobe`, `hopprobe-container`, `/home/hopprobe` | **PASS**: real SSH authentication → host login shell → Incus exec → inner guest |
| SSH to the outer Box's assigned IP | From that same outer Box: `ssh -T ... hopprobe@10.100.0.13 'printf "IP_SSH_HOP_OK\n"; hostname'` returned **exit 0**, output `IP_SSH_HOP_OK` and `hopprobe-container` | **PASS**: network-interface SSH, *not* separate remote-machine ingress |
| SSH wake from STOPPED | `sudo incus stop hopprobe-container`, observed `Status: STOPPED`; set test `user.containarium.stopped_at`; repeated the SSH command | **PASS**: exit 0; stderr `Waking hopprobe-container (was: STOPPED)...`; stdout `WAKE_SSH_OK`, `hopprobe`, `hopprobe-container`; afterward `Status: RUNNING` |
| Wake bookkeeping | Observed `user.containarium.last_started_at=2026-11-03T18:34:40Z` and empty `user.containarium.stopped_at` after wake | **PASS** |
| Unauthorized SSH key | A separate unapproved private key used with `IdentitiesOnly=yes` and `BatchMode=yes` | **PASS negative**: SSH exited **255**, `Permission denied (publickey)` |
| SSH daemon | `sudo sshd -t` returned exit 0 and SSH service stayed active | **PASS** |

These are actual *sshd* sessions, not a direct invocation of `containarium-shell` or a simulated command result. The real SSH client ran inside the outer Box, so **public/external SSH ingress and sshpiper are not validated**.

## Security concern: broad host Incus sudo authority

The Containarium v0.120.1 host-account creation path wrote this **per-account sudoers file** in the disposable outer Box:

- `/etc/sudoers.d/containarium-hopprobe`: `hopprobe ALL=(root) NOPASSWD: /usr/bin/incus`
- `sudo -l -U hopprobe` confirmed passwordless root execution of the Incus binary without a per-container argument restriction.
- `sudo -u hopprobe sudo -n /usr/bin/incus list --format csv -c n,s` succeeded.

This exposes a **broad host Incus command capability** and needs an upstream account/authorization review before declaring tenant-isolated host-hop access. We did **not** demonstrate a cross-tenant exploit, create a second guest or bypass the host login shell. A normal successful SSH forward does **not** prove that all alternate execution paths are denied. Do not respond by adding global Incus sudoers in BoxWeave. Record scoped-broker / least-privilege and audit requirements separately.

## Separate interactive PTY coverage — not accepted

After the successful non-interactive SSH tests and credential cleanup, a follow-up interactive PTY check was considered. Its prerequisite of re-authorizing a new short-lived test SSH credential was **blocked by the active tool safety check** before a PTY SSH session was initiated. No bypass was attempted and **no interactive PTY result is claimed**. The outer test Box was stopped again. Interactive login remains **U**, separate from the PASS evidence for real SSH command forwarding and wake.

## Cleanup and limits

- After the live tests, the injected `authorized_keys` was removed after matching it to the temporary test public key; both test key files and known-host test files were removed. The disposable outer Box `bwhop-1103-e2e-container` was gracefully **stopped**, not destroyed, preserving the nested filesystem for authorized repro.
- **Closed in this isolated topology:** host SSH login, regular command forwarding, host-interface access, stopped Box wake, lifecycle markers, unauthorized-key denial.
- **Still unverified:** separate-machine SSH ingress, sshpiper, meaningful cross-tenant/cross-Box denial, alternative host execution paths, Containarium versions newer than `0.120.1`, default stable-install pointer promotion.
- No BoxWeave source/runtime, outer/owner host Containarium binary, stable installer pin or production SSH configuration was changed by this test.

[Back to owner decision](../design/CONTAINARIUM_UPSTREAM_MCP_FOLLOWUPS_2026-11-03.md)

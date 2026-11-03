# Recover missing Containarium host-hop login wrapper (operator runbook)

[Agent rules](../../AGENTS.md) · [Open risks](../verification/OPEN_RISKS.md) · [issue #19](https://github.com/vanishs/boxweave/issues/19)

**Status: HOST-HOP NOT REPAIRED on the owner's `dev` backend.** These are reviewed instructions, not evidence of a production repair. No real host/sshd/sudoers file was changed. Follow explicit, host-specific operator authorization; preserve console/admin access and obtain a rollback window before any live writes.

## Diagnosis: not all SSH is broken

Read-only `debug_container` confirmed on Provider **v0.120.1**, backend `dev`, that the account corresponding to `containarium-prqa-1102` (also `bwphase2-1102`) already exists and has login shell `/usr/local/bin/containarium-shell`, but **hostUserShellExists=false**. The reported shell is a **host-side**, not a guest-side, executable: it forwards *host-hop* SSH sessions through Incus to the guest.

**Independent live countercheck:** BoxWeave MCP `connect(box="containarium-prqa-1102", exec="id -un && python3 -c 'print(\\"hello world\\")' && hostname")` succeeded with exact user name, Python stdout `hello world`, guest hostname and exit **0**. The same direct `connect(exec)` succeeded for `bwphase2-1102`. Thus this is a **host-hop-only missing component**; direct managed guest SSH continues to work. Agents MUST try identity-pinned `connect(exec)` before classifying an entire Box as SSH BLOCKED; do not create replacement Boxes or install the host wrapper inside a guest.

## Why a new Box does not repair it

The current BoxWeave local-provider bootstrap (`bwmini/online.py`) installs a pinned `containariumd` executable and starts its service, but **does not configure the Containarium host-side SSH proxy wrapper**. Upstream `scripts/setup-ssh-container-proxy.sh` contains that wrapper, but running the **entire** script also writes a broad `ALL ALL ... NOPASSWD: incus ...` sudoers rule and global sshd/MOTD settings. **Do not run the whole script automatically or as a generic repair.** A later Provider binary upgrade, new Box, or a direct guest `connect` is not an installation of the host-side component.

## Minimal operator-only repair (on `dev` host, NEVER inside a Box)

**Read-only preflight (operator shell on `dev`):**

```bash
getent passwd containarium-prqa-1102
test -x /usr/local/bin/containarium-shell && echo PRESENT || echo MISSING
command -v incus
sudo -l -U containarium-prqa-1102  # inspect existing rights; do not paste secrets
sudo /usr/sbin/sshd -t              # preexisting SSH configuration must already parse
```

If and only if the operator needs host-hop SSH, confirms this exact missing-file condition and approves a targeted fix, retrieve the *pinned official upstream* wrapper. The following **only extracts the body of the reviewed upstream heredoc to a nonprivileged /tmp file**; it does **not** run the upstream installer:

```bash
curl -fsSLo /tmp/containarium-ssh-proxy-source.sh \
  https://raw.githubusercontent.com/FootprintAI/Containarium/4be7f81aa0dc98bb1dd6ad17b3fba2225993e7ba/scripts/setup-ssh-container-proxy.sh

awk '/^cat > "\$WRAPPER_SCRIPT" << '\''SHELLEOF'\''$/ {on=1; next} on && /^SHELLEOF$/ {exit} on {print}' \
  /tmp/containarium-ssh-proxy-source.sh > /tmp/containarium-shell

printf '%s  %s\n' \
  '2212651e2083b80d76363523bf794ec4d530f8eb74bbac0def0c347bb544caa6' \
  '/tmp/containarium-shell' | sha256sum -c -
bash -n /tmp/containarium-shell
```

The extraction and syntax command were **actually executed successfully in a non-host test Box** against the pinned source on 2026-11-02. The hash is for exactly that **106-line extracted wrapper**, not the entire upstream installer. Inspect the file and the existing per-user sudoers constraints before installing; if a wrapper already exists or differs, **STOP** and reconcile ownership instead of overwriting.

**Approved host write, if preflights pass:**

```bash
sudo install -o root -g root -m 0755 /tmp/containarium-shell \
  /usr/local/bin/containarium-shell
grep -Fxq /usr/local/bin/containarium-shell /etc/shells || \
  printf '%s\n' /usr/local/bin/containarium-shell | sudo tee -a /etc/shells >/dev/null
sudo /usr/sbin/sshd -t
```

This writes only the root-owned login shell and (if absent) an `/etc/shells` entry. It does **not** touch sshd configuration, install wildcard Incus sudoers, change host users, restart daemon/sshd or set up any public ingress. On default Ubuntu sshd, the next login reads the executable shell without an sshd reload. The wrapper calls `sudo incus info/start/exec` and `incus config` in its supported paths, so **existing per-account sudo authorization may still require separate privileged review**; a missing permit is not permission to install a global `ALL ALL` rule. In particular, deny cross-tenant Incus privileges and review root delegation carefully before allowing host-hop logins. Do not invent a test success based on `sshd -t` alone.

**Post-repair acceptance:** `debug_container` must show `hostUserShellExists=true` on the *same exact Box* and backend. Separately test the intended host-hop transport with a known, trusted SSH host key and an existing authorized identity, verify actual guest user/hostname, `python3 -c 'print("hello world")'` stdout and exit 0, and verify that a forbidden identity cannot traverse to another Box. A successful direct managed Box `connect(exec)` is **not** proof the host-hop repair succeeded. Do not alter client SSH pins, authorized keys, user shells or host firewall to make an unrelated check pass.

## Follow-up product fix (not implemented here)

- Add a **read-only provider-host SSH preflight** to fresh local-provider bootstrap and diagnostics. Report the absent wrapper as *HOST_HOP_SHELL_MISSING*, explicitly distinguish direct managed guest SSH, and show operator-only remediation. Do not silently add broad sudoers, install a shell or mutate a reused/externally managed Provider.
- If enabling host-hop is a selected deployment feature, add an **explicit, reviewable, narrowly scoped provider SSH installation mode**, with pinned wrapper content, file ownership checks, existing-user migration, least-privilege Incus authorization and positive/negative host-hop integration tests. Keep the default direct-MCP route working without the wrapper.
- Until those features ship, issue #19 remains **OPEN for host-hop**; direct guest MCP access is verified separately.

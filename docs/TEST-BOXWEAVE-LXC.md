# Ubuntu LXC installation test: 2026-10-22

The outer BoxWeave Platform MCP created the logical box `boxweave-test` as
`boxweave-test-container`, using Ubuntu 24.04.5 LTS, 2 CPUs, 4 GB RAM and a
40 GB disk. Its observed internal address was `10.206.30.5`; no additional port
route was created. The creation request timed out, but a subsequent inventory
query confirmed the instance was Running. It was not created a second time.
MCP execution confirmed the account and hostname before installation.

## Installation and first-run fix

The installer and components were fetched from commit
`9956e81e884a0493771eff2caae1aef3383a0426`. Installation ran with a 600-second
outer timeout and `--api-ready-timeout 600`. The first attempt took 494.3 seconds
and exited 1 after the provider API became reachable with HTTP 401. It failed at
`systemctl reset-failed`: newly introduced inactive oneshot units were not loaded
in systemd and reset-failed rejected them. The managed timers remained stopped.

The installer was changed to call reset-failed only when `systemctl is-failed`
reports an actual failure. The test's fake systemctl now rejects resetting an
unloaded service, reproducing the behavior that the earlier mock missed.

The patched installer was copied into the same test box and rerun, keeping the
other component files pinned to the original commit. Its SHA256 was
`c23013895e2dee22df010704a5d7eef3080bfc4d5521589969e601f2e4341f4d`.
The rerun finished in 7.6 seconds with exit 0 and removed the pending-update
marker. This is a successful test of the patched installer; the original commit
alone did not pass a fresh installation.

## Final command results

Commands were executed through MCP with administrative privileges because the
configuration file is root-readable:

| Command | Exit | Observed result |
| --- | --- | --- |
| `sudo boxweave status` | 0 | Incus and Containarium active; both managed timers active/waiting |
| `sudo boxweave doctor` | 0 | Configuration available; LXC environment detected; component versions available |
| `sudo boxweave doctor --api --json` | 0 | API reachable, HTTP 401; authenticated health explicitly not verified |

Versions observed: BoxWeave `0.1.0-dev`, Incus client `7.5.1`, Containarium CLI
`0.114.2`. The Containarium service, Incus socket and both managed timers were
enabled. Token and host-key oneshots each completed with Result=success and
ExecMainStatus=0; being inactive afterward is normal for these oneshot units.
No reboot was performed, and LXC autostart was not verified.

The doctor reported an `unconfined` process label from inside the LXC and left
effective device filtering unverified. This test does not certify host-side
confinement. Authenticated nested MCP execution was tested afterward as below.

## Nested MCP hello-world test

At 2026-10-22T21:55:53Z, the outer Platform MCP connected to `boxweave-test`.
A Python stdio client there launched the installed `/usr/local/bin/boxweave-mcp`,
completed JSON-RPC initialize (protocol `2024-11-05`), sent the initialized
notification and discovered tools. The inner server identified itself as
`containarium-mcp-server` version `0.114.2`.

Through that inner MCP session, the test listed containers, created the logical
box `hello-world`, queried it, and executed commands using `connect`. Creation
used an explicit verified public key belonging to this nested host; no private
key or JWT contents were read or returned by the test client. Creation completed
in 60.5 seconds. The instance was `hello-world-container`, Running at the observed
internal address `10.223.4.2`, with 1 CPU, 1 GB RAM and an 8 GB disk.

The inner MCP `connect` first returned `hello-world` and `hello-world-container`
for `whoami && hostname` with exit code 0. A subsequent MCP connect executed:

```bash
python3 -c "print('hello world')"
```

The returned stdout was `hello world` and exit code was 0. This verifies the
chain: outer MCP -> BoxWeave LXC -> inner MCP -> nested workload LXC -> Python.
It did not substitute direct Incus execution for the workload MCP call. The
workload container remains available; no external port was exposed.

Protocol discovery, create results, container metadata, identity verification and
the final execution result remain under
`/home/boxweave-test/boxweave-nested-mcp-0tv0u4zj/` in the outer test container.
A local diagnostic copy is
`outputs/boxweave-test-20261022/nested-hello-world-result.json`.

## Evidence

The following files remain in the test container:

```text
/home/boxweave-test/boxweave-install-test-bm1nc2kn/install.log
/home/boxweave-test/boxweave-install-test-bm1nc2kn/result.json
/home/boxweave-test/boxweave-install-test-bm1nc2kn/install-fixed.sh
/home/boxweave-test/boxweave-install-test-bm1nc2kn/install-fixed.log
/home/boxweave-test/boxweave-install-test-bm1nc2kn/result-fixed.json
/home/boxweave-test/boxweave-install-test-bm1nc2kn/diagnostics-final.json
```

Full installer logs should not be published without redaction. Only selected
phase messages and the CLI's filtered diagnostic output were returned to chat.
The local diagnostic copy is under the ignored `outputs/boxweave-test-20261022/`
directory. The original research containers and host services were not modified.

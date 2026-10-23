# Tasks, receipts and the nested smoke test

These commands run on the machine where the CLI is invoked. Run them inside the
assigned remote LXC through the existing Platform MCP when doing remote work.
They do not add a gateway, expose a network port or register a per-box MCP server.
Python's standard library is sufficient; execution and stdio tests require POSIX.

## Durable local tasks

The installed CLI provides:

```bash
boxweave job start --timeout 600 -- python3 -c "print('hello world')"
boxweave job status JOB_ID
boxweave job status JOB_ID --logs
```

From a source checkout, use `python3 server/jobs.py` with the same subcommands.
`start` returns a task ID after launching a detached worker. State lives under
the invoking user's `~/.local/state/boxweave/jobs/`, with private directories and
files. `--state-dir PATH` selects another private directory, and must also be
supplied when querying that task. `--output-limit` sets the shared stdout/stderr
budget (default 1 MiB, maximum 16 MiB). The command is an argv list, not shell
text; use a shell explicitly only for a reviewed command that requires it.

Receipts record the executing host/UID, queued/running/succeeded/failed/timed_out/
output_limit/launch_failed states, timestamps, duration and the command's actual exit code. A vanished
worker with an unfinished receipt is reported as unknown, with exit_code=null;
a lock avoids confusing a recycled PID with the worker. A brief queued startup
grace period is allowed. Querying a receipt does not rerun the command. A worker
lost to reboot/crash requires a separately considered new task, not automatic
replay. Query on the same host and as the same user that submitted it.

The start/status CLI exit code describes submission/query success; inspect the
receipt's state and exit_code for the command result. Logs are bounded, and
recognized JWT, credential-assignment and PEM private-key lines are redacted.
Arbitrary application output cannot be guaranteed secret-free. Do not pass
credentials in command arguments or use this tool to collect secrets. Credential-
like arguments are rejected; protected file interfaces belong to the application.
The private request file contains the command argv, while the public-facing
receipt records only its executable, argument count and a digest.

The timeout covers the worker's local process group. It does not prove that a
remote SSH operation, daemon request or detached descendant outside that group
was cancelled, on this host or another. After a transport interruption, query both the receipt and the actual
resource before retrying a state-changing operation.

## Plan an inner MCP smoke test

The script runs **inside an existing BoxWeave host LXC**, as the configured
non-root user. Its purpose is to validate that host's installed inner MCP. It
does not install BoxWeave, create the parent LXC or need a physical host socket.
First inspect the environment and show the proposed test to the user as required
by AGENTS.md. Plan generation itself checks local launcher/public-key files only;
it does not start MCP or contact the provider.

```bash
python3 scripts/nested_smoke.py plan \
  --box nested-smoke \
  --public-key "$HOME/.ssh/containarium_ed25519.pub" \
  --directory "$HOME/boxweave-smoke-run"
```

The installed equivalent is `boxweave selftest plan` with the same flags. Review
the emitted JSON with the user: it fixes the current host, launcher hash, public
key fingerprint, exact logical target, Ubuntu image, 1 CPU/1 GB/8 GB resource
request, phase deadlines and cleanup policy. No private key/JWT is read by this
test client; the managed launcher/backend uses its normal protected credential
interface. Use the actual managed public key for this nesting layer, not the
outer host or Windows transport identity.

After the user confirms the plan, run using its exact printed approval digest:

```bash
python3 scripts/nested_smoke.py run "$HOME/boxweave-smoke-run/plan.json" \
  --approve APPROVAL_SHA256
python3 scripts/nested_smoke.py status "$HOME/boxweave-smoke-run/plan.json"
```

`--approve` records the operator's confirmation of that plan; it is not a
cryptographic authorization token. Editing a plan or changing the host, launcher
or key requires generating/reviewing a new plan. The provider's authorization
remains authoritative. `--reuse-existing` must be included when generating a plan
that permits operating in a preexisting workload. Otherwise existing targets are
rejected, with no workload commands executed.

The client performs initialize/initialized and tools/list, then tools/call for
inventory, creation, inspection and connect. It verifies `whoami && hostname`
before running the fixed `python3 -c "print('hello world')"` command. The result
must have exit 0 and exactly the expected stdout. Its wire parser is deliberately
limited to these fixed commands and the tested provider text format; it is not
a generic trusted parser for arbitrary guest output. Unsupported formats fail
closed. MCP stdin/stdout stay separate from backend diagnostics, and protocol
messages have time/size budgets.

Creation intent is saved **before** issuing create. On an ambiguous response or
resumed execution the script queries that same logical target; it does not issue
a duplicate create. It binds later observations to the returned name, username
and creation identity. This is scope/replacement detection, not a general tenant
authorization or security certification. The provisioning and execution phases
each use a persisted deadline (default 600 seconds, maximum 600); retries and
resumes do not restart a phase's allowance. Expired phases require a new reviewed
plan, normally allowing reuse of the existing workload.

Completed receipts are returned as historical results, not rerun as fresh tests.
The script leaves the workload available and creates no port route. Cleanup is a
separate authorized action. Read-only debug_container is used when available for
connection failures; no host-key verification bypass or service repair is applied.

To keep the local test running across an outer SSH/MCP disconnect, submit its
approved command as a durable job. The example is intended for the installed CLI:

```bash
boxweave job start --timeout 1250 -- boxweave selftest run \
  "$HOME/boxweave-smoke-run/plan.json" --approve APPROVAL_SHA256
```

The 1250-second outer job budget covers two 600-second test phases and protocol
overhead; it does not increase either phase limit. Query the job receipt and the
smoke result after disconnect. A supervisor timeout can interrupt a create, so
the persisted intent must still be reconciled on resume.

## Installation readiness and provenance

`status` and `doctor` now report installation_state and overall as:

| State | Exit | Meaning |
| --- | --- | --- |
| ready | 0 | Required local daemons active, provider/timers enabled, successful oneshot runs observed, no pending update |
| starting | 2 | Required service/oneshot is still starting or executing |
| incomplete | 2 | Configuration/observations missing, required timer disabled/inactive, unverified oneshot run, or pending update |
| failed | 1 | A required unit reports failure, or an explicit API probe returns an unexpected HTTP status |

Incus client/service activation and provider authentication are distinct. The
optional API probe reports 200/401/403 as reachable only. Local ready is not proof
of authenticated MCP execution, successful LAN access or effective confinement.

```bash
sudo boxweave status
sudo boxweave doctor --api
sudo boxweave version --verbose --json
```

The installer records `/etc/boxweave/install-manifest.json`, included in its
managed-file backups. A branch/tag resolves once to a commit before component
downloads; all components use that commit. The manifest contains requested ref,
resolved component commit, component/backend hashes, observed daemon version and
hash, observed Incus server version, preparation and successful completion time.
Unknown version values remain null. An old install without a manifest reports
not_recorded, not fabricated provenance. Version commands do not start MCP or
query the daemon; they display recorded data plus optional client binary versions.

The executing installer hash is compared against install.sh from the selected
component commit. A difference can mean a local patch **or an older installer**;
the manifest does not assert which. A pipe/stdin execution without a regular
source file reports unknown hash/comparison. This record is diagnostic evidence,
not signed release attestation. Read-only CLI output filters manifest fields and
never dumps arbitrary configuration or credentials.

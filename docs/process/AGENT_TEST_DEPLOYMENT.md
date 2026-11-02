# Agent-driven BoxWeave source deployment and real-Box testing — v1.0

**Global plan:** [BoxWeave feature roadmap](../design/FEATURE_ROADMAP_MASTER.md) — P-03 Agent test workflow / P-05 source transfer limitations. The roadmap summarizes selection and current status; this document retains detailed feature boundaries.

**Status: active developer working practice; source/helper implementation only.** Owner choices, 2026-11-01: local current worktree first; temporary private delivery allowed within previously authorized infrastructure; public GitHub only for already-public revisions; default newly created Box; existing historical Box only on demonstrated environment dependence; 1A layered acceptance, 2A bounded autonomous diagnosis/fix, 3A success cleanup/failure retention, 4A in-scope automatic actions, 5A transport fallback. **This does not create a BoxWeave end-user product, modify MCP schemas, authorize any real host or claim a live deployment.** Complement [Agent test workflow](AGENT_TEST_WORKFLOW.md) and [AGENTS.md](../../AGENTS.md).

## 1. Separation of responsibility

The development Agent orchestrates existing permitted tools; there is no new remote-execution service.

1. **Source identity**: which exact code, including uncommitted edits, is under test?
2. **Source location**: Agent-accessible worktree, verified copy in target Box, existing GitHub/CI artifacts, or another previously approved first-party remote.
3. **Transport capability**: local SSH/SCP, Box-level `connect(exec)`, a separately authorized ephemeral private transfer channel, or target-originated HTTPS download. An HTTP MCP connection is **not** proof of SSH, upload, arbitrary command execution, general host shell or network reachability. An SSH MCP connection is also **not** necessarily an Agent-local SCP transport.
4. **Install capability**: the actual guest OS, non-root app account, systemd user bus, independently prepared Provider/Incus and verified build/release installer. A *source ZIP is not an installable BoxWeave Release*.
5. **Proof and cleanup**: runnable tests and exit status are independent of both successful file transfer and control-plane responses.

The typical test topology is an independent outer BoxWeave controlling one newly created disposable Box, which may host inner BoxWeave, Provider/Incus, and a separately created inner workload when needed. Failure of the inner control plane must not prevent outer observation of the original test resource.

## 2. Agent decision algorithm (the default)

1. **Pin the goal and preflight locally.** Identify affected code and acceptance condition; run the smallest focused unit/fixture check first. Do not create a new Box for every edit. Determine actual Agent capabilities, permissions, source locations, current Git HEAD, dirty/untracked status, selected target and known baseline. Read existing `AGENTS.md`, [development workflow](AGENT_TEST_WORKFLOW.md) and current installed help.
2. **Select exact source**, defaulting to the Agent-accessible current local worktree including modified, staged and untracked nonignored files. Record parent commit SHA and the hash of the actual source bundle. Detect secret-like files and review all content before transfer. Never change to another commit merely because its transport is easier.
3. **Select Box lane**, defaulting to a new explicitly authorized disposable Box for real deployment. Historical Box reuse requires recorded positive evidence (observed previous-state/config/data/dependency difference that affects reproduction). For ordinary upgrade compatibility, a newly created Box seeded with the specified older release is preferred over an unrelated old Box. Ensure the previous state is observed before upgrade. A clone, snapshot or future environment fingerprint cannot be assumed to exist.
4. **Discover transport**, not just MCP connection. Prefer direct locally available private SSH/SCP with pre-pinned exact host key; verified identical on-Box source; then for *already-public commits* target-side GitHub/CI/Release retrieval. For local uncommitted content if direct transport is impossible, an Agent may set up a private, narrowly scoped, expiring first-party delivery endpoint/port-forward **only under existing specific authority**. Do not use public GitHub temporary branches, third-party artifact hosting, public exposure, unsecured listener, insecure host-key fallback or credential export. Require target identity and content hash at receipt. On all-path failure, return `BLOCKED` with unchanged source identity.
5. **Install/test** using only the selected authorized route. Verify package provenance, installed manifest, running app and worker versions and relevant service status; do not silently equate them with source checkout. Layer verification by change domain (table below).
6. **Diagnose and repair in scope** on a confirmed failure; record first failing step, logs, environment and actual exit. An Agent may repair only task-owned guest files/dependencies and reviewed development source within its grant, then recheck the same criterion. No automatic outer-host repair, Provider replacement, SSH-trust reset, network/Tenant policy mutation, unrelated Box restart or destructive remediation.
7. **Decide outcome**, separately recording `PASS`, `FAIL`, `PARTIAL`, `BLOCKED` or `UNKNOWN`. Remote timeout after a mutating request is `UNKNOWN`: inspect the *original* identity/receipt before any attempt to replay the mutation.
8. **Finalize and clean**. Confirmed PASS: collect minimal non-secret evidence, then remove only this task's newly created Box/guest and transient delivery files/tokens/listeners **when that deletion was part of the specific authorization**; observe cleanup state independently. FAIL/UNKNOWN: retain the necessary test scene and record it, rather than auto-destroying evidence. Reused historical Box stays; only task-owned temporary changes may be cleaned. Cleanup failure is a distinct unresolved result.

## 3. Source-selection / delivery matrix

| Source location and state | Preferred action | Truthful fallback and constraint |
| --- | --- | --- |
| Agent-local dirty/uncommitted checkout | Snapshot *current bytes* with `scripts/agent_source_bundle.py` and deliver on an authorized private path | If the Agent cannot access files/transport, `BLOCKED`; never push unpublished edits to a public GitHub temporary branch |
| Agent-local clean checkout | Use local snapshot or exact already-public commit | Choose source by proven identity; no silent switch between clean and dirty revisions |
| Code already in target Box | Verify checkout/source-content identity, modifications, dependencies and intended task ownership | Reject unknown or mismatched source; do not overwrite preexisting edits |
| GitHub public commit/Release/CI artifact | Pin immutable commit, tag source/asset metadata and independent SHA-256 when available | GitHub network access is not guaranteed inside Box; if unavailable, the Agent can fetch and privately relay the *same exact artifact* |
| Previously approved first-party remote | Fetch by independently pinned identity, then check bytes at receipt | No unapproved third-party bucket, mirror, temporary public branch or accidental disclosure |

A temporary first-party channel is a **delivery mechanism**, not an alternative source identity. It must be private to approved parties, time-limited, scoped to one bundle, authenticated or protected by an equivalent approved tunnel, and revoked/closed after use. A reusable bearer URL alone is not sufficient protection for unpublished code. Unapproved port creation, firewall/TLS/route mutation or public exposure requires separate review.

## 4. Offline local worktree bundle (implemented)

The developer-only `scripts/agent_source_bundle.py` requires Python 3.12+ and `git`. It has **no network, MCP or host-write behavior**. It includes currently present Git-tracked plus nonignored untracked files, thus capturing local modified bytes without making a commit. Ignored directories (`.git`, `.venv`, `node_modules`, generated caches when ignored) are not imported. Symlinks, non-regular files, obvious credential filenames and oversized snapshots fail closed. **Filename screening is not a secret scanner.** Review source contents before sharing. Do not include work-in-progress source when permission to disclose it is unclear.

Example on the Agent machine, where an actual checkout is locally accessible:

```sh
python3 scripts/agent_source_bundle.py pack --repo . --output /tmp/boxweave-agent-source.zip
# Save the returned SHA-256 and source commit/state OUTSIDE the bundle.
python3 scripts/agent_source_bundle.py verify \
  --archive /tmp/boxweave-agent-source.zip --sha256 EXACT_PRINTED_SHA256
```

The helper emits machine-readable `ready`/`verified` evidence with source parent commit, `clean|modified` state, file count and SHA-256. The ZIP contains a per-file SHA-256 manifest and a `source/` directory; it deliberately excludes `.git`. Package outputs must be outside the checked-out repository and are created exclusively (never silently overwritten).

For an authorized local SSH/SCP route, send the ZIP and (if required) the verified source-helper program to a **new task-owned path** inside the exact test Box, retaining `StrictHostKeyChecking=yes` and the independently pinned host key. Run the verifier there against the **original SHA-256 supplied through the trusted Agent context** before extraction. If only outer `connect(exec)` is available, it may run guest commands but **does not automatically accept local files**; use an actually available authorized delivery channel or return BLOCKED. Never copy private SSH keys into the Box.

The ZIP is *source*, not the installable offline application `boxweave.zip`: for a built package on a prepared build environment, run the relevant `npm --prefix web run build`, prepare verified locked wheels, and invoke the actual `bwmini/distribution.py build --root ... --wheels ... --output ... --source-commit ... --source-state modified` according to its current `--help`. Confirm the built package's SHA-256 and manifest; then use the existing hash-gated distribution/install entry with exact authorized target and preexisting user/systemd/config prerequisites. Do not invent flags or bypass dependencies. For a clean-host installation test of a **modified host installer**, explicitly test that modified installer in a separately approved disposable environment: bootstrapping with a stable published release and overlaying a candidate app package alone does **not** prove modified fresh-host installation.

For a published Release, the existing source-bound streamed installer supports `sudo bash -s -- --version 0.X.Y`; this never substitutes for deployment of unpublished working-tree changes. If a controlled older-version baseline is required, install/observe it first and upgrade with the reviewed candidate; downgrade/rollback is not automatic.

## 5. Layered acceptance and bounded failure handling

| Changed domain | Minimum source/fixture evidence | Additional live evidence **when claiming deployed behavior** |
| --- | --- | --- |
| Documentation or pure code | `scripts/check_knowledge.py`, focused tests | Not necessarily a Box |
| Build/packaging | Source archive/manifest integrity, install refusal fixtures | Hash-gated candidate install, installed and running versions, observed specific failure path |
| Installer/fresh host | Offline preflight and correct rejection | Actual approved clean Box, installer exit and installed services |
| SSH/HTTP MCP | Contract and negative/authorization tests | Actual authenticated initialize/tool read or exactly scoped operation over the selected transport |
| Provider/nesting/network | Targeted fixture and recorded capability | Approved actual guest create, verified exact target identity, Python output and exit 0; packet-level tests only when network behavior claimed |
| Update/migration | Repeatable fixture with prior-version data | New Box seeded with old release, independent before/after state; real historical Box only when environment dependence is evidenced |

The Agent may adapt diagnostic steps and repair task-owned resources within the same approved test grant. Do not erase a failure scene or turn an UNKNOWN remote write into a blind retry. An error in the transport, source delivery, installed runtime or actual tested behavior should be classified to the correct layer; only the *actually observed* layer may be marked PASS.

## 6. Minimum authorization, evidence and cleanup receipt

A session authorization identifies outer controller, allowed **specific test Box namespace/pool**, create/start/stop/delete scope, allowed guest execution/deployment, optional nesting, deadline, resource ownership and whether cleanup/retention is permitted. A default preference for automatic execution is **not** blanket authorization for a real host or every Box. Missing scope => only read-only analysis or a plan; high-risk operations and scope expansion require specific approval.

```text
Goal / exact expected behavior:
Authorization / execution mode: auto-within-specific-scope | plan-first
Source: local|verified-in-Box|existing-public-remote; commit; clean|modified; SHA-256
Transport: available channel(s), target identity, sender/receiver hash verification
Lane: new-disposable (default) | new-seeded-upgrade | historical-existing (evidence)
Outer controller, backend, exact Box, observed OS/Provider/Incus:
Package manifest / installed BoxWeave / running app and worker versions:
Actual checks: command or MCP call; exit; stdout evidence; failure-path proof
Outcome: PASS | FAIL | PARTIAL | BLOCKED | UNKNOWN
Remediation attempted: in-scope only; first failure kept
Resources: task-created; cleaned and independently observed | retained and why | uncertain
Evidence type: S source | D prior report | T executed | U untested
Next independent acceptance or blocker:
```

Never store private credentials or private-key material in reports. Redact any log that might include tokens, keys, URLs or personally sensitive output. Cleanup outcomes are recorded independently; an unverified delete is UNKNOWN, not success.

## 7. V1.0 implementation and known capability gaps

**Implemented in repository**: Agent rules in `AGENTS.md`, this decision/runbook, the aligned test workflow, local snapshot helper and deterministic no-network tests. **Agent-orchestrated, not a new program**: selecting/creating Boxes, transport fallback, authenticated guest execution, bounded repair, end-to-end checks, and cleanup using the existing tools that are actually granted in each client.

**Not implemented or not implied**: new MCP upload/deploy/run tools, self-hosted artifact hosting, public source promotion, clone/snapshot API, guaranteed SCP access, guaranteed Box egress, fully autonomous cross-transport daemon, safe arbitrary remote shell, always-working nesting. The actual outer dev backend has previously shown a missing SSH login wrapper ([issue #19](https://github.com/vanishs/boxweave/issues/19)) and an unverified nesting helper ([issue #20](https://github.com/vanishs/boxweave/issues/20)); this design does not fix that host and cannot claim a live nested PASS. Keep [verification queue](../verification/REMEDIATION_TEST_QUEUE.md) authoritative for executed evidence.

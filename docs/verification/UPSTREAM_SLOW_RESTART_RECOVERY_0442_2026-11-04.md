# v0.44.2: slow Containarium restarts and recovered checkpoint reconciliation

**Evidence status:** operator observations, authenticated MCP read, source review; no new real host mutation. Fixture/CI results are recorded separately after they complete.

## Retained operator observations (2026-11-04 UTC)

- The operator tried pinned BoxWeave 0.44.1 and received `UPSTREAM_EXTERNAL_RECOVERY_REQUIRES_OPERATOR` after entering `upstream_compatibility`.
- `/usr/local/bin/containariumd version` then printed `Containarium v0.120.1`.
- `/var/lib/boxweave-default/upstream-host-upgrade.json` retained `{"status":"prepared","from":"0.120.1","to":"0.128.0","prior_digest":"7d992e5df41d7070c187e8b11d516d61e8b89b4b7be677f7a0880b9cfed5d4bd"}`.
- `containarium.service` was `active/running` with main PID 389016 started at 08:13:12. During that startup it waited for core containers until 08:15:15, then opened HTTP/REST by 08:15:17. The ~125-second wait exceeded both the 0.44.1 new-daemon verification budget (120s) and its old-daemon rollback verification budget (60s).
- Read-only live MCP subsequently reported local backend `dev` healthy, daemon 0.120.1, Incus 7.5.1, 10 running Boxes. This corroborates an ultimately recovered old runtime, not complete provider/upgrade acceptance. Storage `dir`, missing core-caddy, and a compiled-in Postgres dev password warning are separate issues, not proven causes of failed binary swap.
- The supplied logs are only the **tail** of the service journal. They do not prove whether v0.128.0 began serving before rollback or whether an additional startup error occurred. Do not declare the 120-second timeout to be the only possible failure.

## Root cause and fix

In 0.44.1, `external_host_upgrade` waits 120 seconds for authenticated v0.128.0 REST/inventory, replaces the executable with a backed-up 0.120.1 on timeout, and waits only 60 seconds for old REST/inventory. When host startup itself exceeds these windows, it raises `UPSTREAM_EXTERNAL_RECOVERY_REQUIRES_OPERATOR`, leaving the checkpoint at `prepared`, even if old service becomes healthy seconds later. A subsequent installer run then refuses every such `prepared` checkpoint, with no reconciliation path.

The 0.44.2 candidate makes three bounded corrections:

1. Each external new/old runtime readiness window is now 240 seconds. The installer `upstream_compatibility` phase budget is 900 seconds to accommodate download, two service restarts and both readiness windows. A timeout remains unknown: only positive authenticated runtime/version/inventory reads are accepted.
2. On the **next new explicit installer invocation**, a retained `prepared` receipt can be reconciled only after verifying the root-trusted on-disk daemon's SHA-256 exactly equals `prior_digest`, its installed and authenticated running version both equal the prior `from` version, and authenticated `GET /v1/containers` succeeds. Any mismatch refuses the new upgrade without replacing binaries.
3. After positive reconciliation, the new `prepared` / terminal checkpoint retains the original incident object inside `previous_reconciliation`. No manual deletion of the checkpoint, systemd sandbox override or silent native POST replay is required. The existing installer lock and canonical unit checks remain.

## Test and release gates

- Fixture: simulated daemon boot where both the new and restored old process need at least 125 seconds before REST is available; exact positive and fallback verified paths.
- Fixture: retained `prepared` checkpoint from the previous incident, authenticated old-version observation and original SHA-256 proof, then a single new explicit host-upgrade transaction. Negative fixture must refuse mismatched digest.
- Source/CI: syntax and knowledge checks, test suite and browser workflows, release manifests, immutable tag/asset digests. Source-only green is **T**, not host acceptance.
- Real host: independently check the original checkpoint digest vs installed binary, old running version/inventory and completed original process; only then use a published immutable BoxWeave version. Post-upgrade check daemon >=0.128.0, MCP >=0.128.0, existing Box states, doctor, and core service health. The managed toolchain for BUILD_TOOLS is a separate upstream prerequisite. No bypass of `ProtectSystem=strict`.

**Risk:** the actual new v0.128.0 daemon might independently fail to start for a reason not visible in the journal tail. The code therefore still rolls back and surfaces a failure after a bounded 240-second readiness window; it does not claim guaranteed success on this host.

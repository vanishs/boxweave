# BoxWeave v1 — scoped `uninstall.sh` implementation and operator contract

**2026-11-19, Draft PR #153 only: implemented but NOT accepted on a physical production host.** This complements the [RC.7→RC8 teardown/reinstall lessons and proposed design](V1_HOST_DECOMMISSION_REINSTALL_AND_UNINSTALL_DESIGN_2026-11-19.md). The existing installed ChatGPT L0 controller (Schema v4, `bc1e8447`) was **not modified** in development. This tool is a **host-root-local, pre-release, installer-manifest-bound** decommissioning aid. It is not a generic machine factory reset.

## Commands and approval model

No arguments or `--help` displays help and performs no filesystem, service, Incus or network action.

```sh
sh ./uninstall.sh
sudo sh ./uninstall.sh --plan --scope service --project user-1001
sudo sh ./uninstall.sh --plan --scope controller --project user-1001
sudo sh ./uninstall.sh --plan --scope full --project user-1001 \
  --box-uuid 11111111-2222-3333-4444-555555555555 \
  --native-prepare-sha256 <CURRENT_NATIVE_JOURNAL_REVIEW_DIGEST>
```

The `--plan` response is versioned JSON containing `scope`, `ready`, `blockers`, `host_machine_id_sha256`, `review_sha256`, precise verified Unit/Binary/SQLite/Box/Native evidence, proposed actions and retained resources. **No operation occurs during plan.** Non-ready plans still print their evidence and exit **2**; untrusted data, unreadable ledger or unexpected host configuration exit **1**. Review the JSON privately because it includes original Operation IDs and native references (never Token/OAuth secrets). The hashes are valid **only on the same host, exact Project, scope, file state, current Operation table and explicit Native/Box inventory**; no `--yes` or self-approval.

After independent human review, apply the **same** scope/Project and, for full, the same complete Box UUID allowlist and Native journal approval:

```sh
sudo sh ./uninstall.sh --apply --scope service --project user-1001 \
  --confirm-host-sha256 '<SHA_FROM_CURRENT_PLAN>' \
  --confirm-review-sha256 '<REVIEW_SHA_FROM_CURRENT_PLAN>'
```

Changing a single input, service state, installation manifest, binary/unit hash, historical Operation, Native journal or Box inventory invalidates the reviewed digest. **`apply` must be run as root** and starts by persisting a `0600` root-only, fsynced `STARTED` audit **outside** controller state at `/var/lib/boxweave-uninstall-v1-audit.json`. On success it writes a `COMPLETE` record. A subsequent `plan` may authorize an additional, larger scope from the remaining manifest, preserving previous successful audit history. If interrupted, the `STARTED` original instead **blocks further applies** until operator-led reconciliation. There is no blind resume, replay or automatic deletion of the original audit.

## Three scopes — exact boundaries

| Scope | Removed on a fully verified apply | Explicitly retained / blocked |
| --- | --- | --- |
| `service` | Stop and disable only `boxweave.service`; unlink **exact installer-owned and SHA-matching** `/etc/systemd/system/boxweave.service`, `/usr/local/bin/boxweave`; daemon-reload | Keep `/var/lib/boxweave` including SQLite, Grants and original receipts; preserve all Incus instances/Project/native journal. Refuses any unverified/pending/UNKNOWN Operation, active Native prepare that would strand its rollback implementation, foreign unit/drop-in and separately installed Root helper. |
| `controller` | Everything in service, plus **only** the recognized files in `/var/lib/boxweave` and the now-empty application state directory | Requires **zero live registered Boxes**, no unresolved Operations/peer/forward/address-fence/output-GC dependencies, no applied Native preparation journal, no unreviewed backup. Host Incus/bridge/pool, image cache, root journals and separately operated OAuth/Tunnel/ChatGPT connections stay untouched. This scope **irreversibly retires** current controller-local OAuth Grants, tokens and original ledger data; preserve external audit evidence before using it. |
| `full` | Everything in controller, plus `incus delete local:<exact-name> --project <exact-user-project>` **only** for the entire operator-supplied set of stopped native Boxes whose immutable UUID/name match the SQLite owned inventory; optionally invokes existing `native-prepare-rollback` then `native-prepare-archive` with the **current** Native journal review digest | Never force-stops a running Box, destroys an unknown/foreign Box, edits ACL by hand, removes Incus daemon, shared/default storage pool, project-managed bridge, cached images, service account, external Cloudflare or ChatGPT account state. A global Scriptlet journal not independently rolled back, an installed native Root helper, foreign/unrecognized state, or uncertain native claims **blocks** the operation. |

**Important:** `full` means the deepest **safely provable BoxWeave-owned** removal, **not** universal host cleanup. Its Native rollback/Archive intentionally precede removal of the last verified implementation binary. Incus Project and image/bridge references are left for independent human review. If the Native journal is present in `APPLIED` or `ROLLED_BACK` state, `full` requires `--native-prepare-sha256` equal to the actual journal `Plan.ReviewSHA256`; it is not a checksum from this documentation or another Host.

## Enforcement details

- Only Linux root with the restricted `boxweave` UID and exact `user-<uid>` Project may inspect an installation. Manifest **schema 2** and exact fixed binary/unit/state/socket paths must match. The actual loaded systemd `FragmentPath` must be the exact unit, with **zero** drop-ins. Unit and binary must be root-owned, regular, nonsymlink, and have the hashes recorded in the installer manifest. No adoption of a legacy main/v0.x or unknown state.
- SQLite Schema **v4 or v5** is read with an explicit **read-only** connection. Every Operation's original ID, Layer, native ID and status participate in a deterministic streaming SHA; nonterminal records (including UNKNOWN) fail closed. Any Peer Grant, forward, address fence, or Exec-output GC dependency not in its released terminal state blocks removal. Ownership records must not cross Project. This is not permission to reinterpret an UNKNOWN as a retryable FAILED.
- `full` inventories Incus **explicit `local:` remote only**, with a scrubbed subprocess environment; its complete set of current native immutable UUIDs must equal the active SQLite-owned UUIDs and the **entire** explicit `--box-uuid` allowlist. Only stopped and correctly re-read individual targets may be deleted; their post-deletion absence must be verified. No `--force`, arbitrary name glob, cross-project option, `incus delete --all-projects` or shared storage/network removal.
- Separate `boxweave-native-peer-root.service` or active global Native Scriptlet journal is a **manual reconciliation blocker**, never silently disabled. Root/native preparation rollback is executed through the already installed approved Go CLI and its own independent checks; Root/Scriptlet journal archives remain on disk. Installed backup at `/var/lib/boxweave-installer-backup` requires independent retention review rather than indiscriminate recursive deletion.
- Symlinks, unexpected files/directories, foreign owners, hash or inode changes, unrecognized databases/journals, a changed systemd unit, a moving original ledger, a started older uninstall, or uncertain Incus inventory abort **before any further step**. An interrupted apply is **PARTIAL/needs host inspection**, not a success marker; the STARTED audit deliberately blocks blind replay.
- It does **not** contact ChatGPT, rotate the user's external OAuth credentials, remove cloudflared/Tunnel DNS, or assert that removing a local SQLite database erases remote registrations. A fresh controller still requires a separately approved `install.sh --plan/--apply`, new Schema-v5 credentials/OAuth origin and native preflight.

## Verification and remaining gaps

**Covered by isolated synthetic filesystem tests:** `scripts/test_uninstall_core.py` (15 deterministic cases) tests no-argument help, reproducible/read-only planning, exact approval boundaries, service-only data retention, controller-state purge, UUID allowlist and stopped-only Box deletion, blocked UNKNOWN, symlinks, foreign manifest/effective Unit, Root/Scriptlet conflicts, Native digest/rollback/archive ordering, durable completed audit, interrupted audit refusal, and staged service→controller transitions. Run `make test-uninstall`. These tests are wired into `make verify` and full-source CI quality-gate.

**Physical-host apply, intentionally NOT_TESTED:** no actual install service, Incus project, Native ACL, Root/PRS or physical SQLite was destroyed. The actual 2026-11-19 Host's connected Schema v4 L0 was preserved. Do not interpret a mocked `incus delete` or native Go rollback response as an independent physical native signoff. Additional negative acceptance with live staged installation, Scriptlet/Root fault window, concurrent host changes, storage drivers, read-only plan purity on WAL while active, and deliberate kill-between-phases remains necessary **before using `--apply` on a non-disposable Host**.

[Original physical reset, exact original native review digests and preservation policy](V1_HOST_DECOMMISSION_REINSTALL_AND_UNINSTALL_DESIGN_2026-11-19.md).

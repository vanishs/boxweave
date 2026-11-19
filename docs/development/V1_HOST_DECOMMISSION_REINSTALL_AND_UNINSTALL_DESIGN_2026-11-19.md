# Physical Host RC.7 → exact-source RC8 candidate teardown/reinstall lessons and future uninstall.sh design

**2026-11-19 PT; status: observed manual reset + installer evidence / `uninstall.sh` NOT IMPLEMENTED.**
Read with [actual Host/MCP/nested verification ledger](../verification/V1_RC8_PHYSICAL_HOST_CHATGPT_L0_L2_ACCEPTANCE_2026-11-19.md) and [current manual installation guide](../getting-started/MANUAL_INSTALL.md). This is not permission to delete an unreviewed environment or a promise to migrate a supported deployment.

## 1. Why a dedicated uninstaller is needed

The Owner explicitly approved **throwing away the pre-release development controller, SQLite v3 credentials/Grants and 14 named development Boxes** to start with one clean role-free SQLite v4 instance. A fresh `install.sh` can install and verify only its own service/unit/DB; it **must not silently remove Incus resources or historical Root/Native journals**. In the actual physical reset, the operator had to manually inventory and remove test Boxes, stop/delete only owned controller files, then **explicitly rollback and archive** an existing Native preparation journal before a fresh `native-prepare-plan` could pass. That is a clear use case for a safe, bounded **operator-assisted uninstaller**, not `rm -rf /var/lib/incus`.

This was an explicitly approved development-only reset. The historical **unresolved `UNKNOWN` from another nested test** was not repaired by wiping a different, disposable environment; a future uninstaller must preserve and refuse destructive actions when its **own current** operation/resource evidence is unresolved.

## 2. Observed manual reset, essential facts

| Boundary | Evidence and result |
| --- | --- |
| Source system | Physical Host originally ran RC.7 at `/usr/local/bin/boxweave`, `boxweave.service`, `/var/lib/boxweave`; owner-selected restricted Project `user-1001` |
| Incus infrastructure | Incus **7.6**, global `default` Btrfs pool, project-managed bridge `incusbr-1001` — **retained**; neither Incus daemon nor shared global pool reinitialized |
| Owned old instances | **14** development Boxes enumerated and owner-approved for deletion; verified absent after deletion. Locale-dependent ordering initially made a destructive script refuse; use deterministic `LC_ALL=C sort` and exact set comparison |
| Leftover image objects | Two old `user-1001` local image references remained; **NOT removed**. An empty `incus list` is not a claim that all Project assets or image cache have been purged |
| Old controller | Stopped RC.7 service, removed only selected `boxweave` systemd unit, binary and application state. Verified service inactive and old paths absent prior to RC8 install. Old SQLite v3/OAuth/credentials were deliberately discarded |
| Native journal before | `/var/lib/boxweave-native-prereq-v1.json` marked `APPLIED`, old review digest `04a6ce6d9d2aab647e624cafbad176c1af997874e39300c855ba35a2b00b19f7` |
| Native rollback | Using safely staged, exact approved RC8 candidate binary: `native-prepare-rollback -native-prepare-sha256 <old digest> -approve-local` returned `ROLLED_BACK`; old exclusive ACL removed, Project/Profile restored to its journaled **before** snapshot |
| Native archive | `native-prepare-archive` with the same digest and local approval returned `/var/lib/boxweave-native-prereq-v1.json.closed.04a6ce6d9d2aab64.01`. The closed archive remains as an audit artifact |
| Reinstall preflight | Pin exact commit `bc1e844703cdfb7ea3813cc614176d35e8883018`, exact staged binary SHA256 `06759b41fbd36727d37684276f82b7b9cfb9447df2f934b61bc79cb2b81c5386`; download its matching `install.sh`, `sh -n` and `--plan` with `--local-binary` / `--sha256`. Confirm fresh **installer-specific** host digest; no guessed hash |
| Install result | `install.sh --apply ... --confirm-host-sha256 <fresh plan hash>` installed active `boxweave` service, same source and binary SHA, clean Schema v4, restart not needed |
| Native re-preparation | Fresh `native-prepare-plan` review SHA `cc6419f9af56f7248ad607e863b4478bb937c73a965d68f0fbac8f5ffffaa4b3`; `native-prepare-apply` `APPLIED`; fresh `native-prereq-plan` review SHA `c68b3cde36b61eb57e36ac478fcb202e86e7be732698e66e1d9fc08a81960d88`; `native-prereq-verify` 8/8 and non-root `doctor.ready=true` |
| Exposure and client | New OAuth secret and current `oauth-set-origin`; public Cloudflare Quick Tunnel and new ChatGPT `boxweave L0` MCP connection. Do not reuse old OAuth Grants or keep the temporary Tunnel hostname in scripts |
| Final independent cleanup | After test, L0 51 journaled operations were 50 SUCCEEDED + one **expected** exclusive-file-create FAILED, 0 UNKNOWN; its two new test Boxes and L1's child were deleted. Host `box_list=[]`, healthy |

**The ordering matters:** the observed physical run used a **staged RC8 executable** for Native rollback even after the old installed RC.7 binary was removed. An uninstaller must not render rollback impossible by deleting the **only usable verified rollback implementation** before the root-local journal has been reconciled.

## 3. Future `uninstall.sh` product contract (proposal, not implemented)

### 3.1 Default behavior: plan-only and no deletion

Proposed operator-facing workflow (the interface below is a **design sketch**, not currently executable):

```sh
sudo sh ./uninstall.sh --plan --project user-1001
# Review manifest + exact owned inventory + pending operations and native journals
sudo sh ./uninstall.sh --apply \
  --project user-1001 \
  --confirm-host-sha256 "<digest-from-this-uninstall-plan>" \
  --confirm-review-sha256 "<inventory-and-actions-digest>"
# Separate, explicitly authorized modes for purge-state / native rollback / test Box cleanup.
```

Required defaults:

- `--plan` must perform **read-only discovery** and output structured versioned JSON or a deterministic human-readable manifest. Require explicit `--apply` plus **machine identity AND review/inventory digest**; any mismatch or changed Project/service/ACL/operation snapshot must abort. No fallback `--yes` or implicit approval from user login.
- Enumerate `/etc/systemd/system/boxweave.service` (actual install path/owner/hash), exact binary path/digest, service user/group, optional override/drop-ins, SQLite/state directory, OAuth secret/config paths, any installer backup/upgrade journal, the **active/closed** Native preparation journal and the exact owning Project/Profile/ACL. Discover **related** tunnel units, but classify foreign or separately operated `cloudflared` as external and **do not stop them by default**.
- Report owned Box UUID/name/current native state/Incus Project, active/UNKNOWN operations with **original** Operation IDs + native IDs and resource claims, private ACL `used_by` and compare-to-journal snapshots; image aliases/assets, other Project references and `default` pool/bridge sharing. This report is a **provenance map**, not permission to delete those resources.
- **Never assume database removal makes a native resource safe**. If there are live/UNKNOWN/VERIFYING/pending operations, ambiguous ACK, foreign references, unexpected config or modified ACL/Profile after apply, abort destructive cleanup and report the exact manual recovery route. Do not cancel, replay, force-release, delete, or silently adopt resources.
- Avoid indiscriminate `rm -rf`, `incus delete --all-projects`, regex-only name globs, loop over unchecked JSON, broad `systemctl disable cloudflared`, deleting shared `default` pool or `incusbr-1001`, or editing global Incus/Scriptlet. Prefer explicit ownership labels, exact canonical paths plus `lstat` (no symlink traversal), device/inode checks, stable ordering with `LC_ALL=C`, and repeated pre-delete readback.
- **Idempotency**: if already stopped or absent, report observed no-op for a known exact manifest, not a false PASS for a changed system. Interrupted uninstall emits an audit journal and resumes/reconciles **its own original** progress rather than creating fresh destructive intent.
- Distinguish **`service-only`** (stop/disable/remove exact owned controller binary+unit; **preserve DB/credentials, Projects, Boxes and native journals**) from explicit **`purge-controller-state`** (delete only proven owned app state after audit and approval) and a **separate, separately confirmed** **`revert-native-preparation`** (call existing digest/host-gated `native-prepare-rollback` and archive; never hand-edit an ACL). No implicit inclusion of either destructive mode.
- A **fourth separate** `destroy-owned-test-boxes` mode should require a complete immutable allowlist of exact UUIDs, project, original operation preflight and confirmation; delete none by default. Two residual image objects show why `purge-images` must be another optional exact allowlist with no foreign `used_by` (not needed for uninstall). Root journal **archives must remain** unless separately reviewed and deliberately retired under a documented retention policy.
- The dedicated OS `boxweave` user, `incus` membership, shared daemon, multi-user socket, storage pool, project-managed bridge, foreign Projects, unrelated host firewall and operator Cloudflare accounts are **not installer-owned** in general; do not auto-remove them. No reduction in security restrictions to make the uninstaller pass.
- Credentials: decommission exposure/expire/revoke applicable **controller-local** Grants where the service and DB are still operable; report that old ChatGPT connector registrations and external Tunnel DNS are not deleted by a host script. If the entire DB is intentionally purged, all its old Tokens/Grants become unusable only because that instance is gone; do not claim the connector configuration was removed from ChatGPT or a separate server.
- Storage: independent check that Incus `default` pool can be preserved; capacity/quota warnings remain **unverified**, not a reason to delete pool data. Before user-visible `uninstall.sh` release, test Btrfs, dir and another supported driver when feasible.

### 3.2 Safe operation ordering and evidence

1. Freeze **current source/install manifest and host scope**, enumerate Box/operation state read-only, capture original native receipt and audit backup. Classify foreign objects, `UNKNOWN`, pending operations, in-flight RPC and Root/PRS work.
2. Review and approve exact-UUID test Box cleanup **only if separately requested** and only once original claims are terminal/reconciled; re-read the native state after each action. If ownership/safety cannot be proven: **BLOCKED**, preserve state.
3. Determine whether Native prepare is applied, rolled back or archived. With saved matching **before/after** Project/Profile/ACL fingerprints and no foreign dependencies, run only its already supported `native-prepare-rollback -native-prepare-sha256 <exact current journal digest> -approve-local`; then verify `ROLLED_BACK`, archive via `native-prepare-archive`. Never reuse a digest from another Host or past apply. Preserve access to a verified rollback binary until completed.
4. Stop/disable only the exact owned `boxweave` unit and verify process/listener stopped. For `service-only`, preserve state. For explicit state purge, audit and remove only the exact own files and directory, excluding active/archived Root journals unless explicitly in approved scope. `daemon-reload`; verify absence and no persistent unowned service/port conflict.
5. Report durable evidence: exact previous/new SHA, approved choices, native journal state/path, instance inventory before/after, retained image/ACL/project/network/storage assets, all skipped/blocked objects, external MCP/Tunnel/OAuth follow-up obligations. A partial outcome is `PARTIAL`, **not** `UNINSTALLED`.
6. A fresh `install.sh` flow remains **separate**: plan with a pinned source/binary/host hash; install only service/data; independently prepare an empty owned Project/Profile/ACL through reviewed digest; `native-prereq-plan` 8/8 and approve; `doctor`; set a **new** OAuth secret and exact tunnel Origin; register a **new** ChatGPT MCP connection; then create a disposable real Guest and verify clean rollback/deletion. Old v3 DB must fail closed; no migration or mixed-auth bridge.

### 3.3 Acceptance gates for implementing the future uninstaller

| Gate | Minimum independent proof |
| --- | --- |
| Plan purity | `--plan` leaves service, DB, ACL, native Project and all instances unchanged; stable machine/inventory digest and precise intended effects |
| Foreign safety | Reject foreign Projects, unexpected symlinked paths, shared network/ACL, modified Profile, active native `UNKNOWN`/resource claims; retain original native IDs |
| Native rollback | Exact-journal fingerprint and host approval, true before-state readback, archive succeeds, **no shared Incus pool/daemon edit**; deliberate rollback failure is fail-closed |
| Controller-only | Removal of only installer-owned binary/unit does not remove native Boxes, Project/bridge, SQLite unless explicit purge approved |
| Separately approved purge | Fresh explicit decision and exact inventory allowlist; verify old credentials no longer accepted by retired instance without asserting external connector destruction |
| Repeat/partial crash | Kill between ordered phases, restart the uninstaller against the same journal; no double-delete or new-key retry; exact remaining state recorded |
| Reinstall | Pin source + binary digest; fresh Schema v4, Native 8/8, `doctor.ready`, real ChatGPT authenticated Tool and owned Box Create/Start/Exec/Stop/Delete |
| Security and external effects | No weakening of Host Incus, AppArmor, foreign NIC/ACL, shared Btrfs; no uncontrolled Tunnel or external OAuth Grant lifecycle claims |

**Non-goals:** generic Host factory reset; deletion of Incus-managed storage/network/shared resources; cross-Layer recursive orchestration; old database migration; auto-creation of OAuth passwords/tunnels in uninstall; recovery by bypassing original `UNKNOWN` claims.

**Current project status:** only a manual physical reset plus the existing independent `native-prepare-rollback`/`archive` and `install.sh` have been exercised. **There is no implemented or tested `uninstall.sh` in this record.** When a future Agent builds it, scope to a fresh disposable Lab and run all destructive negative gates before considering use on a real host.

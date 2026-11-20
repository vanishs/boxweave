# RC8 Host Btrfs quota verification and Exec quota-full recovery

Date: 2026-11-19  
Repository: `vanishs/boxweave`  
Target: Draft PR #153 / `feat/v1-s4-runtime-integration-20261116`  
Scope: native read-only status evidence; narrow Exec pre-dispatch and finalization; tests. No installed host binary, ACL, Root/PRS, or existing Box is modified by these source changes.

## Independent host evidence already collected

Host Incus 7.6, restricted Project `user-1001`, default Btrfs pool with source `/srv/boxweave-incus`. Actual host commands returned:

- `btrfs quota status`: `Enabled: yes`, `Mode: qgroup (full accounting)`, `Inconsistent: no`.
- Disposable 2 GiB Box `quota-probe-1119`, native UUID `4b887448-bc95-424f-9812-da83eb71190a`, qgroup `0/343`: `Referenced=2.00GiB`, `Exclusive=1.42GiB`, `Max referenced=2.00GiB`.
- A write intended to exceed the quota terminated without reaching the intended length. A subsequent recorded `box_exec` became `UNKNOWN / POSTCONDITION_UNVERIFIED`, requiring host-reviewed recovery. The operator executed `recover inspect/challenge/release`; the original stayed UNKNOWN and was not replayed. BoxWeave then stopped and deleted the disposable Box, leaving `box_list=[]`.

These observations prove that the *observed* native Btrfs qgroup had a real configured cap; they do not prove every current/future volume, every storage driver, or every Exec spool will behave identically.

## Implementation

1. `internal/native/exec_quota.go`: before dispatching a recorded Exec, query the exact confined Incus root storage volume's live `usage.used/usage.total`. Refuse when less than 16 MiB remains with typed `NATIVE_EXEC_SPOOL_QUOTA_HEADROOM_LOW`. This read-only preflight is not a lock or allocation guarantee. Missing/unsupported Incus volume-state data leaves the prior path and strict terminal verifier intact.
2. The exact source-backed Incus 7.6 `OpenFile` quota failure, *before* Guest Exec and *without* output/exit metadata, settles as a typed verified pre-start failure only if the native final operation ID and failed log file basename are exact. All generic, partial, mismatched or uncertain native failures remain `UNKNOWN`; no Guest command is reissued or called successful.
3. `internal/native/storage_quota_witness.go`: `host_status.native.selected_storage_quota_verified` can be true only with a recent, root-owned, non-writable, exact Project/pool/Btrfs witness of an actual saturated qgroup cap. Witness must bind boot ID and selected Incus mount device+inode, is valid for at most four hours, and automatically fails closed after expiry, reboot, or pool replacement.
4. `scripts/host_btrfs_quota_attest.py`: opt-in read-only host-root verifier using `incus query`, `btrfs quota status` and raw `btrfs qgroup show`. It verifies restricted Project, native UUID, exact root disk size, selected Btrfs source, quota consistency, matching qgroup max and actual referenced limit, then atomically writes the sole root-owned witness at `/etc/boxweave/storage-quota-witness.json`. Never invokes host shell through MCP or modifies an Incus instance.
5. Non-Btrfs pools and unavailable/untrusted/wrong/expired evidence remain `quota_verified=false`; pool availability alone remains usable and never implies quota verification. `selected_storage_space_checked` is not silently changed.

### Known safety limits

- Incus Btrfs qgroups are not a general hard security boundary for nested subvolumes; do not equate one qgroup cap with hostile-guest isolation.
- A read-only headroom observation races with subsequent Guest writes; a dispatch or terminal receipt can still become uncertain. Do not change `UNKNOWN` or auto-retry semantics to hide this.
- Native Incus 7.6 sometimes cannot record stdout/stderr if the enclosing volume becomes full. An observed native `Failure` is **not** evidence of no Guest side effects without the exact pre-start signature.
- A four-hour witness is historical host evidence, not continuous enforcement surveillance. Expired proof returns `false`.

## Source-level acceptance

Execute in a disposable Box against the exact PR source HEAD:

```bash
gofmt -l internal/native/exec_quota*.go internal/native/storage_quota_witness*.go internal/native/box.go internal/native/status.go internal/control/box_tools.go
python3 -m py_compile scripts/host_btrfs_quota_attest.py
python3 -m unittest discover -s scripts -p test_host_btrfs_quota_attest.py
go test -p 2 -count=1 ./internal/native ./internal/control ./internal/state
go build -o /tmp/bw-rc8-quota-fix ./cmd/boxweave
```

Verify source + CI output before claiming pass. Source-level tests cannot substitute for installed native Host acceptance.

## Host acceptance after independently reviewed binary upgrade

1. Create a one-time, exact-owned 2 GiB Btrfs Box in the restricted Project. Fill only **that disposable Box** to its quota as in the previous isolated host observation; avoid changing or filling host-global storage.
2. While this Box still exists and its qgroup shows an actual full limit, run on the trusted host from the reviewed repository checkout:

   ```bash
   sudo python3 scripts/host_btrfs_quota_attest.py --project user-1001 --box-name <actual-test-box-name>
   ```

   Confirm the script returns `HOST_QGROUP_MAX_REACHED` and `host_status.native.selected_storage_quota_verified=true` on the **updated, installed** binary. After four hours or reboot it must return false unless re-attested.
3. With a saturated root qgroup, invoke a *new* recorded Exec and verify it is safely rejected **before native dispatch**, without a stale UNKNOWN claim. Confirm the error class/readback is honest. If Incus volume-state cannot be independently read or a native OpenFile failure signature differs, record BLOCKED and do not generalize or bypass safety.
4. Clean up the Box through normal BoxWeave stop/delete. If any original operation is UNKNOWN, follow the documented host-root `recover inspect → challenge → release` protocol and never assume success or replay it.

## Signoff state

- Previous Host Btrfs qgroup cap: **OBSERVED**, original disposable Box cleaned.
- Go source unit tests/build: rerun at final exact source HEAD.
- Updated binary on actual Host: **NOT INSTALLED** by this change.
- Actual updated Host quota-full Exec / witness flow: **PENDING NATIVE ACCEPTANCE**.
- Do not mark RC8 native acceptance PASS solely from source tests, old host evidence, or documentation.

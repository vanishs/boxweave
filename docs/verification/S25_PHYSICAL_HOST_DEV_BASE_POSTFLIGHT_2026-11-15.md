# S2.5 physical host postflight — 2026-11-15

## 2026-11-15 owner-operated actual-host evidence — S2.5 scoped signoff

**Verdict: `S25_DEV_BASE_PASS=SCOPED_PASS_WITH_DECLARED_LIMITS` for a supervised, single trusted operator, dev-only L0→L1→L2 workflow; `BH_FULL_ASSURANCE=BLOCKED`, `HOST_GO_FULL=NOT_SIGNED`, `S5_RELEASE_PASS=NO`.** This section supersedes the prior `PENDING_HOST_POSTFLIGHT` checkpoint further below *only* for the amended dev-only S2.5 exit. The old full-assurance BH/TC matrix and all logged gaps remain open. Owner's reviewed operator-run script output is an external, host-local readback, not something inferred from a CI runner or MCP.

### Exact operator result

- Physical L0: Linux **7.0.0-43-generic**, x86_64; Incus Client/Server **7.6**; `user-1001` Project readable; restricted Incus socket present; service identity has `incus` and **none of** incus-admin/sudo/docker/lxd; `boxweave.service` active as `boxweave`, NoNewPrivileges=yes.
- **Installed exact release verified:** binary `0932ad434d2a1b113e2254f6ca02a218c2dff473f70d7ed3b2f45e93d0610174`, RC6 commit `370e419c43f06b7488c7330ba61913ffec0e9f78`; `INSTALL_INTEGRITY=PASS`, matching install manifest, manifest active=true, restart_required=false.
- `NATIVE_PREPARE_STATE=APPLIED`, `NATIVE_PREREQ_READY=True`, **8/8** host-local prerequisite checks accepted. This establishes configuration/readback prerequisites, **not a new on-host packet-level negative test**.
- Available on the reported root/state/Incus filesystem path: **1,949,227,060 KiB** (approximately **1.82 TiB**) and reported usage **3%**; logical CPU **4**; `MemAvailable=13,128,812 KiB` (approximately **12.52 GiB**). Those are free-space and RAM *snapshots*, **not** disk/Exec spool quota, Btrfs allocated-chunk guarantee, total guest reservation accounting, independent I/O/PSI stress or concurrency throughput proof. 5 L0 and 10 L1 existing running Box counts still stand.
- `FREE_INODES=0` on all three paths is an **uninterpreted statfs/df metric**, **not proof of exhausted inodes**. On Btrfs this is expected due to dynamically allocated inodes, but `findmnt`/filesystem type was **not** captured by this script. Do not claim proven inode exhaustion or real inode headroom.
- `INSTALLER_BACKUP_DIR=YES` proves directory presence **only**; `HOST_STOP_OR_ROLLBACK_TESTED=NO_READ_ONLY`. The operator showed independent host-local privileged access and existing `systemctl` plus `boxweave install-status/rollback` CLI code paths; actual *physical-host* stop/rollback was correctly not attempted on the running shared development system. Prior disposable native LAB's independent rollback proof remains scoped to that LAB.
- `ORIGINAL_UNKNOWN_MUTATED=NO`. L0's two and L1's five pre-existing fail-closed original UNKNOWNs remain untouched; L0 parent-side Exec on its claimed L1 remains locked and **must not** be bypassed. Supervised **direct authenticated L1 MCP** remains the approved dev workflow instead.
- L1 reports a **different pre-release candidate binary**, commit `89b0f891`, SHA256 `85788181e606300b441326527d853e04984f00b4cc2c4c7618a33f75845a6509`. This is an explicit known version divergence, **not** same-byte nested rollout. No physical upgrade/restart is required merely to sign this dev-only phase.
- Current pre-signoff HEAD `c2d35175a46691a53854cc74e5e7f303b7a2fc35`: [quality](https://github.com/vanishs/boxweave/actions/runs/43775387175), [S1](https://github.com/vanishs/boxweave/actions/runs/43775386195), [dual architecture](https://github.com/vanishs/boxweave/actions/runs/43775386200), [native Exec](https://github.com/vanishs/boxweave/actions/runs/43775380796), [installed HTTP MCP](https://github.com/vanishs/boxweave/actions/runs/43775380795), [independent nested MCP verifier](https://github.com/vanishs/boxweave/actions/runs/43775380834): **6/6 SUCCESS**. After this documentation-only signoff commit, reread its same-head checks before PR promotion.

### Signed scope of D-01–D-08

| Gate | Scoped S2.5 verdict | Exact boundary |
| --- | --- | --- |
| D-01 dedicated host / identity | **SCOPED_PASS** | Operator local manifest/Incus service and independently authenticated L0/L1 native IDs confirmed; existing L1 binary divergence disclosed; no upgrade/equal-byte claim |
| D-02 install / recoverability | **SCOPED_PASS_WITH_WARNING** | Installer hash/unit integrity and root-local independent stop/fallback **mechanism** present; backup directory only and physical rollback **NOT EXECUTED/NOT VERIFIED**. Keep service running; preserve ledger/UNKNOWN |
| D-03 authorized development Agent | **SCOPED_PASS** | Real OAuth ChatGPT-connected MCP read/write and strict service user; #148 OAuth refresh actor change **not safe for unattended mutation** until S3 |
| D-04 actual native minimum workflow | **SCOPED_PASS** | Live L0-owned L1 and L1-owned L2 plus previously accepted functional full native lifecycle, separate independent installed nested Verifier; cannot use L0 claim-locked Exec |
| D-05 minimum own-tenant network | **SCOPED_PASS_WITH_WARNING** | Native Project/NIC/default Profile/ACL installer 8/8 and independent earlier isolated real packet-negative proofs; **actual physical-host packet-level negative not performed here**; NO hostile tenants or independent layered AA/BPF guarantees |
| D-06 original receipts / no blind retry | **SCOPED_PASS** | Historical UNKNOWNs stay locked, exact installed MCP native WAL/ACK/recovery and independent verifier all PASS on disposable host; #151 intermittent history retained |
| D-07 finite dev workload capacity | **SCOPED_PASS_WITH_WARNING** | Measured 4 logical CPUs, 12.52 GiB available RAM, ~1.82 TiB df-free; **one supervised dev workload at a time**, no hard native disk/spool quota assertion; unknown df-inode semantics, no stress/load tests on live host |
| D-08 independent limited postflight | **SCOPED_PASS_WITH_WARNINGS** | Owner-run root-local verified readback, separate authenticated L0/L1 MCP witnesses, independently authored disposable CI Verifier, all six pre-signoff SHA Actions green. Not physical packet test or live rollback exercise |

**Operational limitations for this signature:** one trusted operator and attended Agent, no multi-tenant/hostile recursive workloads, no unattended OAuth-refresh automated redispatch, no cross-UNKNOWN workaround, no uncontrolled concurrent build/test or disk exhaustion, no assumptions of disk/Exec spool hard limits, no unrelated Box/Project cleanup, no native host changes, no independent per-depth Policy Namespace/BPF claim. Existing L0/L1 Books/Boxes and locked operations are left intact. The S3 #148 and X-02/X-03, S4 #150, S5 complete BH/TC, host rollback/network hardening remain owned and mandatory within their advertised future security/release contracts.

**Result:** S2.5 *early development deployment* is accepted under this narrow, already deployed and operator-observed trust model. It does **not** certify that the full original BH-001–019 or formal high-assurance `HOST_GO` requirements passed, nor authorize PR merge or production release. The next source stage can start immediately without another host mutation.


Scope: owner-directed **supervised, trusted single-tenant early development base** D-01–D-08, not full strong BH-019 or S5 release. Historical original UNKNOWNs unchanged; this review used only independently authenticated live MCP reads.

| Observed | Actual physical L0 | Authenticated L1 |
| --- | --- | --- |
| Native and MCP | ready/ready, Incus 7.6 | ready/ready, Incus 7.6 |
| Commit | official RC6 370e419c | local candidate 89b0f891 |
| SHA256 | 0932ad434d2a1b113e2254f6ca02a218c2dff473f70d7ed3b2f45e93d0610174 | 85788181e606300b441326527d853e04984f00b4cc2c4c7618a33f75845a6509 |
| Project | own user-1001 | own user-1001, different daemon |
| Owned Running Boxes | 5 | 10 |
| NIC, Profile, pool | ready | ready |
| Actual storage capacity | selected_storage_space_checked=false; quota unverified | selected_storage_space_checked=false; quota unverified |
| Original UNKNOWN | 2, locked | 5, locked |

Read-only UUID witnesses: L0 owns running L1 cd1ea3d0-a24f-477b-b01c-08fb5acac53e; L1 owns running L2 98ff4e4e-969c-4c77-803f-4af9f7b99463. Five L1 locked original Operations remain in HOST_LOCAL_REVIEW: e38022d284ee5ecde41b9a040bb856f6, d1e0f3c2b36a47391cd7781e00c3e1e1, 7fdd17f178b99b3c9dc181a2685f4b19, 0e4ad1324a029fb58bbc5573e1a9588b, e7e2fff12ac9f12fe7248df6b29fb409. Two L0 original UNKNOWN: 4a5130ee9025e7ee39c1866d16ad1b39 and 718b99438cfa8f1349895ade0e8dfa9a. Do not replay, unlock, delete or use an independent Incus shell to bypass their active BoxWeave resource claims.

The published RC6 L0 and local candidate L1 are **not identical binaries**. This is a disclosed limitation, not grounds for unauthorized upgrade. 5+10 running Box reservations are not proof of adequate available CPU/RAM/disk. Guest root_disk_size is **not** real free capacity.

| Gate | Result | Remaining physical-host evidence |
| --- | --- | --- |
| D-01 installed identity | PARTIAL | Live SHA/versions checked; host-local manifest, owner-trusted workload inventory and version-divergence acknowledgement pending |
| D-02 installation/fallback | PARTIAL | No host-local install-status binary/unit integrity or operator-independent fallback check yet |
| D-03 authenticated Agent | SCOPED_PASS | Real ChatGPT MCP and original S2 lifecycle already used; OAuth same-grant refresh #148 *not* approved for unattended operations |
| D-04 native dev workflow | SCOPED_PASS | L0→L1→L2 live owned UUIDs and earlier original lifecycle; isolated independent native two-Agent MCP Verifier PASS |
| D-05 minimal isolation | PARTIAL | Restricted Project/NIC/Profile observed; current physical ACL and default-deny native packet postflight not newly signed; deep hostile-sibling AA/BPF remains S4 |
| D-06 native receipts | SCOPED_PASS | Original UNKNOWN locks remain, isolated installed MCP WAL/SIGKILL/ACK-loss original receipt tests PASS; historical intermittent #151 retained |
| D-07 finite budget | PENDING | 5+10 Boxes, selected_storage_space_checked=false, quota/spool unverified; need actual disk/inodes/memory and permitted workload limits |
| D-08 actual host independent postflight | PENDING | Need root-local read-only evidence, approved fallback/stop boundary and owner acceptance of supervised trust constraints |

Exact pre-scope SHA 7be26cc3f19e609619aa2843ea2ecec62bfe549e **six Actions SUCCESS** (quality, S1, dual arch, native Exec, installed MCP, independent native verifier). Current branch SHA changes are documentation/probe only, so verify exact new SHA Actions separately. Do not turn a green disposable CI runner into physical host acceptance.

**Verdict: S25_DEV_BASE_PASS=PENDING_HOST_POSTFLIGHT.** The new read-only collector scripts/s25_host_readonly_postflight.sh should be manually reviewed and run locally by the owner on the actual L0 host:

    sudo bash ./scripts/s25_host_readonly_postflight.sh

It prints bounded non-secret KEY=VALUE install integrity / Incus Project / unit / native prerequisite / free memory, disk and inode values. It does not expose local OAuth tokens or raw SQLite, execute rollback/recover/stop, or alter a live native Box. A backup directory's presence is not proof of a working rollback. After receiving its reviewed output, separately confirm a host-owned stop/fallback plan without triggering destructive work on live L0–L5, and sign only the true minimal D-gates. Never release PR #146 or RC6 merely because this script prints RESULT=EVIDENCE_ONLY_NOT_AUTOMATIC_PASS.

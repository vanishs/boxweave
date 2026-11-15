# S2.5 physical host postflight — 2026-11-15

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

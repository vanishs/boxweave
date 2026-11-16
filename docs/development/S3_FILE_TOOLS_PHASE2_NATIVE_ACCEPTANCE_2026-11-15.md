# S3 Phase 2 — real Incus File Tools safety acceptance (2026-11-15 PT)

**Verdict: S3_PHASE2_SCOPED_NATIVE_PASS** for one supervised trusted Guest; NOT full S3_MINIMAL_PASS, not hostile/untrusted shared filesystem assurance, and not an S5 release signoff. Source candidate tested: **2f51e26e6f175211ef560f8ad32d971ea0fa3678** on the existing Draft [PR #152](https://github.com/vanishs/boxweave/pull/152).

## Independent exact-source evidence

| Real gate | GitHub run | Observed result |
| --- | --- | --- |
| Disposable real Incus 7.6 native SFTP | [#43807499748](https://github.com/vanishs/boxweave/actions/runs/43807499748) | PASS: 12 concurrent exclusive creators → one writer, unchanged original victim; native symlink/hardlink/FIFO and special-path refusals; bounded offset reads; verified temporary Guest cleanup |
| 6,000-entry directory read budget, native | [#43807499748](https://github.com/vanishs/boxweave/actions/runs/43807499748) | PASS: official Incus SFTP underlying connection + pinned pkg/sftp v1.13.12, 8,192-byte incoming budget negative; limited v1 eager directory decode, original Guest bytes unchanged |
| Real installed nonroot HTTP MCP/Incus | [#43807499806](https://github.com/vanishs/boxweave/actions/runs/43807499806) | PASS: official MCP SDK, installed service, restricted per-user Incus Project, 8/8 native prerequisites, 16 currently exposed Tool schemas, real Box Create/Exec with original receipt and exit zero |
| Installed Guest File List/Read/Apply and denied scenarios | [#43807499806](https://github.com/vanishs/boxweave/actions/runs/43807499806) | PASS: listing pagination, Admin+Viewer read and SHA-page bytes, Viewer write denial, foreign UUID refusal, path/symlink/hardlink/FIFO original byte non-mutation, exclusive create SHA/readback, same-key exact original once, changed intent denied, no fabricated native async operation ID, existing target FAILED without false UNKNOWN lock, replace/delete UNSUPPORTED |
| Installed huge directory fail closed | [#43807499806](https://github.com/vanishs/boxweave/actions/runs/43807499806) | PASS: 6,000-entry directory rejected under new 512 KiB SFTP incoming read cap, service responsive afterwards, original Guest bytes unchanged, owned native Box clean stop/delete/independent absence |
| Format/source quality | [#43807508620](https://github.com/vanishs/boxweave/actions/runs/43807508620) | SUCCESS |
| S1 source/race regressions | [#43807508624](https://github.com/vanishs/boxweave/actions/runs/43807508624) | SUCCESS |
| Linux amd64/arm64 candidate builds | [#43807508522](https://github.com/vanishs/boxweave/actions/runs/43807508522) | SUCCESS, NOT release certification |

**Implementation bounds:** no more than 50 list entries per reply, maximum 4,096 native directory entries when they fit in **512 KiB incoming SFTP wire budget**, up to 64 KiB per File Read and exclusive Create, explicit non-snapshot lexical cursor, page-only SHA-256 for partial reads. Exceeded wire/entry limits reject with no result or native file mutation. The SDK opens only the exact Incus restricted Project and checks owned Box UUID again after the native read/write.

**Failure-accounting note:** an earlier installed candidate run observed box_file_list failure before its test fixture checked Guest exit status. The runner was improved to require the original Exec exit=0 and expose typed MCP errors; the full later installed run passed. The earlier failure was not independently root-caused as a product bug, and cannot be used as a green claim on its own. S5 #151 continues to track deep restart/unknown fault-soak.

## Explicitly excluded from Phase-2 PASS

- **Adversarial parent-directory rename race:** SFTP v3 Lstat-then-path-OpenFile is not an atomic directory-fd-relative open. A Guest process able to mutate an ancestor simultaneously could change resolution after the check. Neither the official Incus SDK nor pinned SFTP client supplies the kernel openat2/RESOLVE_BENEATH primitive in this endpoint. This phase only accepts a **single attended trusted Guest**, not arbitrary mutually untrusted writers or adversarial tenant paths. Document unsupported security model, do not claim hostile TOCTOU safe; revisit with S4 #150 security/strong topology, S5 verifier.
- **Hardlink provenance:** Lstat reports a hardlink as a regular file; the native tests proved O_EXCL never overwrites an already existing hardlink but cannot prove arbitrary source provenance of hostile hardlink content.
- **Absolute memory isolation:** transport byte cap prevents unbounded directory data on the tested signed Incus 7.6 native connection, not a proven full process RSS or malicious daemon packet-parser cap under many simultaneous clients. No S3 production multi-tenant guarantee.
- **Unsafe File changes:** replace and delete intentionally UNSUPPORTED because current SFTP protocol has no atomic conditional CAS; no false pre-read/overwrite success.
- **Other phases:** #148 stable OAuth token/grant actor and Create/Delete X-02/X-03 are S4 gates; #151 installed restart original Exec readback remains S5 release blocker; full multi-Agent TC-015/016 remains S5. Historical UNKNOWN never replayed, L0–L5 real users not touched.

## Current signoff and next phase

**S3 Phase 2 accepted only under the limited supervised trust model** with distinct real native and installed MCP evidence. This change does not grant permission to deploy the candidate on a physical host, merge the draft PR, or claim all 20 Tools. A small further harness change verifies installed binary SHA against staged SHA and emits the ephemeral original operation IDs; it must pass **its own exact-HEAD CI** before those additional observations are claimed.

**Next: S3 Phase 3** in existing Draft PR #152: one real installed supervised Guest Git test initially FAILS, original Exec exit/output is read, a Guest-only authorized fix is applied, test PASSES, then independent original Guest source/Box UUID readback. Only after that and fresh exact-HEAD S1/S2/quality/native security validation can S3_MINIMAL_PASS and normal PR merge be considered.

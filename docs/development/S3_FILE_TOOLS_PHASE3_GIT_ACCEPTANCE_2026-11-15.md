# S3 Phase 3 — supervised single-Agent Git failure, repair and independent original verification (2026-11-15 PT)

**Verdict: S3_PHASE3_SUPERVISED_REAL_NATIVE_PASS** for the tested disposable source `3edb89553ed7da4f3ba4e1af37e9e661d582c0d4`. This is a **scoped one-Agent S3 milestone**, NOT a declaration of `S3_MINIMAL_PASS`, an S4/S5 acceptance, release, unattended OAuth refresh safety, or approval to modify physical host/L0–L5. Keep the existing S3 PR #152 Draft until exact final-HEAD checks and the remaining scoped exit audit are complete.

## Exact installed and native witnesses

- [Real one-use Ubuntu 24.04 / Incus 7.6 installed nonroot MCP Git and restart witness #43813435214](https://github.com/vanishs/boxweave/actions/runs/43813435214): **SUCCESS**. Official HTTP MCP client, 16 typed exposed Tools, Admin and Viewer, restricted per-user Incus Project; installation and test source SHA match. Installed binary SHA-256: `c902f7bed36acb34ec58b120d9e48dda9e1fbcd08fd2171a75640544801be371`.
- [Same-source real native SFTP security matrix #43813435225](https://github.com/vanishs/boxweave/actions/runs/43813435225): **SUCCESS**, including already-scoped O_EXCL and hostile file type/refusal checks. [Required quality #43813442142](https://github.com/vanishs/boxweave/actions/runs/43813442142) and [dual architecture #43813442154](https://github.com/vanishs/boxweave/actions/runs/43813442154): **SUCCESS**. S1 source/race result must additionally be read at its final completion before any overall S3 signing.
- One disposable owned Box: exact native UUID `5960689f-46c4-4e5e-ae39-eb18936cfcae`, original create admission `7f1351c479391c6b75fbe18fd9dd4fd8`. Original File apply exclusive create receipt `4356e51e0e3e3e7f40ede7511fe9dea7`; no imaginary Incus native SFTP Operation ID. No shared resources touched.
- The Guest Git working tree was created **inside the owned Guest** through the official MCP tools. An intentionally incorrect `calc.py` implementation (`value * 2 + 1`) and a Python unittest were exclusively created with `box_file_apply(create)`. First Git commit `c6f688a4eb1c22b3ca50b82b3af09f3692de5c17`.
- The intentionally failing test produced original Guest exit **1**, stderr assertion **`5 != 4`** and original operation receipt; a single authorized Guest `box_exec` corrected `calc.py`, committed the second Git revision, and the original rerun reported Guest exit **0**, one test and `OK`. No unsupported File replace, native side effect retry, peer Tool or second Agent was used.

## Independent result and original operation readback after installed restart

After the test and fsynced external one-use checkpoint, the disposable runner **restarted the installed BoxWeave systemd service**. A *separate process and new official HTTP MCP connection* then fetched the **same durable original BoxWeave and Incus native Operation IDs**, their terminal state and persisted original stdout/stderr, without redispatching any mutation:

| Step | Original BoxWeave operation | Incus native Operation ID | Fresh Guest exit |
| --- | --- | --- | --- |
| First Git commit | `4ed83dff43d416fb63aec779fb6ffe3e` | `7d56b1d6-7da3-412c-870d-615550b1e9c3` | 0 |
| Deliberately failing unittest | `db252710cf2623d58d3281c98d623924` | `6e2fd01d-6925-4bde-9a20-5ca76e8548bb` | **1**, original assertion |
| Guest-only fix and second commit | `c94e38b3fd20d46fc9c615fa3307b621` | `7d83d286-070d-44a0-92ad-cbf2be9635f4` | 0 |
| Passing unittest | `6215d38be4fa008256d297db70fbc420` | `91eeb29d-e2e2-42e1-bcd8-11a40a28ee6e` | **0**, original `Ran 1 test` and `OK` |

The verifier independently confirmed exact corrected `calc.py` bytes via MCP File Read after restart (SHA-256 `e77335c804a116a2e8a7445a96ea133bd60d7bbe5361606d933bcb5509392962`) and the original synchronous file create receipt stayed `SUCCEEDED` with **empty native Operation ID**.

The separately privileged **runner-only read-only Incus CLI verifier** pulled both Guest files, compared corrected Python source, test assertion, checkpoint digest, the unchanged owned Box UUID, final Git HEAD distinct from initial Git HEAD, and exactly **two commits**. Only after that verifier passed did a fresh authenticated MCP process stop/delete the original UUID. An independent Incus read verified native Box absence. Recorded required markers include `S3_PHASE3_FRESH_CLIENT_ORIGINAL_OPERATIONS_AND_FILE=PASS`, `S3_PHASE3_RESTART_FRESH_MCP_ORIGINAL_OPERATIONS_VERIFIED=PASS`, `S3_PHASE3_INDEPENDENT_NATIVE_SOURCE_GIT_HEAD_AND_ORIGINAL_IDS=PASS`, and `S3_PHASE3_SINGLE_AGENT_GIT_NATIVE_VERIFIED_AND_MCP_CLEANUP=PASS`.

## Failures preserved; no successful rerun erases original evidence

1. [Earlier installed run #43810942751](https://github.com/vanishs/boxweave/actions/runs/43810942751), source `0cfd8242`: original fixture Guest Exec yielded exit 0 and success marker, **but** the initial File Read and independent Incus check found the fixture directory and file absent. Root cause remains **UNCLASSIFIED**. Subsequent fixture fsync/extra original-ID diagnostics and successful fresh runs do not retroactively prove what happened to those bytes. Preserve incident for S5 #151/further attribution; no same-original replay.
2. [Run #43811628591](https://github.com/vanishs/boxweave/actions/runs/43811628591) and [original diagnostic run #43811972693](https://github.com/vanishs/boxweave/actions/runs/43811972693): File matrices passed; Guest Git/Python package installation exited nonzero. The original installed Guest stderr identified public Ubuntu apt network inaccessibility (exit **100** in the latter). Approved one-use **HTTPS-only, host-allowlisted bridge package relay** addressed the CI prerequisite; it is **not** direct Guest WAN/IPv6 proof or a new product capability.
3. [Run #43812691723](https://github.com/vanishs/boxweave/actions/runs/43812691723) actually passed Git failing→fix→passing and independent native readback, but Actions returned FAILURE solely because the phase-3 path omitted the older phase-2 final PASS marker. This missing marker was corrected; [run #43812942879](https://github.com/vanishs/boxweave/actions/runs/43812942879) was SUCCESS before the additional restart/fresh-client verifier was introduced.

**Contract remains limited:** the SFTP v3 attacker-controlled concurrent ancestor rename race and host process total RSS are not proven safe, while `box_file_apply(replace|delete)` is correctly UNSUPPORTED. #148 OAuth grant-refresh actor remains S4 P0; #151 historical intermittent newly admitted postrestart Exec fate remains **OPEN S5 release blocker** despite this narrowly successful same-original readback. Full TC-015/016 two-Agent recovery and all 20 Tools are S5, not established here. Historical UNKNOWNs are immutable; no new user PR, Release or physical host daemon changes.

**Next action:** add this stage-3 receipt to the final S3 scoped exit audit, require all exact-final-HEAD Go/Incus/installed checks and a fresh independent verifier, and leave Draft PR #152 unmerged until actual `S3_MINIMAL_PASS` criteria are signed.

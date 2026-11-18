# S4-B2 Stage 3 - scoped controller -> root integration candidate

Date: 2026-11-17 PT
Single development PR: https://github.com/vanishs/boxweave/pull/153
Audited baseline: `a9c6f51c35d6b3d94a63ff4013e6a897c4c85ff5`
Status: **S4_B2_PARTIAL**. **B3_NOT_STARTED**. This file is NOT an installed acceptance declaration.

## What changed

The previous controller callback-based `SubmitRootPeerRevocation` was not a product-wired isolation/verification path. The new internal installed-controller route splits original operation preparation from execution so an operator can approve a **pre-mutation** host SHA, rather than trying to approve a conntrack digest that exists only after native Stop and ACL PUT:

1. Restricted installed controller `native-peer-root-stage` binds a canonical admitted `box_peer_set` revoke to its exact admin actor, claims the two UUIDs and exclusive ACL, and persists `DISPATCHING` in SQLite. This CLI does NOT create/admit arbitrary Peer intent or expose a public B2 Peer writer.
2. Host root `native-peer-root-plan` independently checks the claimed original, protected SQLite owner/lease bindings, same-Project UUID/MAC/static IPv4/IPv6, all-Project managed network/NIC inventory, kernel route overlap, current exact single native peer grant, per-instance ETags/state and host ID. It emits the pre-isolation SHA for explicit one-original host approval.
3. An operator launches `native-peer-root-serve` pinned to that original ID/SHA and exact project/bridge/ACL, with SO_PEERCRED requiring the installed service UID; the helper neither accepts arbitrary network parameters nor grants shell/host files. Restricted controller `native-peer-root-execute` may then request read-only review and single-use execution.
4. Root independently re-reads the original and preflight, locks the existing root/Scriptlet serialization point and fsyncs a **per-original DISPATCH_UNKNOWN journal before any native effect**. Under an exact all-owned-NIC barrier, it disables autostart, stops all owned bridge consumers, independently checks no host bridge ports, updates only the exact reviewed dual-stack ACL with ETag DROP and reads it back, executes owner/route-fenced four-direction conntrack cleanup, selectively restarts only formerly-running owned NICs, and verifies exact per-NIC `in/fwd/out` nft IPv4/IPv6 bidirectional DROP with no matching ACCEPT. Any drift/partial result retains the UNKNOWN original; resumed guests are best-effort stopped on failed nft proof. Successfully resumed NICs keep autostart disabled pending explicitly reviewed later lifecycle work.
5. Only a root response carrying the exact original SHA, 4/4 scoped probes, completed mutation, nonempty post-native digest and kernel evidence on at least two NICs allows SQLite `VERIFYING` and atomic grant `REVOKED`+original `SUCCEEDED`. Controller callers cannot inject an isolation callback or a fake verifier. Lost root ACK, incomplete nft or bad digest freezes UNKNOWN without replay.

The old narrower standalone root conntrack primitives remain available solely as private code/test helpers; the actual root socket now routes to the integrated path.

## Source-level proof

Fresh disposable L0 RC7 Box: `4c0e7d92-14ff-4cb0-b3a0-e4013e2ba4f2`; original Create `f4dd83a27adf6fce1631509bb75d8dbc` / native `82f6d2fb-c020-4630-9751-c24fb8921220`. Clean branch baseline cloned and Go 1.26.9 installed. Focused installed-worker tests PASS for atomic closeout and lost-ACK/incomplete-kernel UNKNOWN non-replay; exact per-NIC nft negative tests PASS (missing/duplicate chain, missing DROP and residual ACCEPT rejected). Complete `make verify` including full race tests passed: BW original Exec `63ac4320e9ab69bd2b6aa2c1cd3bc945` / native `4e1f4c71-0261-4197-bf17-c64fd67e71a6`, exit 0.

## Independent native acceptance still REQUIRED

- This source-level implementation is **not yet installed-native B2 PASS**. The new opt-in real Incus 7.6 nft witness in `v1-s4-b2-quiescent-native.yml` must be observed on exact source SHA and tuned only after independent raw kernel readback, not by weakening fail-closed checks.
- The actual installed nonroot CLI!-> host-root Unix service!-> full native stop/ETag/nft/conntrack!-> original SQLite atomic finalization must be exercised in a **separate** approved disposable native host with a real admitted original and operator-pinned exact preflight SHA, plus foreign-Project live noninterference and negative faults.
- There is intentionally no new public B2 peer mutation Tool; root helper systemd/upgrade/restart lifecycle, Scriptlet permissions and independent UNKNOWN re-arm remain distinct pending P0 gates.
- The controller stages an **already admitted** original. B2 does not fabricate a public grant writer or mint authority to prepare arbitrary native operations.
- Current state admission locks the ACL and source/target Box operations; it does not yet hold claims for every additional sibling affected by the all-NIC quarantine. Root/native readbacks fail closed on ordinary concurrent drift but do not substitute for full independent host/claim exclusivity. This remains an integrated P0 before installed acceptance.
- No shared L0 host bridge, ACL, Scriptlet or other Agent's Box was changed. Retain `S4_B2_PARTIAL` and do not start B3, merge or release until the corresponding installed/native independent gates pass.

## Manual installed-only sequence (requires explicit local authority)

Run the private stage action as the installed unprivileged `boxweave` OS identity with `-db`, `-project`, and `-operation-id` referring to the already admitted admin original. Then run `native-peer-root-plan` as host root using the exact `-db`, `-project`, `-network`, `-acl` and `-operation-id`; inspect the SHA, native scope and impact. Independently authorize and start `native-peer-root-serve` as root using `-approve-local` and that exact SHA, then invoke `native-peer-root-execute` as the unprivileged account. This is host-authorized internal maintenance, **not** a general-purpose public MCP Peer writer. A crash or UNKNOWN journal requires separate original-specific review, never blindly run a new approval against the same operation.

## Exact native kernel follow-up

Same-source independent [Incus 7.6 native run #44229906481](https://github.com/vanishs/boxweave/actions/runs/44229906481) at SHA `65227b79857bd8b05bdd2de78001eee7097883e7` is SUCCESS; raw marker `S4_B2_NATIVE_INTEGRATED_ROOT_NFT_DUALSTACK_PER_NIC=PASS` independently proves the strict per-NIC nft verifier against real Incus DROP chains. Four live TCP/UDP IPv4/IPv6 sessions and exact conntrack deletion passed, and the isolated native fixture reported exact cleanup. That runner itself explicitly labels the scenario `DESIGN_CANDIDATE_ONLY_NOT_INSTALLED_PRODUCT`, so no integrated installed-P0 PASS is inferred.

A separate one-use `v1-s4-b2-controller-root-installed.yml` runner plus `native_peer_root_installed_e2e_native_test.go` now provision a genuine named restricted service user, operator-scoped root helper, static dual-stack A/B/C, SQLite admin original and full registered Project/bridge identity; it must independently prove the installed binary's Stage/Plan/Serve/Execute, SO_PEERCRED, actual original root journal, nft readback, postrevoke packets, terminal SQLite status and attempted cleanup. This is **pending** until its exact-SHA run and original logs succeed; any incomplete field remains a P0 failure.


## Cross-session recovery and capability proof

The work was recovered from the **existing** Draft PR #153 and a still-running one-use L0 Box, not recreated from memory. On 2026-11-17 PT:

- L0 `box_list(limit=20)` rediscovered `s4b2-root-wire-1117`; `box_get(box_id=4c0e7d92-14ff-4cb0-b3a0-e4013e2ba4f2)` confirmed exact native `Running`, project `user-1001`, and guest readiness `null`. The latter is not an execution-ready claim.
- L0 `operation_list(work_ref=S4-B2-controller-root,limit=20)` rediscovered the persisted Create, staged source writes and native Exec UUIDs, including original `814b106230e5efa76ed98679ec927675` with native Incus ID `83fbafe0-b976-45fc-8a69-5483f49476ad`. `operation_get` confirmed SUCCEEDED; `operation_output_get` confirmed guest process exit 0. `box_file_read` recovered this complete report and its independent SHA256 page hash.
- L0 `box_exec` confirmed the original worktree was based on `a9c6f51c` with staged integration files, while remote PR HEAD was already `4efd78b`. **Before any checkout**, the old staged patch was saved as `/root/s4b2-previous-staged-preserved.patch` with SHA256 `068c2c8e1fcb8350123d3390e5dcaacc5fdb0a5a47d89ccae826702ccbdaffe4`, and a clean, independently cloned `/root/boxweave-root-resume` was used for continuation. No unrelated L0 Box was changed.
- GitHub MCP `get_pr_info`, `fetch_workflow_job_steps` and `fetch_workflow_job_logs` revealed source `99d119af` produced actual installed nonroot→root `S4_B2_INSTALLED_ROOT_EXECUTE_EXIT_ZERO=PASS`, SQLite `SUCCEEDED`, peer grant `REVOKED`, root journal `COMPLETE_VERIFIED`, 4 scoped probes, three verified NICs. Job failure was **only a runner assertion** that used unprivileged `test -f` on the intentionally private root-owned journal directory. The exact native kernel probe had also been accidentally split by a misplaced shell `phase` marker. These two test-harness issues were corrected **without weakening any root permissions** at candidate `c3ed36d814ab6912f9cfcaa6d61adac63f227a6d`; acceptance remains pending its independent CI.
- The original private root helper continues to require fixed original ID, pre-mutation SHA, SO_PEERCRED, native dual-stack/UUID validation, an fsynced UNKNOWN journal before side effects, four-direction conntrack verification and per-NIC nft evidence. Independent S4-B2 signoff/merge remain separate from root integration.


## Installed controller to root exact-source acceptance

**ROOT_INTEGRATION_PASS** on product SHA `c3ed36d814ab6912f9cfcaa6d61adac63f227a6d`. Independent [Incus 7.6 native run #44235991186, job 134572145238](https://github.com/vanishs/boxweave/actions/runs/44235991186/job/134572145238) concluded SUCCESS. Original `964e7e20edfc440468b1f295ca8d344a`; operator-approved pre-mutation digest `c1ad4400010908ac90b83ef90e69d02ef47387695c39aab3197322bac2a1dbf8`.

Actual installed nonroot controller -> one-original root Unix socket -> all-NIC native stop and ACL dual-stack DROP -> exact conntrack -> resumed three-NIC kernel nft readback -> original SQLite `SUCCEEDED`, peer grant `REVOKED`, root journal `COMPLETE_VERIFIED`, replay refusal and IPv4/IPv6 packet denial. Real markers: `S4_B2_INSTALLED_ROOT_EXECUTE_EXIT_ZERO=PASS`, `S4_B2_NATIVE_INTEGRATED_ROOT_NFT_DUALSTACK_PER_NIC=PASS`, `S4_B2_INSTALLED_NONROOT_ROOT_SOCKET_OPERATOR_SHA_ORIGINAL_NFT_SQLITE_REVOKE=PASS`. Project, ACL and bridge cleanup succeeded. Source `make verify`, quality gate, source tooling and both architecture builds passed.

Earlier failing CI had a **runner-only** unprivileged `test -f` against the intentionally root-private journal and a misplaced shell phase marker splitting the real nft test command. Corrected to `sudo test -f` with unchanged permissions and actual kernel test invocation. The separate successful exact-source CI run, not a reinterpreted old failure, establishes acceptance.

**Scope:** Only the *internal* installed controller -> root chain is signed. This does not add the public Peer MCP admission Tool, all-sibling/foreign-Project noninterference proof, operator helper lifecycle/re-arm, or resolve all S4-B2 gates. `S4_B2_PARTIAL / B3_NOT_STARTED`; PR #153 remains Draft, no merge or release.

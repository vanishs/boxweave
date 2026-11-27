# Consolidation progress and resumable handoff

Canonical task: [plan](CONSOLIDATION_PLAN.md) | [inventory](RETIREMENT_INVENTORY.md) | [acceptance](CONSOLIDATION_ACCEPTANCE.md). Topic branch: `feat/v1-pre-release-consolidation-20261124`, base v1 at `9d07665dcfc5ae6345d6bfcd0d71de7b6510b51d`. Owner requested one PR, Box-first iterative work and independent CI in parallel. P0 preflight is closed; P1 product implementation has begun with behavior-preserving helper extraction.

## Active P2-P5 handoff

Owner decision 2026-11-27: end P1 development, publish any remaining
verified work, replan P2-P5 and prepare the next Agent to wait for a
start command. **P1 development CLOSED; formal acceptance DEFERRED.**
P2 R0 completed; R1 source checkpoint, independent CI pending. See the [active delivery plan](CONSOLIDATION_PLAN.md#active-delivery-plan-p2-p5-2026-11-27).

### P2 R0/R1 checkpoint

R0 investigation and finite exit ledger:
[PR #191 R0](https://github.com/vanishs/boxweave/pull/191#issuecomment-7216753691).
At 36b2654, S2/S25/Forward PASS; N2 26/28, known Legacy-only
failures; S1/quality/dual-architecture remain red on inherited H.
Independent clean-lab Box is now populated, preserved, not a fresh fixture.

R1 identified unequal postflight validation: explicit apply did not share
the interactive wrapper's installed identity and restricted Doctor checks.
Both now require Scoped manifest mode, exact candidate digest and
Project/user, active service and no pending restart, matching version/commit
and Doctor configuration readiness. OAuth remains interactive only;
historical upgrade helpers are not made reachable. Regression checks
cover both entry paths, failed postflight, stale/declined review,
prerequisites and existing OAuth preservation. No Go runtime change
or P1 H disposition. Independent fresh/installed verification of changed
wrapper remains required; reuse hosted Forward/N2/S25 lanes without
waiving any known gates. R2 remains separate Scoped offline retirement.

### Exact baseline and preserved workspace

- PR #191 Draft, topic feat/v1-pre-release-consolidation-20261124.
  Product checkpoint ff3c20e; reviewed head
  0e903a2a5d69ca35287109d4a1ef9c35bc2688c7, tree
  65b5302c9e55117f0812e96a7208ead04430856c. Canonical v1 9d07665.
- Owned Box pr188-h0-lab, 9d374e18-196a-493d-bf1f-2d582589f04d,
  Project user-1001, Running, 4 CPU/4096 MiB. Preserve
  /root/pr191-p1-replan, pr191-r2-r3, local HEAD 9c76b339 (128 dirty
  entries). Exact remote fetch and file/mode audit: 1399 identical,
  10 documents differ only by already-published newer text, one identical
  shell blob has correct executable mode remotely, no local-only paths,
  one newer remote BDH document. Original 6393a8aae7744495e904908f39937812
  and c4ef157cd2235ceb2107b4f39d75a9c8 exit0. No unpublished product bytes
  in the active P1 workspace; no stale tree push.
- New isolated planning worktree /root/pr191-p2-ready created from the
  exact reviewed head (3728c6151d2a461c0aa7f2b4f18cc7ef exit0).
  Resolve actual post-documentation head on resume. Do not reset old
  worktrees or publish unrelated historical experiments. Retain clean
  installed lab 9124a75d-b3df-4b10-af2a-972aa6eab566 and its proof;
  verify target before reuse. No fixture mutations in this replan.

### Finite remaining acceptance ledger

| Item | Current disposition | Next owner / action |
| --- | --- | --- |
| A/B/C/D/G | Verified applicable fixes | Reuse; revalidate affected final candidate |
| E | PR S2 exact crosswalk PASS, 23 jobs/72 markers | P2 R4 release S2 and P4 consumer cleanup |
| F | Legacy fresh refusal/no effect and normal Scoped retirement PASS; old N2 legacy/offline still FAIL | P2 R2-R4 full Scoped offline and explicit gate disposition |
| H | Root cause independently confirmed; no new Scoped defect found; two fails and vacuous third test remain | P1 deferred acceptance; retain real reader/native composition and all regression obligations until valid proof |
| C04 | Local WAN toggle PASS, hosted independent TLS UNVERIFIED | P5 R2 or earlier legitimate evidence; owner-approved bounded equivalence only |
| I / C01 | Exact full acceptance not PASS | P5 final full source/race/dual arch/native/quality gates |

Baseline CI: S2 45973479532 PASS (job 140419353531), Forward
45973479456 PASS, S25 45973479743 PASS; N2 45973479370 26/28
(legacy/legacy-offline fail), S1 45973479340, quality 45973479368 and
dual arch 45973479429 fail on two original Install assertions. Latest
Scoped retirement 140419352217 PASS with fresh Legacy refusal/no effect;
Scoped S2 140419352235 schema7/23 tools PASS, WAN UNVERIFIED.
[Integrated R5](https://github.com/vanishs/boxweave/pull/191#issuecomment-7216191395)
and [GLM review](https://github.com/vanishs/boxweave/pull/191#issuecomment-7216012723).
No new product bug established. Do not count closed issue #194 or
historical tool refusal as a product acceptance result.

| Phase | Development state | First command |
| --- | --- | --- |
| P0 | COMPLETE | None |
| P1 | CLOSED for development; formal acceptance deferred | No repeat investigation |
| P2 | R1 source checkpoint; independent installed CI pending | P2 R2 after stage review |
| P3 | NOT STARTED; depends on installed contract | P3 R0 |
| P4 | NOT STARTED; depends on verified operator contract | P4 R0 |
| P5 | NOT STARTED; depends on coherent candidate | P5 R0 |

## Current P1 recovery ledger

Historical P1 ledger; preserved for evidence, not active development.

**Current development disposition — owner-directed deferral:** Owner decision (2026-11-27): P1 implementation is considered development-complete for scheduling; proceed with independent P2 development. This explicitly defers the unresolved Install composed witness in #194, not a PASS, waiver, or permission to repeat a refused operation. The existing 577 PASS / 66 SKIP / 2 FAIL aggregate and Race exit 1 remain recorded. Retain the original failed tests and all safety assertions. Exact-candidate regression, native and CI evidence, C04 hosted equivalence if needed, and old-gate applicability/replacement remain open before formal P1 acceptance, merge or release. This Schema 7 development checkpoint is not an accepted or release candidate. See [#194](https://github.com/vanishs/boxweave/issues/194) and [R5 review](https://github.com/vanishs/boxweave/pull/191#issuecomment-7212654941).

The R5 pointer and stage rows below preserve the historical acceptance state; they are not the active development objective. Replan P2 against the published checkpoint. Preserve #194 and inherited final acceptance obligations without resetting R5 history.

**Current executable R5 pointer:** /root/pr191-r5-phase-history,
work_ref pr191-r5, revision10 REVIEW_BLOCKERS. Four independent review
groups RECORDED, no ready or working group; inherited Install BLOCKED,
integrated-candidate and p1-exit waiting. Exact candidate gates remain.
Use `python3 dev/bwdev --json task next /root/pr191-r5-phase-history --expect-work-ref pr191-r5`.
Actual Race original 302139a752474f6d82309ce837a87c3c terminal guest
exit1, both streams consumed, 577 PASS/66 SKIP/2 FAIL, no reported
DATA RACE warnings; unchanged source. Log SHA256
2b29956c31ff5918197222e4b63617f7761c28f1000bc607bc62d735cec4997b.
Not an aggregate PASS. Current checkpoint
/tmp/pr191-r5-checkpoint-consumed.json and Finish Gate
/tmp/pr191-r5-finish-consumed.json (REVIEW_REQUIRED, pending_output=[]).
Revision10 digest 41c18534bdbd6b934ad4bf7f6868d0f9d9cd77a0fb158eb62cf796a760c6ca75.
Source/installed/retirement independent review 6035dd73f53a992ec93f248829a09aa1,
6063cda155b5a97ce9ab872902ea1158, 029ee936999f9db124a039260d48fc34,
4b0a3c1f145233b127d21b2706696675. C01/C03/C05/C11 not full PASS;
C02 and C04 local bounded PASS; C10 inspected runtime/DDL only.
e233220 CI 7/7 and N2 28/28 success, both former 403 jobs actual success.
Not current dirty product exact-head CI. No gate removal, no P1 close.

**Preserved R4 pointer:** owned Box
9d374e18-196a-493d-bf1f-2d582589f04d, /root/pr191-p1-replan,
active history /root/pr191-r4-phase-history, work_ref pr191-r4.
Use `python3 dev/bwdev --json task next /root/pr191-r4-phase-history --expect-work-ref pr191-r4`.
Revision8 REVIEW_BLOCKERS: consumer-source, local-native and wan-c04
RECORDED within local evidence scope. Inherited Install remains BLOCKED;
candidate-ci waits on it and replacement-exit waits for candidate-ci.
Separate CI-only commit e233220 undergoing hosted validation; not current
product acceptance. Original R2/R3 history preserved below.
No further scope waiver or product acceptance.

**Preserved R2/R3 pointer:** existing BDH task history in owned
Box 9d374e18-196a-493d-bf1f-2d582589f04d, workspace
/root/pr191-p1-replan, history /root/pr191-r23-phase-history.
On resume reconcile candidate/evidence and use
`python3 dev/bwdev --json task next /root/pr191-r23-phase-history`.
Latest actual revision10 REVIEW_BLOCKERS: runtime/schema local,
consumer-source-parity and independent clean startup RECORDED.
Only original Install core regressions BLOCKED; final local exit waits
on them. Local rebuilt candidate startup evidence is not hosted CI or
recursive isolation proof. Do not treat projection as acceptance.
Do not replace the whole-stage objective with the next individual task.
Reuse task transition projection, preserve mandatory gates. P2 offline
migration scope decision unchanged; no product code published by this
workflow correction. See dev/LEARNING_INBOX.md for evidence.


Active [R0-R5 plan](CONSOLIDATION_PLAN.md#p1-convergence-recovery-r0-r5-active).
R0 assessed published 363ee542c41a748fde5a59dd63b247ac534d4a52.
[Prior convergence audit](https://github.com/vanishs/boxweave/pull/191#issuecomment-7207100365).

| Stage | Status / exit |
| --- | --- |
| R0 | COMPLETE (planning only): finite cutover sequence and gate placement below. |
| R1 | COMPLETE (preparation): Scoped fixtures, retained negatives and full State regression. |
| R2 | IN PROGRESS: core runtime/CLI/installer cutover locally verified; final R2/R3/C/E joint exit open. |
| R3 | IN PROGRESS: direct schema 7, old SQL removed, four-open refusal and targeted tests PASS; regression/consumer gate open. |
| R4 | IN PROGRESS (parallel, owner-directed): source consumer calibration; exact candidate native gates and C04 remain open. |
| R5 | OPEN: independent integrated candidate review and exact-head CI. |

Historical R0 evidence refresh: original read-only audits
b4d9a961885b13e47b44e6fb5cacdd93 and
9d38d803cfba2c20b5e36414ded20a86 exit0. Work-ref UNKNOWN list empty
at observation (not global absence proof). Remote 363ee54 seven runs:
quality/S1 running, five queued. Earlier a3b3ad0 N2 45918809967:
1 success, 2 running, 25 queued. Historical 738b269 N2 45912024502
28/28 success including WAL rerun 140233828965. Existing source/schema
matches unchanged; do not repeat valid historical acceptance.
72-marker map and later follow-ups retained: 48 direct, 24 changed;
the alternate pool, combined Update WAL, two-controller and schema
refusal/review-route witnesses have historical replacement evidence
(see crosswalk follow-ups); still verify final exact candidate.
No claim of final-head native PASS.

Publication feasibility: local temporary index removed one historical
Markdown path without changing checkout; expected tree
c4826405bd597d7a5879a32b82eda81708e2f1ef. GitHub create_tree with explicit
null blob SHA returned the identical tree; no commit or ref move and
no actual tracked deletion. Use normal Git if available, otherwise
reviewed exact-tree entries including deletions, verify tree and lease
the observed parent. No new BDH tool required.

Historical R0 workspace: same owned Box 9d374e18-196a-493d-bf1f-2d582589f04d,
 /root/pr191-p1-replan, clean at 363ee54 after one fetch. Preserve
 /root/pr191-p1-d36 local 60448ba (BDH committed-range helper) and
d2338b3 (D37 test), not published. Policy a3a01ea already represented
in 363ee54. Defer helper; D37 only if it closes a specific remaining
witness gap. No reset, force or automatic integration.

### R4 local WAN and remaining gate review

258 current product input files unchanged: manifest SHA256
574ad8cd3d58f6006bc80b0ea8e0c19738885e6006b434418defc819d0ff17c1.
Installed rebuilt binary db7c43c55509f5f5c9bcc8c06e11f896db778bd32ae2bdfab13f9542671d9943.
Real public IPv4 HTTPS example.com on 0/200
(3409ea76bbc6785181f7c0bc502aa637), off Update
b226a80b1e67c5105319a15cea0cec38, off 28/000
(3bc33c998c25c95b088dfef79eb55f79), restore
8eed43c1b4d2c05ad5fc7d81c81ad11f, restored 0/200
(00a06672af0e19e397f2964d8eb6d20e). Outer b32293a421004581d4298e4957ac4cd9
exit0; independent protected SQLite original/exit/order/UUID/integrity
PASS 8776e181ea1133a660f187c96b95aae2. Receipt digest
ce3931901f61f4a1264eeb4d4c1d42ee37f354baae9997a58675a9d34e2b86c7.
Local C04 WAN evidence only; no full native/hosted acceptance.
See PR comment 7212059093. Workload retained.

Prior remote f431bac all7 and N2 28/28 success, not current product.
Isolated seven CI-only files verified f15d8e3bf86bd0134b2514508305f8f8;
exact tree c225ceacdc596382a914722c646e9b0120537449, export
e5f1fa7513d50362cc2a49c74a445152 (131258 bytes,5 pages); published
e233220b3c3a9fa214e79d31e468fc43fcde3832 with parent lease.
Separate CI observed 1 success,4 running,2 queued.
08c42a814efa30f6c43ee6e012fe0ecb exit0: N2 262 ordered checks/28 modes
and path coverage retained; S25 52 old PR check lines and two release
markers; old S2 PR/release/publish dependencies intact; 37 tests,
format1408/YAML48/shell/diff PASS. Historical WAL jobs
140233828965 and 140233829066 pre/final SUCCEEDED consistent.
Revision8 no independent ready required work. Remaining candidate CI
and replacement exit depend on unresolved Install combined evidence.
No denied action replay, waiver, product publication or R4 exit claim.

### R4 independent installed workload

R4 resumed using guarded active history pr191-r4, rev4 then rev6.
Latest BDH-only remote f431bac CI initially 6/7 success, N2 running;
not the current product candidate. Exact authorized clean lab
9124a75d-b3df-4b10-af2a-972aa6eab566 installed rebuilt candidate
db7c43c55509f5f5c9bcc8c06e11f896db778bd32ae2bdfab13f9542671d9943.
Image prep 6a5780ac54f71806109a211e0169e67d exit0, official Ubuntu
image fingerprint 990054afd62769257b8d386635a09182191ec230480bcf99267574d186209bf5.
Actual local MCP discovery 23 tools, Create 3d2af399cb3716347ceb0d142c52770b,
Tracking 5876c4908dcf37cc174f715b7827163e, native
b2e8d652-4181-4ef6-87a1-1f4d0150f036, guest UUID
53bdf3e6-34ba-46b3-a7a2-db2ca8ea6ff0, running exact native UUID
verified (9fd0905eecf6015317c805492b307052/d946cb6ce46dee54d8540fbd55a17b7a).
Actual Exec adaa901cea3b6a396a489a42da54a8bc, tracking
76b7dda747d14ef0fe79e3abbf79f7b4, native
11a40054-adb0-405f-b11f-5616866d91f0, both original streams ended,
no truncation, guest exit0, Ubuntu 24.04.5/curl available
(49d4aa2bbf20ba544a58c3f4493c69b9 exit0). Local functional smoke
RECORDED, independent not whole native C03/C04 or hosted proof.
Retain Box, no parent socket or configuration changes; WAN task now
WORKING; consumer source still WORKING.

### R4 continuation and hosted APT prerequisite correction

Owner challenged premature termination after first R4 correction.
The prior R4 checkpoint did not establish that all independent work was
blocked. Do not equate pending verification or an unfinished task with
a blocker. R2/R3 Install obligation still explicitly BLOCKED; R4 remains
active and independent work may continue.

Original jobs 140327573272 and 140327574298 both failed before product
acceptance at the same Microsoft optional repository HTTP 403. CI needs
Ubuntu and signed Zabbly dependencies, not Microsoft product packages.
Source-only correction in N2/S25: select only an exact standalone noble
Microsoft product source for preservation outside active APT sources,
only inside the existing disposable hosted CI guards. Refuse mixed,
ambiguous, untrusted source files; retain all required source signatures,
exact Incus pin and review, and strict refresh error handling. No access
to the refused repository, signature bypass or live Host source edit.
Two private selection tests, five existing signed-refresh tests, 14
consumer tests, shell syntax, format1408/YAML48/diff PASS (5ebee3a229691f809722d374c2f6405d).
Actual hosted remediation remains UNVERIFIED until exact CI run. No
product Go change or acceptance gate removal. Preserve local checkpoint.

### R2/R3 exit review and R4 parallel entry

Owner directed R2/R3 review and R4 start. Verified saved worktree
checkpoint 73453fdbd75c2e898ec8426eabc4ce86d884d7d7dd22f25c8a533f7866c1c129,
aggregate-13 and candidate-13 hashes (3be6bf033b07a9f886eed302b628394d).
R2/R3 local runtime/schema, installed startup and source mapping accepted
within their evidenced scope only. Final local exit BLOCKED: two original
Install failures and missing combined evidence; task revision10 retained.
No full PASS or scope waiver. R4 may advance independent reversible work,
not remove dependent gates or publish a failing candidate. Latest remote
d614359 CI 5/7; N2 and S25 failed, previously diagnosed APT 403.
Not current dirty candidate CI.

R4 first source consumer correction: require Exec WAL final state to
agree with pre-restart state, in addition to exact original/native IDs.
Contradictory synthetic transcript admitted before fix (63301db3c32e6446fcb927eec6355a8a).
Correct positive fixtures; missing-marker calibration now removes all
actual full matches rather than one generated example. 14 map, 4 N2
identity, 4 S25 evidence tests pass (5d246931f8dbd2edf87b7fbe0b01e2e6);
format initially failed because gofmt missing from PATH, not code format.
Rerun pinned toolchain checks. No Go change or native PASS claim.

Next R4: preserve old S2/release and N2 gates until complete reviewed
replacement; exact current candidate 23-lane and C04 evidence still
requires a passing coherent candidate and applicable fixtures. Hosted
APT prerequisite repair is independent of Install regression resolution.
P2 offline migration unchanged; R5 remains gated. No merge/release.

### Finite remaining cutover work

| Group / stage | Actual dependency and retained oracle |
| --- | --- |
| A / R1-R2 | C03-C05,C10: server, box_tools, peer_tools, operation_tools/recover; recovery Service and plan/apply. Scoped only; retain identity, default deny, replay, reviewed UNKNOWN and independent poststate. |
| B / R1-R3 | C02,C05,C10: store/schema and nine old-table files in audit. Remove old SQL and empty-table guard only with version refusal; keep bindings, fences, held addresses and claims. Verify Open, OpenExistingRecovery, diagnostic and network read-only entries. |
| C / R1-R3 | C02,C04,C10: install.sh, local.go unit, main.go serve, diagnostic_observe.go, recover_plan/apply, scoped_drain, uninstall_core.py. Scoped default, retain old-artifact refusal, fresh review, root/guard/ledger, maintenance identity. Update schema-6 assertions/fixtures. Broad installer/CLI cleanup stays P2/P3. |
| D / R1-R4 | C02-C05,C11: 58 Go test files match Admit/CommitRecovery/mode setup, not 58 mandatory rewrites. Classify shared behavior versus obsolete-only expectations; reuse Scoped fixtures and proofs. Preserve negatives. |
| E / R1-R4 | C03-C05,C11: accept_s2_native.sh and ten direct-serve fault scripts, S25/release workflows, s2_p3 audit/manifest. Route candidate to installed Scoped producers, verify exact-source/job evidence against 72-marker map; no fake aggregate. Disposition legacy/legacy-offline N2 modes; retain generic lifecycle/retirement proof. Preserve quality/signatures/final CI. Other cleanup stays P4. |
| F / R4-R5 | C04,C11: actual public IPv4 MCP on/off/on, independent ACL/NIC and unrelated-flow denial. Exact new candidate evidence required; separate from reversible local cutover. |

Counts are diagnostic, not quotas. Actual callers and behavior decide
removal. R2/R3 and minimum C/E wiring may land in one coherent locally
verified batch; no knowingly broken intermediate publication. R1 need
not pre-run R4 or add parallel compatibility tests.

### WAN and clean-environment decision

D34 job 140240813611 succeeded as a diagnostic, not WAN PASS.
Prebootstrap Docker ACTIVE, FORWARD DROP, Docker jump PRESENT,
nft accept/drop/other 0/1/0; before/after 2/1/0, DROP 2 to 27,
source TCP/443 conntrack zero, host TLS PASS; four guest attempts
DNS positive, TCP/TLS zero, timeout. Hosted public positive UNVERIFIED.
Supports a pre-existing forwarding conflict, not exact attribution.
Read-only lab 9b71009f-a41d-411a-9739-85db5adc12de observation
8b06884660f2c77d3e2ba58f9728bd56 exit0: Incus 7.6, Docker inactive,
BoxWeave and scoped-lifecycle active, forwarding=1, two projects,
one running instance. Populated, not a clean candidate fixture.
D14 actual on/off/on is historical evidence at 92ffeda only.
Use a clean authorized native environment for new schema and R4/R5;
do not overwrite existing lab or change global firewall. If current
independent positive and Hosted negative persist, request explicit
bounded equivalence (BH009 precedent, not transferable approval).
No decision required to start reversible R1-R3; final gate stays OPEN.

### R1 outcome and residual cutover

Base 188fb680; same Box/workspace, branch pr191-r1-cutover.
Source commits 88efab9, 65657ac, c1f08fc, 24affea; source tree
60f50b931d4bf3cda33430c574fb4f20c30245db.
15 test files +127/-247; deleted shared Legacy peer-history fixture.
Retained Scoped originals, Forward, claims, capacity and diagnostics;
refusal negatives target exact originals and retained schema objects.
Targeted tests PASS; old bridge and capacity failures repaired.
Full State 155.040s, vet, diff and clean HEAD exit0:
c5a1e704d5f9ea87c5a417d9f507335e. Format PASS.
No new Box/worktree or BDH tool; no production cutover yet.

### R2 local runtime checkpoint (2026-11-26)

Remote parent 1ce5799aa874f61cf6ef95719bd097fd618f7f6c:
seven CI runs successful; not evidence of changed R2 code.
Same Box and worktree, local branch pr191-r2-r3.
Source commits 52da855, cee10c8, 3f50b95; exact source tree
764b021749dc8458c554253851370302f891d6bf.
41 files +392/-1680. Controller Create/Delete/Peer/admission
and recovery dispatch Scoped-only; legacy retirement loop removed.
Root peer and scriptlet CLI routes removed; upgrade/rollback/install-commit
explicitly refuse, fresh stage requires exact review, default wrapper
uses scoped plan/re-review/approval. No existing installation altered.
Existing scoped flag retained only as assertion; false refused.

Full Go and Python (414 tests) pass, changed package vet and format:
060d4c3739338d787575661804a0c166 exit0 (Go State 158.733s).
Go log SHA256 06425654767c11210520eca3fa79d91bdcd0c1731579790e3addb25b71d0d96a;
Python log SHA256 8f0df1173f47be25582402913c54f96879dbbeeec0340dd0e97438e94e98e213.
Binary build/ten refusal checks in d766549c7b87fb76b664ca42a388db60;
binary SHA256 a9b23b9f937ee1c6ab0ac9b273cf9e907c28e2824a49bed2e1b718d1deb77d2a.
That operation exit128 only at missing Git author; commit operation
3b9df1fed9f8edb4228e0de7e3f4ead1 exit0. Prior wrapper fixture failures
and old unit expectation failures corrected without dropping approval/drift
and no-activation negatives. No native installation/packet proof for R2.

### Install oracle disposition

Owner-approved applicability review recorded in CONSOLIDATION_ACCEPTANCE.md.
Old positive contracts obsolete, retained combined evidence still open.
Existing installed partial-ACL test is not equivalent; aggregate-13
skipped it. Revision10 REVIEW_BLOCKERS; 577 PASS/66 SKIP/2 FAIL unchanged.
No code or test edits. Platform review pending user submission; no
repeat investigation absent new evidence or legitimate resolution.

### Independent clean-lab installed startup result

Native plan 15bf416ee81fb31ebceb1fed3d2e61d0 exit0: signed Incus
1:7.6-ubuntu24.04-202611072126, plan SHA256
56c6cf67a558cfe808bedb104377831e78e32724ab8fdad042245cc804f80d95.
Bootstrap 550ea77dd8da1ae58d83071d18fcdc0a exit0: inner client/server
7.6 active, new-daemon-ready, /21 pool 2045; no existing installation.
Wrapper read-only scoped plan 39ac2fcbae05a0b2ae7b062b2417b329 exit0;
exact plan, candidate and derived restricted Project confirmed.
Actual wrapper apply 8dfb6535e239de5015b0bbdc69f1227c exit0:
STAGE, ACTIVATE, INSTALLED_STATUS; three units active, exact installed
binary db7c43c55509f5f5c9bcc8c06e11f896db778bd32ae2bdfab13f9542671d9943,
schema7, integrity OK, zero originals. Actual ready/mcp/native true
and localhost-only listener (406888338696611f8d8cdc9f756c700c); that
combined command exit1 due to private fixture permissions, not service.
Read-only diagnosis 9de4a5701babc34d4f7878c19f749478: correct private-file
refusal, fixture mode incorrect. Correct fixture mode0600 only.
0ed2ca02fefc8c6b8942b14cc36af58d exit0: exact old schema refusal,
unchanged bytes/no sidecar; retired runtime refusal; ordinary service
restart with new PID, same digest/schema, integrity and ready true,
zero operations. No nested workload or hosted CI claimed. AppArmor
unconfined, quota enforcement unverified; no security setting override.
Revision10 REVIEW_BLOCKERS: startup local evidence RECORDED; original
two Install failures still BLOCKED. Not a coherent passing aggregate
or exact remote CI candidate. R4/R5 stay open. No product push.

### Independent clean-lab startup attempt

Owner requests completion of both remaining tasks. Prior explicit Install
conversion refusal remains in force; no alternate execution or waiver.
Clean lab UUID 9124a75d-b3df-4b10-af2a-972aa6eab566 only. Network
preflight de0979156d894c4bfc6192141c3098eb exit0; GitHub/Go/Zabbly
reachable. Build deps and base clone 20b58d93f9f966b1e0facb51fa4630ba
exit0. Public base 75e6aeb, product-only diff excluding all test edits,
249234 bytes SHA256
7b11db691749fa33cc1be659b1c928a58ab59dbd77c9df8a70ba5cb61ccab75b;
gzip archive 55859 bytes SHA256
cf4b05fd0a52ac65aa912c3cb9f0654f3338a4ba0e9ae265f7b7e416994d43ae.
Two independent exclusive file creates with matching readbacks
5d566ff95e22d69ec4b05f4751030046 and 8e00ec00cb851633a53f291181a4ea8d.
Manual two-part transfer, no new BDH tooling. Both archive and expanded
patch hashes verified; Go official metadata SHA checked and go1.26.9
build exit0 c3c8fa7dc34f0a73ae52613d607ccd17. Candidate new binary SHA256
db7c43c55509f5f5c9bcc8c06e11f896db778bd32ae2bdfab13f9542671d9943.
Rebuilt dirty-source candidate, NOT original candidate-13 binary or
exact remote commit; embedded base commit metadata denotes base only.
Reconstructed textual git diff digest differed; actual sorted 258
non-test product file path/content SHA manifest identical on both
workspaces: 574ad8cd3d58f6006bc80b0ea8e0c19738885e6006b434418defc819d0ff17c1
(459e6b8b4c8a63359ba2dafd6cb5120a / 64f5f0d38e8ae7f04e2241c784d30c28).
Independent local startup only; prior aggregate two FAIL unchanged.
Native plan original 15bf416ee81fb31ebceb1fed3d2e61d0 admitted; inspect
original before any further native effect. No hosted runner identity
substitution, host modification or blocked Install regression replay.
Task history rev9 startup-native WORKING; remaining Install BLOCKED.

### Owner-authorized clean lab provision

Owner authorized a new clean test Box, not modification of any existing
installation. Created pr191-p1-clean-lab, UUID/incarnation
9124a75d-b3df-4b10-af2a-972aa6eab566, Project user-1001,
original Create 9fabfe8b255f3887cbb109ee8411b21d SUCCEEDED; native
receipt 94f8675b-b4cf-421a-9746-df58fec8341c. Requested explicit
images:ubuntu/24.04, 4 CPU, 4096 MiB, 30 GiB, standard nesting option
allowed in current Project. No privileged, interception or AppArmor
override. Actual guest preflight 495736fdb74768bf1d5b12dc116ccf01
exit0: Ubuntu 24.04.5 LTS, nproc4, memory.max 4294967296;
native configured disk 30GiB, physical quota enforcement UNVERIFIED,
no Incus or BoxWeave installation, ledger, unit or prerequisite manifest.
Guest AppArmor unconfined; UID/GID mappings observed but no verified
recursive confinement or inner native readiness. No inner Incus or
candidate installed; no old Box or Host configuration modified.
Clean inventory prerequisite satisfied, startup-native acceptance
still blocked pending supported environment capability and verified
candidate. Original two Install failures unchanged; no alternate
execution of prior refused conversion. Phase revision7 retained.
New Box reserved for independent authorized functional validation,
not a replacement for hosted N2/S25.

### R2/R3 consumer source closure and blocker review

Resume from rev6. Prior orchestration call limit observed, threshold and
scope unknown; see learning inbox and PR comment 7210732641.
Historical N2 run 45912024502 (22 mapped jobs) and S25 job 140288933978
(75e6aeb) compared against current 72-marker map, 23 lanes: 134/144
distinct lane requirements matched; 10 new schema7-only checks missing
as expected from old schema. Prior source/binary schema refusal checks
retained; not current installed proof. Actual new N2 trigger regression
11 missing path cases then 13/13 map tests pass, op
f0f6fd73c252540413684d18be3b0651. Final 29 related Python tests,
format1406/YAML48/shell/diff pass, 7ba6d573f8f52df05e48d072f8001506.
No Go changes; retain aggregate-13 577 PASS/66 SKIP/2 FAIL.

Latest BDH head d614359 CI 5/7 success; N2 27/28 (sdk-compat job
140327574298) and S25 job 140327573272 fail at APT repository 403,
before product acceptance. No signature-check change or new native PASS.
Clean fixture review: main lab and three other existing P0/WAN/Scoped
labs have schema6 installations (c4513291fea942c242f130870c197d49,
24e895634ba1049ac5fec298e5b74ec9,6025eb9ac4863416edcaf3cf7b375dbd).
No overwrite, cleanup, or runner-identity substitution.

Phase-wide revision7 REVIEW_BLOCKERS: local runtime/schema and consumer
source parity recorded; two original Install failures and clean exact
candidate installed startup blocked. Review of remaining R2/R3 exit
requires both and coherent aggregate; no independent mandatory local
task identified. R4 actual native consumer cutover/C04 remains later.
Next: legitimate resolution of original Install regression blockage,
verified candidate and authorized clean fixture, then local exit review.
No prior denied action replay, no waiver, no new BDH code, no claim of
current quota exhaustion. R2/R3 OPEN; product remains local only.

### R2/R3 exact candidate CLI and schema consumer

Existing candidate-13 SHA256
8bd7a575fe5e04ac9d36efa0b642e1b5aefdd59426e1dbdefffd5a854a6eb105
reused; no Go source change or repeat aggregate. Linked-symbol inventory
66fec5f45f643c8fbb02ccb7ccf47094 / 803f6ed5650f87a8ddcb01b5be398b16:
old NativePeerRootServe, native IPv4 compaction/review execution absent;
shared listener, historical refusal stubs and maintenance unit text
remain. No old table references in tracked non-test Go search. Symbol
scan is bounded, not proof of all reachability or final P2/P3 cleanup.

30644e777841e96af0dfc934fc65ca43 exit0: actual candidate status AND Doctor
refuse all seven private invalid DB layouts; original bytes unchanged,
no sidecars. Old serve mode and three retired CLI command names refuse
before creating ledger. Evidence dir /tmp/pr191-r23-cli13-mhmqrd_y.
No installed service or native effects. Strengthened existing installed
schema verifier and N2 crosswalk: require exact status error per layout,
no generic failure or altered DB; retain original Doctor oracle. New
status oracle calibration + 7 actual status repeats:
35dc2a7ba16d7e97adb71930f4a686c2 exit0, 8 schema tests, 12 map tests,
1406 format, 48 YAML. Current candidate new ledger/reopen twice:
34b9272b5c4dd32a4a34d86512aaaa06 exit0, 23 non-internal tables, exact
schema 7, required claims/history tables retained, old DDL absent,
integrity OK, zero operations. Ready false, MCP ledger readiness true,
native ready false; not installed startup. Evidence
/tmp/pr191-r23-fresh13-uptrjgsa.

R2/R3 exit not achieved: unchanged Go aggregate 577 PASS/66 SKIP/2 FAIL;
two previously explicitly refused Install conversions remain blocked.
Positive installed startup and exact candidate native/consumer cutover
not executed. Do not replay denied write, skip failures or claim native
PASS. Full offline retirement remains P2; old gate retirement and final
native equivalence R4. No new BDH tools or changes; existing fixtures,
binary and historical full aggregate reused. No quota exhaustion claim.

### R2/R3 Forward-history correction and CI evidence consumer

Owner-approved P1/P2 boundary implemented (d6b7d9ca6f9918b68cb990f6c891016d
exit 0): full offline retirement migration P2; minimum P1 dependencies
and both blocked Install regressions remain P1. No waiver.

Concrete Forward defect: 17e4060291dac062135b8d179ab07d1f exit 1, changed
host_port accepted by ReadScopedRetirementLedger. Restore exact original
Create/Delete intent, native receipt, identity and typed claim proof
in same read-only Scoped snapshot; no Legacy DDL. Normal revoked and
no-dispatch cancelled rows accepted, port/device/owner/receipt drift
refused. a5e28c776b6440c818d02bdd2fcf3189 7/7 selected PASS;
6347576f8bf5047a0c8925e89c92fce7 8/8 top-level race tests PASS.
No installed native retirement claimed.

84f8cf30b2a3ef3a68a969daaa0d18d6 exit 1: seven already-deleted Legacy
files still indexed. Stage those exact deletions. No format bypass.
154a7844a234214c1eafd3efba5d8ad5 guest exit 1: format 1406, YAML48,
vet and candidate build PASS; binary SHA256
8bd7a575fe5e04ac9d36efa0b642e1b5aefdd59426e1dbdefffd5a854a6eb105.
Aggregate /tmp/pr191-r23-aggregate-13/go-test.log 323004 bytes SHA256
e1f2d89d9377345f67b92c254bfafed67f3f003f24140261a3e763db12dd2a4b.
No source drift; 645 top-level, 577 PASS, 66 SKIP, same two Install
failures: TestS4B2ConntrackScopeMustComeFromFullSQLiteLedger and
TestS4B2RootAuditRefusesSQLiteAndNativeInterleaving. No new Go failure.

C/E: exact ACK originals twice, WAL cut/restart/native ID and order;
N2/S25/RC S25 sanitized CI lane locator after original checks.
23-lane receipt index rejects missing/duplicate/wrong-map/source/context,
reused digest and mixed originals. Does not authenticate CI or establish
semantic equivalence. Old N2/S2 gates retained; see crosswalk dispositions.
R4 actual replacement and C04 pending. d1db37b62621dc8cb6479eda26c55777
exit 0: 31 direct ignored Go probe builds, module and format/YAML PASS.
60b071afb8a45b2dab93af0a36d8f9f5 exit 0: 111 related Python tests
(19 S25, 66 Scoped, 12 map/receipt, 4 identity, 6 offline, 4 roundtrip),
shell syntax, tracked format, YAML and diff check. Source calibration
only; no exact-head hosted run.

R2/R3 OPEN: two pre-existing platform-blocked Install conversions and
known failing aggregate prevent verified product publication. Independent
Forward and consumer source work completed. Preserve original refusal,
do not weaken gates or claim native acceptance. Old gate cutover requires
actual candidate/review. BDH index reused, no comparative benefit or new
promotion. No quota exhaustion claim, merge or release.

### Owner-approved scope sequencing (2026-11-26)

Owner approved P1 runtime/schema/core regression scope and P2 full offline
retirement migration. Preserve all necessary P1 dependencies and final
maintenance evidence, no gate weakening. See plan and acceptance.
Blocked Install conversions remain P1 OPEN. Continue unrelated safe work.

### R2/R3 continuation: recorded learning and exact lane crosswalk

Owner requested record first, then continue product work. Learning inbox
and execution log updated without new BDH tools or candidate promotion:
479943fc401700926e266f699e16a19a exit 0. Safety refusal hypothesis remains
unconfirmed; no retry of the refused Install write. Other work continued.

C/E source work: map all 72 original S2 requirements to 22 N2 modes plus
S25 (48 direct, 24 explicit replacements pending review), retain the six
ordered operation_list witnesses. Read-only lane consumer and aggregate
index validate source, candidate identity, distinct lane transcripts and
required exact lines; no forged old aggregate, no inference of native PASS
from untrusted logs. Added checks to current N2/S25 and release S25, and
source verification. All prior per-mode checks and old S2 consumers remain.
S25 records actual candidate digest; distinct N2 digests are exposed,
not silently treated as one executable. Six unmapped N2 modes retain
their full original checks.

5c91e08d975da8a9f749efc7527f6dcb exit 0: 107 related Python tests
(19 S25, 66 Scoped, 8 crosswalk, 4 N2 identity, 6 offline, 4 roundtrip),
shell syntax and 48 tracked YAML files no duplicate keys. Source-gate
wiring and updated crosswalk documentation: b83b9dd90928dc678b9ece1ed3476315
exit 0, 16 source tests. Prior full Go aggregate is unchanged; no Go edits.

Remaining actual dependency: two Install conversions are still blocked
by the earlier explicit platform safety refusal; no verified override.
R2/R3 cannot publish a known failing candidate. The multi-lane source
consumer is now ready for independent native evidence review; old S2
single-run gates and two legacy N2 modes remain in force until complete
replacement/obsolete-contract disposition. Existing offline retirement
supports only reviewed Legacy; its UNKNOWN archive and foreign-resource
oracles must not be retired as equivalent to Scoped live drain. P2 full
offline conversion remains an explicit obligation. Exact-head CI, C04
public WAN and R4/R5 remain open. No new runtime or acceptance waiver.

Manual Finish Gate: recorded known original exit results and bounded
verification. Independent source mapping and calibration performed before
checkpoint; no active test execution claimed. Do not replay the blocked
write or claim quota exhaustion. Candidate remains unpublished, preserve
the original workspace and obtain legitimate review of blocked action.

### R2/R3 continued C/E source verification (2026-11-26)

The prior Install write was explicitly blocked by OpenAI tool safety, not an
observed BoxWeave admission failure. Platform rationale is unavailable:
possible false positive, not established. Preserve the refusal; do not
retry the denied action via another transport or representation. Prefer
pure fixtures and read-only checks for future verification, with actual
native effects restricted to existing reviewed disposable environments.
Request platform review of the original error with redacted context.
No measured false-positive reduction or BDH promotion claimed.

Independent C/E work in the original Box:
- Shared S2.5 PR/release consumer retains 55 exact markers, two original
  lines, explicit Legacy disposition, and exact single source binding.
  Release selects existing Scoped producer and retains both original
  release witnesses. 55c03a8cec1e9d63c8a78bdaac53c815 exit 0: four tests,
  including omission of every required marker. No native run or release.
- N2 installed Doctor requires exact schema 7 and seven refusal layouts
  (five retained, future and partial added). Source tests 7/7,
  ede6f4a146e7dc345b42cb914afbebd7 exit 0; native pending.
- Candidate f0e77f29ac553791c18c252403e478c623a110de10b458db06b99af22b954afc
  ran status twice on a fresh private offline DB, no Incus socket:
  f05fcb614a0de2e86fad6c7ae3087644 exit 0, 24 tables, schema 7, zero
  Operations, retained tables present, old retirement/compaction absent,
  both not ready. Add checks to existing dual-arch CI smoke. Not native
  service startup or proof of both architectures.
- N2 transcript binds source, built and installed candidate digest, before
  native witness checks. Read-only consumer rejects missing, duplicate,
  malformed or mismatched identities. 9e7db02013316d394f43df305e150af3
  exit 0: identity 4, S25 4, schema 7 tests, shell syntax and workflow YAML.
- Rechecked exact 72-marker union: 48 direct, 24 changed,
  85f5a4adde9a36c0d1e71c46a8b8b3db exit 0. Old S2 consumers and release
  route remain intact until exact multi-lane replacement. Legacy N2 offline
  interrupted archive/purge, foreign preservation and exact UNKNOWN
  disposition remain open; existing Scoped retirement is not complete
  equivalence. No fake aggregate or WAN waiver.

R2/R3 OPEN: two Install conversions require legitimate resolution of the
explicit refusal; S2/N2 convergence remains incomplete. Last full regression
31 State failures resolved, two Install failures remain. No Go change in
this continuation, no product publication, merge or release.

### R2/R3 resumed outcome-group checkpoint (2026-11-26)

Box pr188-h0-lab, /root/pr191-p1-replan, local HEAD 9c76b33,
remote head 75e6aeb (7/7 CI success on non-product remediation only).
History/retirement: 16/17 selected tests passed in 94219e0c50d4b092302ad37d000d4c82
(exit 1, log SHA256 03f0eacc335ceb31c67ed1ed1acfd3d70a4f533f0fdb4749cdb617e676789231).
Remaining concurrent Create test corrected to real Project admission fence
(one admission winner, retired address refused and fresh address accepted);
7bcf3ab4c615cf904f6a536015454634 exit 0. All ten baseline history failures
now individually have passing source witnesses. Full aggregate still required.
Retired queue/archive/reuse/compaction APIs refuse before callbacks; existing
Scoped Delete atomic fault and UNKNOWN tests retain the transactional oracle.
History includes 2048 exact synthetic retired rows with tail corruption,
257+1 retained rows, real Scoped admission, peer tombstones, restart, held
identities and forward history digest change. A new negative found a real
wrong-bridge same-Project fence omission (ef49ff7d72ddc20554d344b383067bbc
exit 1); fixed full-project fence scan, 0614b5537869fffa09d010552d21686d
exit 0. No native claims from SQLite/synthetic receipts.
Peer 17 baseline State failures converted to Scoped original, stage/claim,
proof, terminal policy and read-only recovery. d6caaf8f1410ea078297d9e3eefd87ad
exit 0, 17/17 selected, no skip, log SHA256
a6f21bb0cbd002ffbe025a2b01ac7a53f40e061d13b312a3ff6e4a80bb8ca451.
Old markers retained for corresponding State assertions, not native proof.
Dependent Delete/recovery: all four baseline failures converted and passing.
Actual defect: legacy peer child intent/dispatch could not satisfy Scoped
Root execution. Canonical schema-2 child with exact parent/child claims and
events, only linked parent excluded from project-wide barrier, no grant
children, no replay, terminal lineage retained.  ebc12b42d74d4a5c5b5bcefb3b9552e0
linked cleanup/forged link refusal; 38b2873c321b4c0817e5061bfad9f145 atomic
Delete recovery (legacy review refused); 042a755cd37b868b6791680a4d10f070
two A4 tests; f82c8004daf9e6216e4a4bff1781851d lost child ACK, parent stays
UNKNOWN and cannot dispatch. All exit 0; source/SQLite tests only.

Full current Go regression 44c60c8bd944644214ab71b6e60a1ee9 exit 1,
source_drift=False, 645 reported top-level records (577 pass, 66 skip,
2 fail); ten packages pass; State 140.410s PASS. Log
/tmp/pr191-r23-aggregate-12/go-test.log SHA256
dd55fcedc6a3fd8ce01ce68fdc2536807dde8624752e38f7e4ad9d490a9cf8f2.
Only remaining failures TestS4B2ConntrackScopeMustComeFromFullSQLiteLedger
and TestS4B2RootAuditRefusesSQLiteAndNativeInterleaving: old shared ledger
positive witnesses refused before native callback. New Scoped Store fixture
compiles, test-write action blocked by tool safety check; do not bypass.
No test deletion, skip or relaxed predicate to force green. Further C/E
consumer wiring remains required (old direct serve and single-run
72-marker evidence consumers still incompatible, exact native gates open).
Updated live MCP descriptions to actual Scoped-only behavior, schema-7
candidate comment. Static verification a9ac8697d60d5c76e89c657bce5db8cd
exit 0: format, diff check, vet, modules, build, 58 uninstall tests,
8 evidence-audit and 10 scoped-acceptance tests, shell syntax. Candidate
SHA256 f0e77f29ac553791c18c252403e478c623a110de10b458db06b99af22b954afc
(not a deployed/accepted binary). Existing PR comment 7209225391.
Learning: reuse aggregate navigation and exact receipts, keep raw log and
exit status; avoid oversized argv (>4096 bytes); never bypass safety refusal
or conflicting original resource lock. No new BDH improvement/promotion.
R2/R3 OPEN/BLOCKED on Installer witness conversion; no product push, merge
or release. Minimum C/E and exact-head native evidence remain open.

### Resume P1 R2/R3 directly

Current local HEAD `9c76b3394f85d8b4545b9db82b2dd2b7328fd504`,
Box `9d374e18-196a-493d-bf1f-2d582589f04d`,
`/root/pr191-p1-replan`, staged/dirty schema-7 and test changes.
Prior published product head `1ce5799aa874f61cf6ef95719bd097fd618f7f6c`;
reconcile the separate non-product remediation with local source without
resetting or replacing dirty work. Tracked checkpoint (not remote backup):
`/tmp/pr191-r23-audit-lineage-checkpoint-1841.patch`, SHA256
`74fee48d37b0bbcfe8aa92b6d6bb982c744efc826b6f7dfd69f420b38572b134`.
Last full State/Install regression `7397d63915e73c8c20d6931bc589168a`
exit 1, 33 top-level failures, log `/tmp/pr191-r23-audit-lineage-regression.log`
SHA256 `5f5c0ea9ed536364293991aac9ee3fe319df465c6df38a8df5350a3a4b8e21aa`.
Inspection `cbece5ba11422cfefc4ee717541c54b8` exit 0, both streams consumed.
See [self-audit](https://github.com/vanishs/boxweave/pull/191#issuecomment-7208593686).
Prior 44 selected failures versus 33 full-suite failures are not equivalent
workloads; no observed prior quota exhaustion.

| Internal outcome group | Retained oracle / action |
| --- | --- |
| History, retirement and compaction (10 reports) | C02/C05/C10/C11: exact Scoped terminal history, corruption, held identity/address/claims; reuse Scoped history/capacity witnesses, retire Legacy-only queue/archive/reuse/compaction with rationale and explicit refusal. Preserve large-history/concurrency where applicable. |
| Peer and Install (19 reports) | C03/C04/C05/C10/C11: exact Scoped original/grant/revoke, direction independence, claims, no replay/drift; convert controller/Root audit and conntrack fixtures, not native-proof claims. |
| Dependent Delete/recovery (4 reports) | C05/C10/C11: exact linked children, retained history and atomic original/review recovery; use Scoped fixtures, preserve negative calibration. |

Counts are navigation, not quotas or permission for blanket test deletion.
Map each changed assertion to a retained witness or obsolete-only rationale.
Finish minimum C/E wiring (schema-7 diagnostics/maintenance, S2/S2.5/N2
producers and consumers), fresh schema/no-write refusal, binary startup,
72-marker crosswalk and aggregate source checks. R4/R5 including WAN remain
mandatory and open. Use the [batch helper](../../dev/skills/verification/SKILL.md#go-failure-navigation-candidate)
and one active ledger; no extra stage or user command for routine fixes.
R2/R3 resumed; latest checkpoint above supersedes prior failure count. Preserve original workspace; do not reset or promote incomplete evidence.

### R2/R3 local schema-7 checkpoint (2026-11-26)

Parent 9c76b3394f85d8b4545b9db82b2dd2b7328fd504; remote PR head
1ce5799aa874f61cf6ef95719bd097fd618f7f6c unchanged. Same authorized
Box/worktree pr191-r2-r3, uncommitted source, no native effects.

Direct schema 7 removes three IPv4 retirement/queue/compaction tables
and two indexes. No old-table SQL in Go production sources. Historical
interfaces explicitly refuse before SQL/witness; remaining caller
cleanup required. Scoped admission, recovery plan/writer, binding,
maintenance and address holds use retained model. History digest v7.
Scoped uninstall requires schema7 and refuses even empty old tables;
non-Scoped historical offline reader retained for P2 review, refuses 7.
N2/example schema assertions updated, not complete native wiring.

Evidence: 4398d548eec21e92d43177a72dc23c02 (exact v6 original UNKNOWN,
receipt, claims, queue, bytes retained across four opens);
0e0fc55fd3b1ef6b547af06996d97ed1 (fresh schema/old-table and callback
refusal, scoped lifecycle/peer); ff39790fbcb0e081898f9c2b07614e48
(uninstall 58/58 and diagnostics); a5e750d6a9c1f70706074dbc2ba7f318
(shared pin fixture); e7fc998b5c5459e2df9e1f00f9d04bd9 (Create faults);
4c8eb934d87c2696bfa00ebcc64b98c0 and 814cc0b69b5ed60f76c34be3abf3cf84
(Delete/Create recovery faults); abc757942719fca9708dff6f090474fe
(replay/plan-writer); e00daff538a84d24ac1a44b09c997bfc (retained holds);
28d1f149788ec1865046024eb5416f47 (maintenance digest), all exit0.
c65312bcec14524019dd333127ef9ff4: all packages compile, vet changed
packages, diff check PASS; selected regression exit1, 44 failures
(42 State, 2 Install), not all suite. Outer audit exit0 is NOT test PASS.
Logs /tmp/pr191-r23-schema7-first-regression.log and
/tmp/pr191-r23-final-failed-recheck.log (SHA256
d6a1521da1a4adad9e1e83e0f364b57e6bb0a44325ff9dc8c7f71d5b70229c5b).

Mixed-table runtime negatives move to incompatible-open tests; retained
same-schema replay, stale plan/writer, pin conflicts and history tested.
Legacy lifecycle-pin subcase removed, Scoped retained. Old status
characterization now explicit refusal. Historical DDL only test fixture.
Next: classify 44 failures and potentially vacuous Legacy negatives,
preserve genuine history/dependency/peer/interleaving/reviewed recovery
witnesses, retire obsolete-only semantics explicitly; finish diagnostic
classification and S2/S25/N2 wiring. Full regression and native gates
OPEN. No R2/R3 exit, no merge/release/publication. Prior suffix preserved.

### R2/R3 BDH cycle 02 reuse

Failure navigation candidate used on actual pending tests. Initial
top-level failure count corrected to 71; previous selected recheck 44.
Converted TestCreatePinOriginalCollisionAndUnknown to Scoped original,
same-project UNKNOWN refusal and synthetic cross-project collision.
Focused PASS fa79d954af9b7de01ec5070d5cff6a67; selected recheck
1273ed77a06e0154676cb2bad30f4da2 exit1 internally, 43 failures.
No acceptance gate changed. See dev/LEARNING_INBOX.md cycle 02.
Next use the failure index to migrate actual remaining Peer/Delete,
history and reviewed-recovery witnesses, then consumer wiring and full
source/native gates. No publication of incomplete candidate.

### R2/R3 recovery/Peer regression conversion

Remote 1ce5799 remains unchanged (seven successful workflow runs);
local parent 9c76b33, prior uncommitted schema and BDH changes preserved.
On the same worktree, converted reviewed concurrency to actual Scoped
originals (same/different keys), retired identity bind to a no-write
refusal with original UNKNOWN, events, claims and no recovery receipt,
and Exec output recovery to Scoped original with foreign-path refusal.
Dedicated Scoped Delete atomicity witness retained and run.
Peer provenance uses actual Scoped admission/stage/finalize: grant,
revoke, regrant exact links; UNKNOWN cannot use dispatch receipt;
lost ACK recovered with the same proof and callback count one, both
directions. Synthetic native evidence, not native effects.
Original tests exit0: 5eb9995a71b7b4d04a0b8490df0b56f9,
60c02f525bc0d6f8c9c7727337d292b0, 64782d1379043eeccbb356df05c25807,
30b27968362058e8a90a8bd488baade9, eeb909ee114842d647068ca190676125.
Six previously failing test names now individually pass, plus retained
Scoped Delete and recovered revoke. No aggregate PASS/count claim.
Long command rejected at admission; split into bounded commands.
Existing failure index and fixtures reused; no new harness asset selected.
Continue actual remaining Peer/audit/history/retirement/dependency and
consumer conversion, then aggregate checks and joint publication.

## Historical phase status (superseded)

| Phase | Box / development status | CI / independent status |
| --- | --- | --- |
| P0 | COMPLETE — preflight/inventory and L1 rehearsal; precise L2 native capability blocker reproduced | Exact merged BDH source/quality/dual-arch PASS; full native S2/S2.5 remains OPEN/BLOCKED for P1/P2/P5 |
| P1 | P1a complete; public WAN BOX_VERIFIED, alternate pool BLOCKED; P1c/d source separation underway | 92ffeda 7/7 workflows, N2 27/27; new source BOX_VERIFIED/CI_PENDING; hosted WAN UNVERIFIED; P1e open |
| P2 | NOT_STARTED | NOT_RUN |
| P3 | NOT_STARTED | NOT_RUN |
| P4 | NOT_STARTED | NOT_RUN |
| P5 | NOT_STARTED | NOT_RUN |

## Initial checkpoint (historical)

No development Box selected, no current installed binary observed, no native Operation performed, no code changes or native tests performed by initial planning task. Existing RC environments may be disposable with exact ownership/cleanup checks; preserve uncertain originals. Before work, read actual PR HEAD and comments, discover authorized existing Box and its operations, reconcile dirty work/checkpoints and pending CI, and select the next safe unfinished slice. Do not infer stage or environment from this initial record once later evidence exists.

Planning evidence: exact source anchors in inventory; 47 workflow files, 54 help-listed CLI leaves (not full dispatch inventory), Schema 6, four installed CI modes. S2 and nested release tests need startup conversion; Scoped still uses shared guard/conntrack/native helpers. No full dependency or coverage proof yet. No current stable publication. #170 is separate and must be reconciled after consolidation.

## P0 checkpoint — Box workspace and dependency reconnaissance (2026-11-24 PT)

Source examined: `1cfcd3da43bc47dfc42edac58d0041c02aa16e56`; canonical v1/base `9d07665dcfc5ae6345d6bfcd0d71de7b6510b51d`. No product source change yet. The exact original PR head's [quality gate](https://github.com/vanishs/boxweave/actions/runs/45565927118) succeeded (job 139040580967). The [candidate run](https://github.com/vanishs/boxweave/actions/runs/45565927228) succeeded only on change scope (job 139040581939); source, amd64/arm64, S2 and S2.5 were skipped. No product CI failure or product CI PASS.

| P0 slice | Status / evidence |
| --- | --- |
| P0a — resume and exact baseline | DONE: no previous PR191 operations found before this task; existing Boxes inspected; initial head and docs-only CI checked. |
| P0b — isolated workspace | DONE: reuse `pr188-h0-lab` UUID `9d374e18-196a-493d-bf1f-2d582589f04d` (L1); separate clean clone `/root/pr191-consolidation` at exact head. Existing dirty PR188 checkout preserved. |
| P0c — candidate and source checks | PARTIAL: Go 1.26.9 recovered from `/root/pr188-h0/toolchain/go/bin`; CGO amd64 candidate built with SHA256 `8380c32c2b05bb8fc0141a64d26ba1a6d9b62156abf6050916d956f85c529d50`. Original build Exec then exited 1 on unsupported `version`; subsequent original `build-info` and three-package test Exec still VERIFYING, not PASS. Make absent from PATH. |
| P0d — dependency and acceptance inventory | PARTIAL: concrete runtime/schema and installed-lane anchors below; complete per-symbol and per-marker witness mapping, dynamic callers, all workflows and actual CLI leaves still open. |
| P0e — exact Scoped native baseline | OPEN: fresh install, authenticated positive/negative, Create/Peer/Delete, restart, retirement/reinstall on exact candidate not yet proved. Existing roundtrip lab was inspected read-only and is historical. |
| P0f — conversion rehearsal | OPEN: prove S2/S2.5 Scoped boot and fault-injection fixture setup; preserve all generic witnesses and isolate genuinely obsolete behavior. |

Original operation record (work_ref `pr191`):

- `b032519267e68e6287ba7a0022c71a14` (exit 0): inspected roundtrip lab; no Git/Go in PATH, active Incus and BoxWeave, no listed instances.
- `40aa0bbf1b88e4d0dcdb205cd67b8852` (exit 0): inspected scoped lab; four running historical fixtures, left untouched. `f280fd321a15bcad20a36f89f3404a64` (exit 0): inspected developer lab; Incus active, BoxWeave inactive, no listed instances, Git available.
- `49637409c66eddbb88147b4c95284b20` (exit 0): old PR188 worktree dirty; no existing PR191 path found. No reset or cleanup.
- `99dfa577f859aeca4c7301ba2cf21bbb` (exit 1): cloned exact clean head successfully; preflight stopped at missing Make. `b9c6bba830ba82cb160265da9e465958` (exit 0): recovered cached toolchain, confirmed 47 workflow files, inspected mode references and S2 startup.
- `95cafc33de56603c3ce01ab0aff42863` (exit 1): exact binary build and digest completed; invalid `version` command caused overall nonzero. `0af2e5c16cca53d0a8afc9a2eb7173de`: corrected `build-info` then `CGO_ENABLED=1 go test -count=1 ./internal/state ./internal/control ./internal/install`; last observed VERIFYING. Retrieve original exit and both streams before further Exec in this Box; a concurrent read-only plan request was explicitly rejected by the resource lock, not dispatched. Do not replay the original.
- `c9db9c8a6b2d6f9d8ce077bae05d9d0d` (exit 0): `pr188-roundtrip-lab` UUID `c5a60ef1-9ab0-4e05-bfe1-60a74e08b631`, installed commit `269d82074c033e069e930c09720ba6c5860095de`, digest `ddecccebcf83e8e1c74ca95858d0cb9cdba654b8504676aedd1d73ffaf82fa2b`, Scoped mode, all four services active, no inner instances, all six key ledger tables empty. This corroborates historical fixture state, not current-source native acceptance. Preserve prior evidence and do not reinstall this active lab merely to repeat it.

### P0 dependency and conversion queue

The following anchors refer to the exact source examined, not an implemented conversion. Marker counts are unique literal uppercase PASS names (including expectations); they are not unique tests, and dynamic markers/other assertion styles still need mapping.

| Baseline witness / dependency | Required disposition and next stage |
| --- | --- |
| `internal/control/server.go:68,125-155,260,326`; `peer_tools.go:58`; `box_tools.go:416,887`; `operation_tools.go:152`; `operation_recover.go:34`; scoped drain and CLI/install mode consumers | P1 retain Scoped admission, host_status, drain fence, native lifecycle and recovery. Trace each Legacy alternative and all callers before removal; convert fixtures alongside code (C02-C05/C10). |
| `network_peer_scoped_execute.go:22,43,65,94`, `network_peer_scoped_root_transition.go:35,48,178`, `network_peer_scoped_conntrack.go:15` | P1 extract/retain shared guard witness/batch, replacement, conntrack, native digest and exact proof. Do not remove by filename; compare actual dependencies and prove directional/recovery behavior (C04-C05). |
| `internal/state/store.go:22,51-53,78-79,272-273,319,404` | P1 trace retirement/queue/compaction readers, writers, indexes, claims and tests. Direct fresh schema bump only with old-state refusal and preservation of original UNKNOWN (C02/C05). |
| `scripts/accept_s2_native.sh` (16 literal direct serve starts at 296, 341, 848, 1103, 1132, 1213, 1251, 1345, 1383, 1570, 1679, 1950, 2014, 2193, 2378, 2443; 113 literal PASS names) | P0/P1 convert setup and repeated restart/fault process launch together. Preserve previous official SDK probe at 357-382, project/UUID denial, duplicate serve, actual Exec/output, crash/WAL/ACK loss, stale ETag, GC/quota, schema refusal. The protocol's legacy SDK is not Legacy controller support (C02/C03/C05/C11). Explicitly map original legacy SimWAN fixture at 2608-2614 to equivalent Scoped witness before retiring it. |
| `scripts/s25_bridgehead_lab.sh` (38 literal PASS names; old install-stage/activate at 95/100, hard-coded build labels at 64/66, positive upgrade/rollback at 557-608) | P0/P2 convert startup to supported Scoped install. Retain actual two-layer lifecycle, distinct actors, old-token denial, restart/SIGKILL original recovery, foreign and parent socket denial, v4/v6 positive/negative packet evidence. Separate old Legacy upgrade/rollback positive behavior from generic recovery assertions; explicit Scoped unsupported-operation negative proof and #170 decision where applicable (C03/C05-C09/C11). |
| `scripts/n2_installed_lifecycle.sh` (24 literal PASS names) and `.github/workflows/v1-n2-installed-lifecycle.yml` (49 literal PASS names; four modes at 72-77) | P0/P2 preserve clean install/read-only/stale review/guard, unique Legacy-hosted generic active/offline retirement/resume/purge, Doctor/Recover, capacity and network churn. Retain scoped roundtrip and directional lifecycle witnesses from `pr188_scoped_roundtrip.py` and `pr178_n2_scoped_lifecycle.py`. Do not delete Legacy lane before each applicable marker has a replacement or explicit obsolete disposition (C04-C08/C11). |
| `.github/workflows/v1-release.yml:194-195,229-231` | P0/P4 retain independent S2 and S2.5 native gates and dual-arch manifests; use exact candidate artifacts and update required witness checks only in step with converted scripts (C01/C09/C11). |
| `cmd/boxweave/main.go:174,189,240,260-290,476-508,561-571,663-738` | P3 categorize real dispatcher and fixed service leaves, including hidden Scoped guard/lifecycle commands. Native Incus package migration is separate from obsolete BoxWeave RC compatibility; do not blanket-remove by prefix (C08/C10). |

Next safe action: first read original test operation above; then inspect developer lab's actual residual installation and prerequisite state read-only and generate exact candidate Scoped plan. If this retained lab cannot host a clean candidate without removing other resources or bypassing a safety refusal, use a separate authorized disposable fixture or independent native runner. Complete the full witness ledger and Scoped boot rehearsal before broad P1 deletion. No false CI or native PASS.

Learning disposition: reuse BDH-L001/L002/L006 (tool and fixture capability), L012 (exact CI evidence), and existing verification/resume skills. Recovered nested Go toolchain despite PATH failure; unsupported `version` was corrected to existing `build-info`. No new reusable asset or promotion justified yet; keep failure and original pending operation in this record. No merge or publication.

### Follow-up: source checks and clean native fixture

- The original test Exec `0af2e5c16cca53d0a8afc9a2eb7173de` reached SUCCEEDED with actual exit 0. Build-info reported `1.0.0-dev.pr191-p0` at `1cfcd3da43bc47dfc42edac58d0041c02aa16e56`. All three targeted packages passed: state (146.500s), control (18.513s), install (10.023s). This is focused source evidence, not full race/CI or installed-native proof. The initial candidate's SHA256 remains as above.
- `8c629ff69b745c9aa940e855fbbc969a` exit 0: actual exact-head tracked-source grep and formatting. No gofmt output; `scripts/check_format.py` passed 1,145 tracked files; diff check clean. Enumerated 47 workflow paths and 244 Go test files across cmd/boxweave (9), control (53), install (84), state (98). This is a reference/format inventory, not all tests executed.
- Shared helpers: `NativePeerGuardPlan` appears in 21 files including fresh admission/bootstrap/kernel and Scoped boot, guard, recover, retirement and root-transition; `nativePeerGuardReplaceOnce` in 5 including journal/transition and Scoped direction/execute; witness/batch types live in shared guard transition/journal. `nativePeerConntrackRun` spans IPv4 retirement, peer conntrack/store and Scoped execution. `nativePrepareStateDigest` spans native prerequisite/retirement and 14 other network files. Keep these helpers and regressions before retiring callers.
- Schema dependencies: `network_ipv4_retirements` is referenced by address holds, audit, Create binding/recovery, deleted binding, pin repair, Scoped audit, maintenance, store, retirement tests and installed script; queue in maintenance, retirement, Scoped audit and store; compactions additionally in uninstall_core and its tests. Therefore a schema-only deletion is not safe; first convert those readers/writers/observers and their tests in the same coherent P1/P2 slices. Maintain current address holds and explicit old schema rejection.
- Exact developer-lab plan attempt `02c9f3c3703198d2fcfe6b203ec0ab99` exit 1: existing /var/lib/boxweave, installed binary and unit remained; Incus 7.6 active, no native instances listed; fresh wrapper refused existing installation. Do not clean, upgrade, or overwrite historical developer lab to force native P0.
- L0 host_status reported ready and existing resource capacity; created a separate authorized nested test Box without explicit privilege/AppArmor/interception overrides. Original Create `e1c1ca9d0d769aaea7358fcbef02d5b1` SUCCEEDED; independent box_get confirmed `pr191-p0-native-lab` UUID `ad5aeb01-4a88-454f-88b0-9686c67a1d8a`, Running, 2 CPU, 3072 MiB, 18 GiB, nesting/internet true, autostart true. Initial schema validation rejected an unexpected argument before dispatch; corrected to the documented `security.nesting` field. Preserve the new Box.
- New native-lab inspection `f0b0ea4e1571585aacb1d59515012103` exit 0: Ubuntu 24.04, mapped root, inherited AppArmor unconfined, systemd running, no Incus/BoxWeave installation or parent administrative Incus socket at /var/lib/incus/unix.socket. The ordinary guest API socket exists and was not used as a parent transport. This environment can be considered for benign nested functional tests subject to actual native admission; no recursive isolation certification or policy override.
- Original clean-lab prerequisite installation and exact-source clone `9b4882c4aff7154d5dd571cdb6f67a5c` last observed VERIFYING. Command installs standard build and networking packages inside the new disposable Box only, then clones the PR and checks out the exact original source. Read original result and both streams before further Exec; no Incus or BoxWeave installation claimed. Next: verify package and checkout results; transfer the verified original candidate or build it with exact pinned Go, review compatible signed Incus native plan/bootstrap and Scoped fresh install plan in this Box only. Never use hosted-runner environment flags to bypass clean-VM gates. If native admission or inherited policy fails, preserve original evidence and use an independent clean runner.
- The previous documentation checkpoint `fe8a20069dcce33e06a52f093199fc60f81d09dd` has [quality run 45567623847](https://github.com/vanishs/boxweave/actions/runs/45567623847) and [candidate run 45567623806](https://github.com/vanishs/boxweave/actions/runs/45567623806) both reported success; docs-only checkpoint, no current product/native CI PASS. No confirmed CI failure to repair.

P0c targeted source evidence now BOX_VERIFIED (full quality still outstanding); P0d remains partial, P0e clean fixture provisioned but native acceptance pending, P0f not started. P1-P5 remain open as ordered in the plan. No new product code or change to production safety. Learning: existing preflight/verification/resume assets cover the observed dirty fixture and missing tools; no new asset promotion. Keep exact pending original and earlier failure evidence.

### P0 continuation — exact clean installed candidate and conversion map

- Resume original `9b4882c4aff7154d5dd571cdb6f67a5c` exit 0: clean native lab installed standard prerequisites and checked out `1cfcd3da43bc47dfc42edac58d0041c02aa16e56`. No previous effective check repeated. Original docs CI for `cf49a550c15b6601258d1e647ac8fc6cb43f1016`: quality [45567993772](https://github.com/vanishs/boxweave/actions/runs/45567993772), job 139047405187 PASS; [candidate 45567993770](https://github.com/vanishs/boxweave/actions/runs/45567993770), job 139047405376 scope PASS, all expensive product jobs SKIPPED.
- Independent clean-lab candidate build `76f660f1bea9c248bc755fc69ae9696e` exit 0: verified Go 1.26.9 official archive checksum, CGO amd64 exact-source build; digest `8380c32c2b05bb8fc0141a64d26ba1a6d9b62156abf6050916d956f85c529d50` exactly matches the earlier developer-lab binary. Build-info exact original source. Source code unchanged between documentation commits.
- Source audit operations `f783afeca234a86d44ecea71a7de8558`, `c3e6d331fa83be393b1d3fb920ff584a`, `b7b61b6a6558838ab1979046dc42e57a`, `c99818636b8c26fecec23c93ddd054b2` all exit 0. [P0 witness inventory](CONSOLIDATION_P0_WITNESSES.md) generated in the actual Box, 1,293 lines, 63,305 bytes, digest `52fca763bb4526b6c977c26e6840d476ae648199846a3da54470a63084cb4025`, all 47 workflow paths with literal selectors, script references, PASS anchors and runtime/SQL references. Initial oversized Exec was rejected at admission; used exclusive file create `349f208b9bc026fda9e2a278c96087ee`, verified readback and generator Exec `e21a1b75e580e6781cd2d8f773ab8b5b` exit 0. Existing BDH-L014 procedure reused. The index is a conservative OPEN queue; dynamic and indirect assertions still need disposition before removal. Commit `150144e21aa562490c155df17b3b30b18c556598` has [quality 45568564745](https://github.com/vanishs/boxweave/actions/runs/45568564745) job 139049314579 PASS; [candidate 45568564853](https://github.com/vanishs/boxweave/actions/runs/45568564853) scope job 139049315474 PASS, product jobs skipped.
- Exact clean native fixture `ad5aeb01-4a88-454f-88b0-9686c67a1d8a`: signed package native plan `4c5a998edc39b462f3e68887947f50cd` exit 0, Zabbly fingerprint verified, package `1:7.6-ubuntu24.04-202611072126`, reviewed digest `3e7cd2be7743a671c3a5b279486abf761b66da6e8ad1bca4c1027492e7bea0a8`. Approved exact disposable-lab bootstrap `f43e369a7ae152e4a5ae149032b20ca5` exit 0, new daemon ready, actual client/server Incus 7.6, active, no instances, /21 bridge with 2045 nominal IPv4 identities.
- Scoped read-only plan `798b4e2234662cb3cb1f39aa853edac3` exit 0, restricted user-1001, no installation state created; candidate digest above, native review `f1246cdb2aeea0ed658097dc840b1302b54143892676a56fac31be974d2bf1d7`. Apply `fbb2a1233623189b7f2dfc86392436e0` exit 0: intentionally stale digest refused without installed artifacts; then exact reviewed fresh apply succeeded. Four services active, manifest Scoped, installed binary hash and build-info exact, no restart required. Recognized inherited LXC policy retained unchanged; not a recursive isolation certification.
- Auth/image operation `e6572831c40402c61c526ca16476955b` exit 0: real loopback anonymous and invalid bearer HTTP 401; temporary authenticated actor discovered exact count 23 tools, ready host_status schema 6 and exact commit, unknown UUID denied. Root Doctor SCOPED_V2, VERIFIED, kernel guard OBSERVED. Real Ubuntu 24.04 image fingerprint `990054afd62769257b8d386635a09182191ec230480bcf99267574d186209bf5` imported to exact inner Project. Credential receipt stays private in the disposable lab.
- Original native N2/N3 Exec `4987d4f1104de7b45428411fcad5dff3` running existing `pr178_n2_scoped_lifecycle.py` and `pr178_n3_scoped_peer.py` in clean candidate lab, last state VERIFYING. Do not duplicate or assume PASS; retrieve original output/exit before further effects. Next: verify lifecycle, directional peer/retained flow evidence, exact controller restart and original receipt readback, then reviewed normal retirement/reinstall.

#### Required conversion sequence and structural dependency decisions

1. P1a first extract neutral shared primitives without changing behavior: `nativePeerRootInstalledDB` in native_peer_root_service.go:22; `nativePrepareStateDigest` in native_prereq_prepare.go:259; guard plan in network_peer_scoped_guard.go:32; witness/batch and replacement in network_peer_guard_transition.go:99-114; conntrack report/run in network_peer_conntrack.go:23-37. Keep exact socket UID/identity handling, fixed DB and native scope verification. Retain all existing shared regression tests.
2. P1b convert real Scoped runtime callers and fixtures together. The fixed Root lifecycle server in network_peer_scoped_lifecycle_socket.go:47-138 derives scope and opens the installed DB read-only, not an arbitrary caller-supplied DB. S2 main has 16 direct starts; 10 additional direct-start fault harnesses: s2_p0_native_sqlite_fault, s2_p1_bound_create_receipt_fault, s2_p1_native_create_delete_overlap, s2_p1_operation_list_boundary, s2_p1_private_read_inflight, s2_p1_stop_delete_fault, s2_p1_token_inflight, s2_p1_two_controllers, s2_p2_alternate_pool, s2_p2_malformed_exec_fault (all .sh). Convert normal/duplicate/restart startup first. Alternate DB/ACK proxy fault families need independently initialized exact-scope installed fixtures and test-only fault injection at the correct layer, not arbitrary Root DB flags or a legacy server behind the tests. Check original effect and receipt, no redispatch. S2 old SDK probe has an actual hosted-runner admission gate; do not spoof it locally.
3. P1c convert SQL users before deleting tables. Concrete references: address holds lines 63,129; Create binding 75,86; Create recovery 56,66; deleted binding 19,50; pin repair 68,89; Scoped audit 96-98,129. Preserve retained addresses, fences, binding/pin repair and explicit same-schema old-state refusal. No migration or address reuse. Maintenance and uninstall observers must be converted before retiring shared table shapes.
4. P2 use actual proven fresh plan/activation for both S2.5 layers, not legacy native-prepare+install-stage/activate. Outer s25_bridgehead_lab.sh:81-100 and inner s25_inner_setup.sh:116-151 need exact installer bytes/manifest transfer, review, matching digest and guard/helper readiness. Preserve image/relay/foreign and packet witnesses. Old positive upgrade/rollback (outer 557-608) is an obsolete-only candidate, not grounds to drop original restart and operator/auth checks. Scoped upgrade/rollback refusal and separate #170 future feature decision. Current offline Legacy retirement is not proof of supported Scoped offline behavior; P2 implements/validates the latter.
5. P3 actual dispatch allowlist has 56 leaves (cmd/boxweave/main.go:174), versus 54 help leaves. Hidden `native-scoped-guard-boot` and `native-scoped-lifecycle-serve` are installed fixed unit callers. Keep until replacement is proved; classify CLI/public/internal and native Incus lifecycle separately. Live dev index already links task docs; history retains exact old evidence. P4 review workflow dynamic witnesses against the generated index; final C01-C12 crosswalk requires actual results, not index counts.

Learning: existing environment, verification and source-transfer skills handled missing tools, dirty fixture and bounded transfer. No new BDH asset promoted. No product change, host-policy change, merge or publication. P0 remains IN_PROGRESS until native original results and the remaining rehearsal are resolved.

### P0 installed workload, restart and retirement evidence

- Original `4987d4f1104de7b45428411fcad5dff3` exit 0, both existing N2/N3 scripts unchanged: boot order, Scoped MCP Create/Delete, busy/replay, retained addresses, full lifecycle, directional IPv4/IPv6 live flows, reverse/unrelated connections, active-policy lifecycle, regrant exact idempotency replay and Root helper restart/boot all PASS. Four real running test instances n3-a/b/c/d. Example originals Create `89a37ca7719148786733c983b58930e5`, Delete `3d58e90994deb9458bb7aaf15e90c1f5`, directional grant `569b0c0e4d16352ab63c7eaf1a24cbf6`, revoke `931db7438d3e01fcb9d8e4dbddd85486`, regrant/replay `2e517b281c943d612e1bdb1407ca45b6`. All SUCCEEDED in actual inner controller.
- S2 startup/restart rehearsal `8f3a077e2892d48975ea59aa699a058c` overall exit 1 due test readback script incorrectly indexing the already-flattened operation reply. Actual negative legacy-mode omission refused `scoped journal requires scoped controller mode`; exact same DB duplicate serve refused `serve already active for this database`; original controller stayed running until explicit controlled service restart. Restart produced new controller PID, original Incus PID and all four guest PIDs unchanged. Corrected read-only `29951f64dde70361ed55790c18efcfd8` exit 0, no repeat restart: authenticated exact ready commit, prior Create/Delete/Peer IDs remain SUCCEEDED, guest inventory identical. This proves current normal/duplicate/restart Scoped startup route, not all S2 fault families.
- Exact retirement plan from that readback: ready, no blockers, four exact owned test guests, three confirmed directional grants, 11 reviewed drain steps, no other instances, review `9878675b279fe3c5fe871fb3e5220819930a1a4c4fc2752e91f6763125eb522c`. Explicit terminal approval within the new disposable test lab only, original `d70b6c28df769b8debf392f876690f72` exit 0, normal Scoped live drain and retirement COMPLETE, actual final post-drain review `9aee8c37fa09e22ce4818a9f280965f4e627b41ce40ddfa2e9612de06e20f08c`. Native inventory empty; Incus PID 6460 and exact inherited LXC policy digest `fcbd512cb8212c6f5720f28398f01e85ff96246aebb73e09cf449d2f3aa2c5b0` unchanged. Installed binary and active state absent.
- Archive proof and separate purge preview `e9995ac268cd958ed72b6434e137986b` exit 0: /var/lib/boxweave-retired/retired-6ddym7ht, DRAINED 11/11, audit COMPLETE, SQLite integrity ok, all 32 original operations SUCCEEDED, all 8 owned Box rows DELETED, selected original receipts readable, archived ledger SHA256 `040594f6a6469051fc03d9dd8abb4fa1bf032f9a7f0bce8acbedbd44207a1b78`. Purge preview ready and irreversible, 268 recognized files, digest `4f391daadea7e976e5c85cbe09aa53d96ad797f4207e3af5d8b87bc698488d65`. No unresolved original or foreign state identified. Actual purge and reinstallation still pending at this checkpoint; no raw private archive or token bytes transferred.
- Further inventory: script anchor scan `b81b16dd7c5a183492afe1d4094e90eb` exit 0 (87 selected script files, 1,055-line [script ledger](CONSOLIDATION_P0_SCRIPT_WITNESSES.md), commit `a644ba69b952787e507598da5cacdd2067c8f166`); Go test declaration scan `8ba3e4ad2aacf29a9b0ea84d8de991db` exit 0 (280 test files, 604 declarations, [Go ledger](CONSOLIDATION_P0_GO_WITNESSES.md), commit `e6a7c9d78a7310df75187364a963c5f26b7419d4`). Crosswalk linked in `e13ae726c9a2229a534a7e18aa8f16928be0e340`. All actual replacements remain OPEN pending P1-P4 review and results. Docs-only CI for 322d9ac, a644ba6 and e6a7c9d passed; no native product CI inferred. Crosswalk docs CI pending at last read.

Next: approve only the separately reviewed completed test archive purge; verify no residue, then exact fresh plan, reinstall, fresh ledger and still-unexpired old bearer denial. Preserve the outer development and native lab Boxes. Learning: corrected testkit reply shape without redoing an effect, reused task resume and verification discipline; no product bug or new asset promotion.

### P0 roundtrip complete; implementation handoff

- Original purge Exec `d299ea88a1c3bf67d577afa619280407` exit 0: reviewed digest `4f391daadea7e976e5c85cbe09aa53d96ad797f4207e3af5d8b87bc698488d65`, PURGED, recognized test archive only, no residue. Incus PID 6460 and inherited LXC policy digest unchanged. Sanitized archive evidence had been committed before purge.
- New exact fresh plan review `6e130e9720dbbd331f65635983e988c55b9a18081e385723ce795d91dd84ff97`; original reinstall and smoke Exec `0c2fd548fa109c13b984a41c9cf80fe2` SUCCEEDED, guest exit 0, both streams retrieved. All four services active, exact original candidate build, six key ledger tables initially empty, still-unexpired previous bearer denied, new actor works and old UUID denied. Actual same-name new-incarnation Create `1b4d64f4029b336b56a826578846b761` and Delete `ebadb2333f2561f84584e698a682402b` SUCCEEDED; no live inner instances, Incus PID and LXC bytes preserved. PR191_P0_ROUNDTRIP=PASS. Stderr is the existing fresh-install Forward default warning, not a test failure.
- Keep isolated native lab `pr191-p0-native-lab` UUID `ad5aeb01-4a88-454f-88b0-9686c67a1d8a`, installed exact source `1cfcd3da43bc47dfc42edac58d0041c02aa16e56`, SHA256 `8380c32c2b05bb8fc0141a64d26ba1a6d9b62156abf6050916d956f85c529d50`. The post-reinstall smoke leaves a retired identity, not an entirely empty current ledger. Preserve the development Box and all unrelated historical Boxes.

| P0 slice | Final Box / source status |
| --- | --- |
| P0a | DONE — exact base/head and prior operations reconciled. |
| P0b | DONE — isolated developer workspace and dedicated native lab. |
| P0c | DONE — two byte-identical native builds, focused state/control/install tests and format checks. |
| P0d | Baseline inventory DONE — 47 workflows, 87 selected scripts, 280 Go test files and 604 declarations, actual CLI/SQL/dependency anchors; semantic per-assertion dispositions and dynamic/indirect consumers remain implementation hold points. |
| P0e | DONE — exact native Scoped install/auth/N2/N3/restart/retire/archive/purge/reinstall roundtrip. |
| P0f | Normal/duplicate/restart Scoped startup and L1 installed route rehearsed. Alternate DB/proxy faults and complete two-layer S2.5 rehearsal NOT_RUN; these must be resolved at P1/P2 before removing old paths; independent acceptance at P5. |

P0 baseline/preflight deliverables and native roundtrip complete; residual conversion rehearsal remains explicitly open. Do not interpret this as full P0 acceptance if the complete fault/two-layer rehearsal is required before P1. No product source change. Next safe implementation work after owner instruction: P1a neutral helper extraction with retained regression tests; do not remove old paths or weaken release gates before the remaining witnesses are established. Documentation-only CI remains separate from native/implementation acceptance; obtain exact latest HEAD CI before phase closure. No merge or publication.

### P0f closure of runnable preflight; blocked native depth retained

Resumed exact PR HEAD `27b1e934303871bcf54c62ec5e0dd07707eea199`: quality [45569938420](https://github.com/vanishs/boxweave/actions/runs/45569938420), job 139053825722 PASS; candidate [45569938346](https://github.com/vanishs/boxweave/actions/runs/45569938346), scope job 139053825730 PASS, product/native jobs SKIPPED. No pending original in ADMITTED/DISPATCHING/VERIFYING/UNKNOWN for work_ref pr191 at resume. Both retained Boxes confirmed Running; read-only preflight Execs `7e502f2fb1df7b558da7bf0c08d5ba9c` and `76236a02e915d52c91fa35b51b379901` exit 0, exact installed candidate and clean developer checkout.

The complete [P0f conversion rehearsal and hold-point record](CONSOLIDATION_P0F_REHEARSAL.md), commit `2de0b697126002b771f05b2b6716107d192dfb69`, records actual alternate-ledger/unavailable-transport no-effect PASS, focused SQLite/Scoped fault tests (including explicit skips), every startup family and the new inner fixture. P0f native L2 bootstrap is BLOCKED: independent Incus package/daemon ready, partial native init exit 1, no managed bridge or configured default Profile, no L2 BoxWeave. Original `c3122bcb96e6d12959d4383e1b14c042` preserved; read-only re-plan refusal verified by `b54562cd18e4bbceb3de0fa36f3741ec`. L2 guest `6b9b5d09-af81-413c-91da-fb3f3fee55bc` retained inside L1 `ad5aeb01-4a88-454f-88b0-9686c67a1d8a`. Do not rerun bootstrap blindly, relax native confinement or destroy its evidence. No full two-layer installed/native or real proxy ACK-loss/WAL-cut acceptance claimed.

| Slice | Latest status |
| --- | --- |
| P0a-P0e | DONE: prior exact baseline and full L1 native lifecycle roundtrip reused. |
| P0f-a | DONE: concrete source/marker routing for 16 starts and 10 fault harnesses, normal/duplicate/restart and alternate-ledger failure boundary rehearsed. |
| P0f-b | BOX_VERIFIED (bounded): local SQLite and modeled Scoped fault/atomicity tests; native accepted-ACK fault fixtures remain OPEN for P1/P2 independent CI. |
| P0f-c | BLOCKED for full two-layer boot: independent inner native initialization failed and stopped safely; precise source/evidence and runner fallback recorded. |
| P0 preflight | Runnable preflight deliverables closed; full nested result remains BLOCKED, not unconditional COMPLETE/PASS. Latest docs CI to verify. |

P1a neutral helper extraction is safe independent work after owner instruction, while the unresolved full native gates stay mandatory before dependent removals. Do not weaken P0/P1/P2/P5 acceptance to claim a green stage. P1-P5 product implementation has not started. Learning disposition: existing resume and verification skills cover exact partial-state refusal; new observation about missing underlying native-init stderr belongs in P2/P3 diagnostic acceptance, not a promoted BDH asset. No host changes, no merge, no publication.

### P0 final preflight closeout — native capability boundary reproduced

Reconciled the actual merged BDH v0.6 HEAD `eb2cddf74e2f5129bebb4dda9cef3a0acd7bfe17` (PR #192 already integrated into #191 by the owner workflow). Developer Exec `80aa43b9f95b46f394c67fcf4d77eac4` and `7fbd8590e4185f53dfda9c327984d9d2` exit 0: exact product source, Go modules, install/uninstall, native S2/S2.5/N2 entry scripts and workflows unchanged. The installed binary remains exact P0 runtime source `1cfcd3da43bc47dfc42edac58d0041c02aa16e56`, digest `8380c32c2b05bb8fc0141a64d26ba1a6d9b62156abf6050916d956f85c529d50`. Exact HEAD CI: [quality 45582511436](https://github.com/vanishs/boxweave/actions/runs/45582511436) job 139095809783 PASS; [S1 45582511508](https://github.com/vanishs/boxweave/actions/runs/45582511508) job 139095809941 PASS; [candidate 45582511623](https://github.com/vanishs/boxweave/actions/runs/45582511623) source quality 139095952493, amd64 139095952755, arm64 139095952808 and manifest 139097811749 PASS. Native S2 139095813678, S2.5 139095814097 and publication SKIPPED by workflow policy. No native acceptance inferred.

P0f investigation: original L2 bootstrap failed before BoxWeave; read-only Exec `be0119e4bbfd64a7b938f7b8f2f17b45` confirmed partial state and original L1 ready. Created distinct diagnostic child `p0f-init-diag` (UUID `8a070060-6361-4e0c-a200-26b970f5b353`, original Create `34d6844cbb5ba4305b212092f937814c`, native ID `77bae653-a0ff-4268-8002-14c6166e627c`, outer Exec `95f012a3401086b50b10d6591198d4cf` exit 0), without touching original child `6b9b5d09-af81-413c-91da-fb3f3fee55bc`. Signed exact Incus 7.6 packages installed under `ef1cf0aeb2b496797259bf376b5e2e8f`; exit 3 was an early service-inactive observation, no init attempted. Existing S2.5 recipe explicitly starts the local daemon. Corrected diagnostic first init `5b816e1ede06db13fc47f18915719bb6` exit 1: native bridge creation fails because apparmor_parser returns 243, access denied to load the dnsmasq profile. Independent poststate Exec `f39517d5ebc32a71ffcb3a45595d12bf` exit 0: both enforced-profile children preserve partial Incus state, neither contains BoxWeave, no inner instances; parent four services, Incus PID 6460 and inherited policy digest unchanged. Full stderr retained locally, SHA256 `f63f1e2e9377132525c562b57034d19a4d26a56b16cbddf31d8b65c71df8b18f`. Exact failure mechanism reproduced in independent diagnostic fixture, not proven on every nesting environment. See [P0f rehearsal](CONSOLIDATION_P0F_REHEARSAL.md).

P0 preflight exit decision: COMPLETE as the planning/inventory/rehearsal phase in CONSOLIDATION_PLAN.md, including actual L1 roundtrip and observed native limitations. This is **not** unconditional native/target acceptance. The full two-matching-candidate S2.5 witness remains BLOCKED in current nested environment, required in suitable independent CI at P2/P5; actual accepted-native ACK loss and WAL cuts remain OPEN at P1/P2/P5. Existing gates and all affected assertion dispositions remain mandatory. No scope waiver, host security change, destructive reset or product change. P1a is the next safe independent implementation slice; P1-P5 not started. No user host intervention is necessary for preflight closure; if independent native CI proves unavailable, request a suitable separately authorized isolated runner, not weaker confinement.

BDH v0.6 actual work history: [begin](CONSOLIDATION_P0_CLOSE_WORK_HISTORY/00000001.json), [handoff](CONSOLIDATION_P0_CLOSE_WORK_HISTORY/00000002.json), and [L012 revision 3](../../dev/learning/BDH-L012/00000003.json). Original work begin `3d396135549add6a65311234da755600`, observe `fd191f42e78c3c2230bcf13b1b2f5a00`, feedback/boundary `aaa4b695248989d375ba6d03c6410731` all exit 0; five asset items FEEDBACK_RECORDED, no issues. Local HANDOFF, not global task finish or promotion. gh absent and connector run records omit raw PR/base association fields needed by bwdev ci; manually verified connector exact-head and jobs without manufacturing a full CI snapshot. Existing L012 observation linked, no L014 transfer issue on this attempt. Native initializer error detail lost by runLocal (lifecycle.go:38-44) is a concrete P2/P3 diagnostic requirement, not a new BDH asset.

## P1a — shared native helpers, Box-verified source checkpoint

Resume base `179b0efd45989add97c3a3038bf0f8c58893ff41`. P0 final quality run 45584339001 job 139101995483, S1 run 45584338991 job 139101993891 and candidate run 45584339006 source/amd64/arm64/manifest jobs PASS; release native S2/S2.5 skipped by policy, not accepted. Developer Box `9d374e18-196a-493d-bf1f-2d582589f04d`, checkout /root/pr191-consolidation, Go 1.26.9. No installed controller or host changed.

P1a extracts fixed ledger/name and preparation/profile/ACL/digest/SDK primitives to native_scope_common.go, exact grant bindings to network_peer_binding.go, guard plan type to network_peer_guard_transition.go, shared conntrack report/read/OS/address primitives stay in network_peer_conntrack.go, stop-fenced bidirectional path explicitly separated into network_peer_conntrack_legacy.go, and exactPeerRuleKey moves from revocation to common policy compiler. Twelve files, no runtime dispatch, schema, constants, authorization, algorithm or test removal. Read-only declaration comparison of the initial eight old/eleven new files found 75 identical top-level declarations and 24 relocated; final rule-key move independently checked byte-identical. This is a bounded textual comparison, not proof of native equivalence. Existing compiler/guard/purge tests retained.

- Compile-only original `27487bec36beb5c71213f07adb730edb` exit 0. Full source gate original `7093aa4a1aad520bd196a44bded5adb5` exit 0: format 1167 tracked files, YAML 47, tidy/verify, vet/build and CGO Go race all 11 packages (state 163.489s). Subsequent exact rule-key relocation original `686909479c83de5e0f351d5c629fec07` exit 0: install race 17.970s, format, Python suites 16+57+10+73+7+6 tests PASS. Expected negative MCP probe message occurs inside passing test suite. These are local source tests, not real native or installed acceptance. Exact tested file digests recorded in `dc538ef24b8b818774ac77a4f6e2b40e` exit 0; publish those exact bytes and independently recheck CI and checkout.
- Oversized inline edit rejected pre-admission without receipt. Verified exclusive script file `f28d6c41407c92c3577a43c4b928d8c7`, 3478 bytes, sha256 f9814392a76529a7c5ee71c725838324be7f863ebc222f06dc25d5b6a2cfb9b9; short Exec succeeded. BDH feedback original `a412c14b59bd18d884e0d247fa11a88f`, `dc538ef24b8b818774ac77a4f6e2b40e` exit 0; [L014 revision 10](../../dev/learning/BDH-L014/00000010.json) and [handoff](CONSOLIDATION_P1A_WORK_HISTORY/00000002.json) FEEDBACK_RECORDED. Reuse only, no new asset or promotion.

P1a source verified, independent CI pending on publication. P1b first convert ordinary Create/Delete/Peer fixture startup and keep original positive/negative witnesses. Current Create branches at box_tools.go, peer at peer_tools.go, Delete verification and dependent cleanup plus server readiness/retirement loop still select mode. Do not simply set ScopedPeerFresh true on bare test DB or delete old branches before replacement fixtures. P1c legacy paths remain, P1d schema six unchanged; shared retirement SQL consumers must be converted before table drop. P1e real ACK/WAL and native regression required. P0 nesting capability blocker remains; no host policy changes.

## P1b — Scoped common source fixture and replay checkpoint

- Parent `a1e73da0e600b3e8d22c0082fb66c4d5d0b66944`: all five triggered CI workflows passed (quality 45586132212, S1 45586131942, candidate 45586131993, Forward 45586132079, N2 45586131910). All four N2 installed lifecycle modes and three Forward modes passed. Candidate source/amd64/arm64/manifest passed; release-native S2/S2.5 and publication skipped by policy, not accepted.
- Existing isolated development Box `9d374e18-196a-493d-bf1f-2d582589f04d`; Go 1.26.9. Converted seven common MCP source test files to Scoped fixture and user-1001 project, with deliberately absent native socket. Real SQLite and official SDK loopback; not installed/native proof. Existing native fixtures untouched.
- Original full control test `56475d13ccae7814f0c983025d85500a` exit 1 exposed Scoped missing-peer classification difference in TestRC8AgentVisibleSchemasAndSafePreflight. Map only not-found/not-owned to existing fail-closed peer scope refusal; other errors unchanged. Focused correction `524b10f80cd8b8dbfe00636f9c11506f` exit 0.
- Added TestP1BScopedMCPReplayWithoutNativeDispatch: exact v2 Create/Peer SQLite history with synthetic native IDs, same-key original SUCCEEDED/UNKNOWN receipts, changed intent conflict, independent actor refusal, foreign/missing UUID indistinguishability, no new admission/effect when native evidence absent. Exclusive source transfer `6d119c32d5954c9200112a6823259cb5`, exact 6207-byte readback; gofmt/test `a613d026dc4e98e64cfb1d37720adb2e` exit 0.
- Full CGO Go race (11 packages), format (1171 files), YAML (47), vet/build/tidy/verify `2deecfb4b7bf3282aeb69b0f73e024da` exit 0; six Python suites (169 tests) and exact nine source digests `28ff8187ca2a2654dd6ef487779b56c9` exit 0. Source slice BOX_VERIFIED; independent CI pending for this commit.
- BDH L014 revision 11 proactively reused exclusive source transfer. Initial feedback `c6afa902cbd1d06c4b8a95904ebd1fa1` required review because paraphrase differed from exact observation and L012 disposition was not explicit. Append-only corrections first refused reference/fact replacement; explicit correction review preserved original facts and linked exact observation. `2d88e9945cde2ac0e16de275a1a6e52a` exit 0, FEEDBACK_RECORDED; [history](CONSOLIDATION_P1B_SOURCE_WORK_HISTORY/00000003.json). No promotion.

### Next P1b native fault slice — open, not passed

The original TestN3NativeMCPCreateACKLoss expects SAFE_RECOVER with no fabricated native ID using an alternate one-use DB and legacy n3AckMCP. checkRecoveryCreateTx rejects schema !=0; readScopedCreateOriginalTx requires VERIFYING and native ID. scopedLifecycleRequestKind accepts Create/Delete only in VERIFYING; ReconcileApproved currently handles Peer UNKNOWN only. Delete likewise requires exact VERIFYING/native ID and verified original Create journal. Scoped Create journal covers ACL/boot/egress stages; controller separately starts and verifies final state. Partial quarantine or missing ACK is not a legacy static-lease success. Preserve old fault tests, original UNKNOWN records and claims. Design equivalent read-only exact-original/root-journal and final native poststate evidence; never forge an ID, reset phase to bypass the gate, or redispatch ambiguous effects. Pre-root/partial effects remain blocked unless independently proved under an explicit safe design. Convert real accepted-ACK/WAL fault fixture to clean installed Scoped ledger, without alternate DB override. Preserve TestB5NativeDeleteAckLoss and Forward/ownership witnesses until equivalent coverage is proven. P0 L2 AppArmor blocker remains; no host security changes. P1c dispatcher retirement and P1d schema removal remain gated by P1b evidence; no merge/release.

## P1b — Scoped completed-journal lifecycle recovery checkpoint (2026-11-25 PT)

Resumed exact parent `ec27a109aee9302565a2216c04dd3dae83a63878`, integrating #193 BDH updates without runtime change from the prior common fixture checkpoint. All five triggered exact-head workflows completed: [Forward 45746339232](https://github.com/vanishs/boxweave/actions/runs/45746339232), [quality 45746339239](https://github.com/vanishs/boxweave/actions/runs/45746339239), [S1 45746339258](https://github.com/vanishs/boxweave/actions/runs/45746339258), [candidate 45746339240](https://github.com/vanishs/boxweave/actions/runs/45746339240), [N2 45746339326](https://github.com/vanishs/boxweave/actions/runs/45746339326). Four installed lifecycle and three Forward modes actually passed. Candidate two-architecture and manifest jobs passed; S2/S2.5 and publication skipped by policy. Developer Box `9d374e18-196a-493d-bf1f-2d582589f04d`, checkout /root/pr191-consolidation, Go 1.26.9. Resume Exec `e93f8df521ec9a94d15957ef117571d8` exit 0.

New bounded recovery covers known durable native ACK, fully completed matching immutable Scoped Root lifecycle journal, independently proved final native state, and lost final SQLite commit. Original-ID-only read-only Root selector checks UNKNOWN, installed project/guard/quarantine, original proof and credentials, complete journal, exact ACL/instance/binding and idle native operation inventory twice. Restricted controller independently checks poststate. State settles original and separate recovery receipt atomically; scoped Create exchanges quarantine reservation and commits ownership/fences, Delete commits scoped tombstone; no native redispatch or fabricated ACK. Existing normal apply selector, journal format, 23 MCP tools and legacy paths retained. Missing first ACK, incomplete Root journal or ambiguous final state remain UNKNOWN. Reviewed CLI recovery remains a separate P3 concern.

- First compile `6097b339bb9abdc18b0b87b0fa12cd90` exit 1 after moving shared journal type; corrected package qualifier, focused state `02c1af6b55d99dc65c23484983e68194` exit 0. Root selector `58e55f285cd1c286ee57fac84fb978fa` exit 0; recovery/control packages `473da62f4a7dfa368b1b77339c7970f2` exit 0. Added three state tests with eight negative subcases (synthetic Root/native proof, real SQLite), initial ACK absence, atomic fault rollback and replay: `8833091a8f500b2a7d9f26faf7ab0f1d` exit 0. Shell/Python syntax `d2efeed98312bec1e9ce9cfbafa664a6` exit 0.
- Registered format `6687e1b32b95b175907bb652eaf98110` exited 1 at inspect because dirty source evidence is UNBOUND; format itself PASS. Full staged source check `591ff87801d81af10f227e0c9ac89bb9` exit 0: 1284 tracked files, 47 YAML, shell syntax, tidy/verify, vet/build and all 11 CGO Go race packages (state 163.418s). All 20 changed source files read back with exact SHA256 and Git blob IDs before publication. This is BOX_VERIFIED source, not native acceptance.
- New scripts/pr191_scoped_lifecycle_recovery.py is wired into existing clean installed Scoped N2 lane, guarded to one-use GitHub-hosted VM, exact installed binary/project and empty fixture inventory. Test-owned trigger faults final commit only, then verifies real native instance/ACL, original operation and immutable journal hashes before/after read-only recovery, repeat recovery receipt and retained address fences. Not yet executed on the new source. No fault injected into retained labs or shared hosts.
- Hold points: actual accepted native ACK loss before Root and WAL process-cut fixtures, independent completed-journal fault gate, generic S2/S2.5 conversions, P1c dispatcher retirement, P1d Schema consolidation and P1e full regression remain open. P0 nested AppArmor failure remains. No scope waiver, host changes, merge or release. Reuse existing BDH exclusive source transfer and verification; work feedback to be recorded separately; no new asset or promotion.

### P1b follow-up — recovered retired lineage and independent native failure

On exact `1c1ca42ebbb913f9c264c77aaeaf08cb0cf44db6`, [quality 45749927224](https://github.com/vanishs/boxweave/actions/runs/45749927224), S1 45749927179, candidate 45749927190 and Forward 45749927207 completed successfully. N2 [45749927198 job 139674970526](https://github.com/vanishs/boxweave/actions/runs/45749927198/job/139674970526) failed after normal Scoped install and Create/Delete lifecycle PASS. Real original Create `fb64457c3e87e6323fbfbb626597cbf4`, native ID `5985edbf-185a-48f9-ae3e-01694666b047`: test final commit fault left UNKNOWN with retained claims, matching completed journal and running native instance; recovery inspect returned BLOCKED / SCOPED_LIFECYCLE_ROOT_PROOF_REQUIRED, native_retried=false. The failure is real, not accepted; exact Root refusal condition unavailable from this log. No original replay or manual unlock. Subsequent one-use fixture emits bounded allowlisted classification of the exact original's Root service refusal; it does not relax the assertion or print arbitrary journal messages. Need next actual CI observation before a targeted product correction.

Independent source review found another concrete issue: retired identity audit only recognized normal completion event names. Recovered Create/Delete must retain exact successful original and separate recovery receipt, matching audit digest, scoped resource/UUID/incarnation and event; missing linked audit rejects. Combined test now recovers both Create and Delete in one ledger and checks retired identity history, then removes its audit in a test-only transaction and observes refusal. All original retired-history negatives retained. Targeted race `f8fffa74621769ccb495b27eb90ff1e1` exit 0; broader state/install/recovery/control race, vet/build and format/YAML `16dcb725db78cac51d2076bb81397bc6` exit 0. CI diagnostic Python syntax and format `8a7195fcca0af8c9f4be15de608a5429` exit 0 after initial missing gofmt PATH `60249c863ee7b805da6a3839d2e5686f` exit 1. The history fix is not asserted to resolve the earlier Root refusal. Exact committed first slice synchronized clean and registered format bound `b3267a706abd83ee0b352840ec4709d8` exit 0. All 3 follow-up source files transferred with matching digest/blob ID. No Legacy dispatcher/schema removed.

Learning: reused exclusive file transfer; attempted feedback while source test claim active was refused before admission, waited for original terminal. Retrospective work begin/observe `e78915796322b88bf2e52de3f7a6be9e` exit 2 (observe: Record directory missing or unsafe). L014 candidate review finding IMPLEMENTATION_CHANGED_OR_MISSING retained; no candidate rewrite or promotion. Local HANDOFF `a263087625822b11410f87d000b1b997` exit 0, REVIEW_REQUIRED, [work history](CONSOLIDATION_P1B_LIFECYCLE_WORK_HISTORY/00000002.json). Further work must reconcile pending feedback without delaying necessary product correction.

P1 remains incomplete at this iteration boundary: native completed-journal gate FAIL (new diagnostic run pending); actual first accepted-ACK and WAL-cut fixtures, old dispatch/schema conversion, clean-DB/old-state refusal, and full native integrated regression remain open. Preserve all prior failures and original witnesses; P0 nested capability limitation unchanged. No merge/release.

### P1b — confirmed native lease project mismatch and targeted correction

Exact parent `11525e80a1a4587f9ec7b4a310ed81566c872e1b`: quality 45750973005, S1 45750972969, candidate 45750973095 and Forward 45750972977 PASS; [N2 45750972997](https://github.com/vanishs/boxweave/actions/runs/45750972997) scoped-retirement 139678385249 and legacy-offline 139678385197 PASS, scoped 139678385208 FAIL, legacy 139678384937 FAIL. Scoped original `6711c83d409e002f665d1062fc4d397a`, native `bbf2b7e1-9d60-4c18-a11c-5aa98b0c7ff9`, completed Create journal and running instance, retained UNKNOWN; bounded diagnostic **LEASE_MISSING**. No redispatch. Legacy job passed earlier stages and positive reviewed plan; actual local reviewed apply then reported idempotency/intent conflict. Generic ErrConflict also covers changed review witnesses, not only existing keys; cause not yet established. Preserve this failure separately.

Developer resume `8c93daa48441b8fc23ae86cd30ecdb0b` exit 0, exact clean source and Go1.26.9; no unresolved original work_ref pr191. Read-only independent retained L1 lab `ad5aeb01-4a88-454f-88b0-9686c67a1d8a`, original Exec `90b7759c711df6dd7aa7e51e700ea70c` exit 0: same managed bridge, root default project lease listing has 2 records, zero static and zero dynamic, neither fixture hostname; owning project user-1001 has 6 records, four static and two dynamic, both retained running child fixtures have exact IPv4/IPv6 static and IPv6 dynamic lease. This confirms the project-filtering mechanism on actual Incus; old candidate not new-code installed acceptance. No lab mutations. Recovery used default-project client for lease check; existing Scoped Peer provenance uses owning project. Correct only completed Create recovery lease lookup to exact original owning Project while preserving Root default/global ACL and instance-inventory checks, dual-stack static lease counts and foreign/duplicate refusals. Unit test verifies correct project and rejects wrong/default, missing static v6, duplicate and foreign lease. Exclusive file `2e70cb92ca561cb97d037777a082313f` 3177 bytes, SHA256 1411c21a55847cdaed3803a3f9e8549ff1d7343b07037128afcd132d959415f5; test `a6452627491548e80bfdaf5d2af03758` exit 0 six subcases; broad gate `eafa8e8e0c612c9d7b0000f36a7c9ae3` exit 0 format 1287, vet/build and install/recovery/control race. Two exact source blobs read back and compared before publication. Independent installed recovery still pending.

Next safe work: actual first accepted native ACK loss with pre-Root quarantine/retained ACL and restart, real final-commit recovery CI; do not remove old path or reinterpret a blocked outcome as success. Legacy review conflict investigation remains open. No merge/release or change of product safety boundary.

### P1b — actual Create recovery PASS; dependent Delete and first-ACK native fixtures

Exact parent `16364fb1a3ac5e33acded746c85c1ceb5ec8ca8c`: quality 45751997474, S1 45751997487, candidate 45751997423 and Forward 45751997588 PASS. N2 [45751997442](https://github.com/vanishs/boxweave/actions/runs/45751997442): legacy 139681725367, legacy-offline 139681725615 and scoped-retirement 139681725832 PASS; scoped 139681725625 FAIL after actual **Create final-commit recovery PASS**: original `11763d1b7bc8a71c160387ee7f462309`, native `b6638a58-f44e-4ddc-8370-52352dbd5e8a`, recovery `ee79155687a19705f4c97bf670887746`, UUID `28f4ca57-a075-4bb5-b519-51f790c0b575`. Exact original/UUID, immutable Root journal, running state and same recovery receipt verified without native retry. Prior lease mismatch corrected. Subsequent Delete original `fac258990f4fe9d3fcf37d518ce68ac6` FAILED with DELETE_REQUIRES_STOPPED before fault injection, no native ACK. Fixture omitted mandatory stop for no-traffic Delete; add normal stop and exact native stopped/UUID witness, never relax product rule. Prior legacy reviewed conflict did not recur on this run; cause remains unconfirmed.

Added one-use GitHub-hosted exact installed Scoped Create/Delete first-ACK durability-cut lanes, retain four existing lanes and require independent PASS markers. Fixture trigger aborts only original native-ID write for exact action/work_ref; native effects execute normally. Check original UNKNOWN, no fabricated ID, no Root journal, Create quarantine/stopped/boot-off and five reservations; Delete absent instance with retained per-Box ACL, owner/pin/fences, actual dependent Forward and Stop child receipts and listener closure. Check blocked recovery receipt replay, original same-key replay, fresh-key refusal, exact parent/children evidence and controller-only restart. No adoption, release, replay or cleanup of uncertain originals. This is accepted-native-effect durable ACK loss with restart, **not** actual process death during uncommitted WAL transaction. Existing WAL/legacy witnesses retained. Developer Box `9d374e18-196a-493d-bf1f-2d582589f04d`: transfer `4f57038c53fca222e656f543981ca3f7` 13203 bytes SHA256 0cd31c149e0ecf689a3919aba1ec34ce97baecf31ffa5ed3137e2ab2502245f6; initial check `b8b07c1bcbc01a532dd58bb7568981cf` exit 0. Read-only L1 `bac154adf0380644f3d9b5c7d6b1f077` exit 0 confirmed Guard is active oneshot with MainPID=0; compare service active identity instead of PID. Source read `3772e4535fcd35a22cb42637ea8ab89b` exit 0 confirms Delete with dependencies dispatches children sharing work_ref; count parent by exact action and preserve child receipts. Final staged source/format 1288/YAML47/shell/Python compile `c93c7452e186a62dff118a402631b959` exit 0. Exact four file blobs and digests verified before commit. New native gates CI_PENDING; no claim of P1 completion. No host effects, product rule changes, legacy removal or release.

Learning: previous local inbox read failed because copied learning history lacked sibling episode history. Actual canonical source is docs/development/BDH_V07_EPISODES, not dev/episodes; attempted wrong source `daf8026cb557e432066f0ab7c56e3872` exit 1, read-only path check `2ecd9fc168c6ffc32f7d6e0811e669ce` exit 0. Preserve prior revision and resolve using exact copied history, no asset rewrite. Further P1b native results, uncommitted WAL process-cut acceptance, generic S2/S2.5 fixture conversion, P1c and P1d remain open.

### Feedback reconciliation and committed-source correction

Exact source `842678c9c32ee791e533dd3a2052d3b9cabef32f` synchronized to the development Box and clean at Exec `688a321667cd2be18e1d1303fb192839`. Registered format failed because the newly appended progress record contained trailing whitespace; direct diagnosis `e9789da336a75c6a46f9bf4356c7cdb2` confirmed the documentation error, not a product code failure. Remove the trailing whitespace; keep the failure evidence. Correctly copy canonical episode history alongside the local inbox and append exact L014 observation, `64e2136da624a0927c11580d9df46207` exit 0; prior work feedback revision 3 `430ae7a5e8823802dbb51c963b525756` exit 0 FEEDBACK_RECORDED. Persist [L014 revision 40](../../dev/learning/BDH-L014/00000040.json) and [work history revision 3](CONSOLIDATION_P1B_LIFECYCLE_WORK_HISTORY/00000003.json). No skill promotion or asset modification. Native evidence still binds its actual tested commit, not this documentation correction.

### P1b — first ACK/final-commit native CI PASS; SQLite process-cut source verified

Exact parent `d1f4d305c1e552d6723abdb5a68dae39917334b2`: quality 45753057756, Forward 45753057709, S1 45753057678, candidate 45753057777 and [N2 45753057708](https://github.com/vanishs/boxweave/actions/runs/45753057708) SUCCESS. Candidate source, dual architecture and manifest actually passed; native S2/S2.5 and publication skipped. N2 six jobs passed: scoped-delete-ack 139685141758, scoped-create-ack 139685141944, legacy-offline 139685142036, legacy 139685142095, scoped 139685142117, scoped-retirement 139685142138. Actual log witnesses (not command echoes):

- Scoped final-commit recovery: Create original `f29e2e9958cadd48d522eb7b14f0807b`, native `9461c150-aa16-4168-b286-485b17589fcf`, recovery `2eaf07105b4aa7bfef57516ae7312d95`; Delete original `3a2710be1eb34dcbb43dcbab3e9778a7`, native `fc74d264-7ce7-4741-bcbc-dbca5109d18d`, recovery `f19e06534943c7f52df3845db6bcc72f`; same UUID `7ec5fa7e-bd28-4d07-91a6-b752e34eebe9`. Both PASS markers and aggregate PASS, exact original and immutable journal, no native redispatch.
- First ACK loss: Create original `49ae8fb903cb627b2165144e8d875bac`, tracking `041e0aeb04e458145bb7b0354c28a70b`, blocked recovery `adbde43fb6630db93160e6c83905427c`; Delete original `1f399dd9e217f90d25adc18eded316ba`, tracking `e636377c816353126550cbfef3f6938a`, blocked recovery `88824b7b93c21c97d3381d15453f2e38`. Both native PASS, UNKNOWN and claims preserved across restart, no manufactured ACK or replay. Prior legacy review conflict did not recur; root cause not established.

Developer L1 Box `9d374e18-196a-493d-bf1f-2d582589f04d`, checkout `/root/pr191-consolidation`, Go1.26.9, clean exact parent at Exec `5a4be3961e3c829cbe9d436e560d535b` exit 0. Managing controller RC14 `3fb71a6ddf47d13d165e5c039144aa66fad0eb05` unchanged. Added test-only `internal/state/network_scoped_lifecycle_process_cut_test.go` using real SQLite private ledger and own child process; exact terminal-event transaction trigger before commit and controlled after-commit cut. Reopen and Recover check original/native IDs, UNKNOWN/claim preservation and no partial terminal event before commit; successful ownership or tombstone with correct bindings/fences after commit. No native effects from these tests: native receipt is synthetic.

- Exclusive transfer `cc8b0d1427d46653d835e0de78c52d63` SUCCEEDED; compile `e9147f1e16ef9808c7ea86c9d983a1d6` exit 1 int64/int mismatch, fixed with type correction and gofmt. Focused CGO Go race `269f709d09f15b037e4dfa50354a83d9` exit 0, all four Create/Delete before/after cases passed. Synthetic original IDs create-before `e0da90f2f40136a3fda0c940dce9a27c`, create-after `1e79eed6541a3d9809abe477d048692d`, delete-before `ae2ed600ff8d0230e362016b6320320c`, delete-after `86923bd350c6c0e3189653f4f883af23`.
- Full staged source gate `e03072edca8df46f53e057f987f421e1` SUCCEEDED, guest exit 0, complete stdout/stderr: format 1291 files, YAML47, tidy/verify, vet/build and all 11 CGO Go race packages (state 163.756s). Exact source readback 7921 bytes, SHA256 `a1978da9c093641c04ed138a586f19181836e0152c00a9e926ffa120bd846d0a`, Git blob `0f90677d889eae2498a874e24248607e02d7e659` independently matched publication blob. New commit independent CI not yet observed. Historical [checkpoint comment](https://github.com/vanishs/boxweave/pull/191#issuecomment-7182454575) preceded terminal gate read.

Remaining: actual installed Scoped native WAL process cut on fixed installed ledger and exact original, generic S2/S2.5 startup/fault conversion, P1c runtime retirement, P1d schema, P1e integration. Preserve existing BH018 native Exec and P0 Update ACK/WAL before/after, exact original/ownership/replay and independent observer assertions; do not substitute synthetic tests or weaken Root fixed ledger/socket binding. P0 L2 AppArmor blocker unchanged. No product behavior changed or Legacy removed this slice. Learning disposition: reused L014 exclusive file creation/readback; no new asset or promotion justified. Formal work-history feedback still to record. Docs amended through API with changed-line whitespace checks; final exact-head CI must verify combined commit. No merge/release or host effects.

### P1b-05 — exact installed Scoped WAL cut fixtures ready for independent CI

Resumed parent `37e8fbc3079add05bd248fc8981e30c88feb7635`; four workflows SUCCESS (quality 45754285046, S1 45754285019, Forward 45754285038, candidate 45754285039), [N2 45754285062](https://github.com/vanishs/boxweave/actions/runs/45754285062) still running at checkpoint (five jobs passed, scoped 139689122559 pending). Reuse valid prior exact-source evidence; do not claim the pending job. Source unchanged in production Go; no Legacy retirement.

Developer Box `9d374e18-196a-493d-bf1f-2d582589f04d`, `/root/pr191-consolidation`. Verified staged predecessor file blob before fast-forward; resume `975dc809fb7d1f35977c66b5e42cdb87` exit 0, exact clean parent. Existing isolated retained installation and Root/Incus/guard untouched. New test-only C/Python harness extends N2 with four clean-runner modes: scoped-create-wal-before/after and scoped-delete-wal-before/after. Guarded exact installed binary SHA, fixed ledger, Project, empty native inventory, exact controller PID, no prior test drop-in or arm. A test-specific nonce in the exact terminal event transaction's dedicated marker table page identifies the WAL generation and commit frame. Controller-only sync interception records its witness and kills that process before or after the actual sync. No alternate ledger, production switch, native proxy, Root authorization relaxation or fabricated ACK. Because startup rejects extra schema objects, create the test-owned marker/trigger only after normal clean startup and remove only those objects after independent pre-restart SQLite/native/journal evidence, before normal restart. Retain original ledger rows and all unresolved effects on failure. Pre-sync process interruption is not power-loss simulation: either complete durable commit or original VERIFYING before restart/UNKNOWN after restart is permitted, never partial ownership/fence/tombstone. Post-sync requires complete commit. Check exact original and native IDs, tracking, journal hashes, no duplicate dispatch, atomic state and separate read-only recovery when needed. Preserve all six existing N2 jobs and original S2/S2.5 fault witnesses.

- Exclusive source operations `c26a4cd83ae659978af1bee541583d48`, `7b5b27cb4b81f6818d0eb548452f8a91`, `e43a609877aa764562466fa64e908d09`, `2b37f339f02c2dccaeb1a67ecc4e1034` succeeded. Patch/compile/syntax `95b606b716988834da54da0785eb33b9` exit 0. Real private SQLite WAL marker selector tests `16c60018883f18e7487eaf877dca2ab5` exit 0, three tests (exact page/commit, unrelated/rollback, malformed or incomplete WAL and checkpoint generation). C compiler warnings treated as errors.
- Full staged source gate `2efc75840d917bfbb383e9c1374811a2` guest exit 1: format 1294 files, YAML47, tracked shell syntax passed; Python 329 tests, 328 passed and one failed at unchanged test_v1_release_manifest.ReleaseTests.test_external_installer_plan_and_wrong_digest, whose fresh-plan assertion encountered the existing Box installation. Clean-runner test requires no existing installation; do not weaken or uninstall the retained Box. Prior exact-source quality SUCCESS; new-head clean full gate remains required. No repeat unchanged Go race in Box (previous whole-source gate exit 0). Final targeted C/Python/shell diff/artifact check `a02aabc73004d67eba65c8af20d95773` exit 0; all five exact staged files independently read back and Git blob IDs matched. Local selector evidence is NOT actual installed controller/WAL/Incus acceptance; new four native jobs remain CI_PENDING.
- S2/S2.5 conversion remains open. Next independent slice: convert the existing official SDK real Exec-in-flight/duplicate-serve/restart witness in scripts/s2_crash_probe.go and accept_s2_native.sh:1082-1175 to the same fixed installed Scoped ledger, retain exact nonzero completed output, original native ID, duplicate-serve refusal, same-key/no fresh-key replay and independent one-effect guest witness. Keep all existing legacy/generic tests until their replacement proves equivalent. No C09/C11 waiver, no P1c/P1d until affected gates pass. Learning: reused exclusive transfer/readback, existing source/test boundary and clean runner fallback; no new BDH asset or promotion. Formal feedback record still outstanding. No merge/release.

### P1b-06a — first S2 installed Scoped Exec process-crash conversion candidate

Prior `37e8fbc3079add05bd248fc8981e30c88feb7635` now all five workflows SUCCESS, including N2 45754285062. Exact parent `629a6b49a88640e878e1b2e04642ec9a3fc330ee` [N2 45755039472](https://github.com/vanishs/boxweave/actions/runs/45755039472) all ten jobs in progress at this checkpoint; WAL Create-before 139691574355, Create-after 139691574357, Delete-before 139691574266, Delete-after 139691574313. Other workflows 45755039387/45755039311/45755039319 running, 45755039361 queued. No native WAL outcome asserted. Existing test/CI evidence retained.

Reused Box `9d374e18-196a-493d-bf1f-2d582589f04d`, `/root/pr191-consolidation`, exact source fast-forward with verified staged bytes `0f80f9ae4d025f8e8c008b46d0f27b5a` exit 0. While WAL CI runs, introduced `scripts/pr191_scoped_exec_crash.py` and scoped-exec-crash N2 lane. Reuse existing unmodified official Go MCP SDK `scripts/s2_crash_probe.go` (start/resume); actual installed scoped fixed ledger, existing exact service PID, independent same-DB duplicate serve refusal with explicit Scoped flag, confirmed original VERIFYING and native ID before process cut, changed controller PID, unchanged Root/guard/Incus, persisted nonzero output/exit 7, UNKNOWN with retained native ID and resource claim, exact-key replay and changed/fresh-key refusal, independently read guest marker exactly once. No admin deletion of unresolved fixture; one-use VM retains it until runner disposal. Historical S2 witness/cleanup kept unchanged; no equivalence or native PASS claimed until CI. Exact same product runtime, no new MCP tool.

Exclusive file operations `f4c3f8fc8c5afb7eddf031f301212794` and `c8983da354cf817bc1b5b4879a668968` succeeded. Source gate `30a1110d8ab2a4cb251af1cb6ffa9a55` guest exit 0, format 1295, YAML47, shell and Python compile, official SDK Go probe build. All three changed staged files exact SHA256 readback and Git blob equality before publication. Previous 328/329 offline Python result and clean-install Box limitation preserved; no repeated unchanged Go race or whole Python tests. New scoped-exec-crash native gate CI_PENDING. Next P1b-06b: remaining S2 exact native ACK/WAL (including generic Exec final commit and Update accepted ACK), native transport/ownership/auth/quota/GC witnesses and S2.5 fixed-scope startup/dual-layer gate; preserve original producer/consumer markers and old paths until verified. P1c/P1d/P1e remain gated. Learning: L014 exclusive source transfer and resume reuse observed; no new asset or promotion, formal feedback outstanding. No runtime/host mutation, merge or release.

## P1b-05/06a — WAL visibility remediation and reviewed recovery revalidation

Exact parent `a3cef61f40acd29abe333a02b0ded5274db13257`: quality 45755253962, S1 45755253942, Forward 45755253948 and candidate 45755253994 PASS (candidate source/dual architecture only; S2/S2.5 not run). N2 45755253955: eight of eleven jobs PASS, including Scoped 139692264789, both ACK, both WAL-before, retirement, Legacy-offline and Scoped Exec crash 139692264767. Exec original bfddf8cf020e99e80c31cd9cb1a7f561, native ce9033b7-7044-40f7-8b65-3972dbdfa624, tracking 31b984a3efca1d668b533bd543c78405, UUID fd34b770-6446-4d9b-831f-156f027b572a. WAL-after Create 139692264711 (original 8229c1b583e5e629b85a577bea182a9c) and Delete 139692264385 (original 466ba3837c930aa06e62686d83ad8219) reached actual controller cut then failed the pre-restart SUCCEEDED assertion. Prior run 45755039472 exhibited the same after-sync failures. Legacy 139692264714 passed positive plan/guided decline then failed approved apply with ErrConflict; exact cause not yet proven.

Box 9d374e18-196a-493d-bf1f-2d582589f04d, /root/pr191-consolidation, Go 1.26.9, installed controller 3fb71a6ddf47d13d165e5c039144aa66fad0eb05. Exact parent synchronized by 74fc2d525b2ca1b196247e00e1712cb6 exit 0. Private SQLite reproduction using the project's go-sqlite3 v1.14.52 driver: c110ce4f865c6cf8fdeb9a8c812c218c exit 0; retained reader live=VERIFYING, independent cold WAL=SUCCEEDED, reopened=SUCCEEDED; without retained reader cold/reopened=SUCCEEDED. SQLite WAL sync precedes shared-index publication. This reproduces the mechanism locally, not installed native acceptance.

Repair: release idle read-only SQL connections after startup schema validation; keep O_PATH anchors for original database and parent identity, recheck both around new connection opening, preserve read-only URI/schema checks and close anchors with Store. Identity rechecks are not an atomic adversarial path-swap guarantee. Add replacement-inode/idle/close regression. Installed WAL oracle additionally checks independent private DB/WAL copy with stable hashes, no shared-index copy and query-only reads; after-sync still requires both cold and live SUCCEEDED, exact original/native evidence and no divergent restart. Legacy positive apply explicitly reviews fresh plan and checks stable original/ledger/installation identity before one approval of that exact digest; negative and replay assertions preserved. Legacy timing hypothesis is not a proven fix.

Box evidence: initial Python targeted 4/4 658b387eddef724af41da396a6fc1c60; initial focused Go 3/3 2dad38b43b8a2b550856e1b0d876d3e3; initial full gate 8627906a8cb5d063768414b0e4da1408 exit 0 (format, vet, build, all 11 CGO race packages, Python targeted). Anchor refinement 2797ed0f3ab4ea262688dd23d1b83d05 exit 0 (format, vet/build, state/DoctorD0 focused and install/recovery/control/cmd race, Python targeted). Final O_PATH refinement c4ba7362c8b8ca4d01f85f7e0ca35aa4 exit 0 (format 1295 tracked files, vet/build, focused state race). All six final staged blobs verified by exclusive readback against Git object hashes. Full final race/Python/native clean-runner acceptance remains CI_PENDING; do not substitute earlier full gate for exact final acceptance.

Learning: existing BDH-L014 exclusive file transfer reused after inline argv admission refusal; verified transfers 6c5e6a0ba28055e5cd5bb0f2cf642985, 53d7d8d047e436205dbab4698a99b60c, 6be63b25c977aa25f883155320a04df6, 862beb4bd14adc17c61a90bf9c6159f2. Formal linked feedback/history update pending; no new asset or promotion claimed. See PR checkpoint comment 7182833198.

Next: inspect exact new-head WAL-after and Legacy results, investigate any remaining native discrepancy without relaxing assertions; continue independent S2/S2.5 witness conversion. P1c/d/e and later phases remain open. No installed lab or host mutation, Legacy retirement, merge or release.

### P1b WAL remediation — exact candidate independent revalidation

Candidate `c3dcc9df10afd5df57b6085f7ff3bd52d2f8980f`, [N2 45757512001](https://github.com/vanishs/boxweave/actions/runs/45757512001): all four installed Scoped WAL modes passed with actual controller-cut, native crosscheck and restart markers. After-sync Create 139699502455 original 9efe1ec722b02b9c229e6b7b1d41c89c native 49861c95-1c07-4bb9-80e1-6f7887c5ba1f; Delete 139699502522 original de252d69682987b3ebff6a9e1b772d77 native 4d0d9689-669e-49e5-ae68-a554e8971198. Both live and independent cold views SUCCEEDED. Before-sync Create 139699502525 original ead2fb88aa70ca8dc4343393bbd93910 and Delete 139699502572 original c7ddc6c35fe38d64207d3d20bb82b412 both observed complete commit; process interruption is not power-loss proof. Legacy 139699502587 passed positive plan, guided decline, authenticated postcheck, reviewed recovery/no replay and cleanup. Prior intermittent conflict cause remains unproven. ACK and Exec crash lanes passed; other installed lanes pending at evidence read. S1 45757511785, quality 45757511814, Forward 45757511811 and candidate 45757511876 SUCCESS; no full release-native S2/S2.5 acceptance.

Exact clean Box resume 560c51f5104594ab014787b8e21ea3d8 exit 0; no unresolved original pr191 UNKNOWN in bounded query. Registered recovery-fixture check 2ae0aec1c449df030877f50eed5690f2 exit 0, CLEAN_HEAD_OBSERVED and LOCAL_RECORD_MATCH, output SHA256 b73065662b77e4a7bc5dcb63e0b1d78184cb86fd4c8d27dd8218f8c6feaa9cdb. No new runtime code this evidence slice. Reused actual prior transfer evidence; L014 revision 41 local observation d4dce9b492042829ba16b591e917d201 exit 0. Work boundary auto-link refused changed source 75c5d20eda6b17e724cbf8bf59546e75 exit 2; explicit revision 4 c262546336884fa15517cd50eddbdae3 recorded REVIEW_REQUIRED for L012; explicit correction bcd890a54521beea2f8d0aca36b97326 exit 0 revision 5 FEEDBACK_RECORDED. Preserve both revisions; no asset promotion or efficiency claim.

Remaining: confirm pending installed gates, continue generic S2/S2.5 witnesses, P1c/d/e and P2-P5. No Legacy retirement, host changes, merge or release.

### P1b-06b1 — pinned previous official SDK on installed Scoped

Exact parent `6140fe6b08d7b3da0c9701885a1e331705ee222a` all five workflows SUCCESS: S1 45758305049, quality 45758305048, Forward 45758305034, candidate 45758305086, N2 45758305050 (all eleven jobs). Earlier c3dcc9d N2 45757512001 also 11/11. Release-native S2/S2.5 not established by these results. Original pr191 UNKNOWN lookup empty (tracking 5a1761328e514148af9b5f56c2326a7b). Reused Box 9d374e18-196a-493d-bf1f-2d582589f04d, /root/pr191-consolidation, clean exact parent, Go 1.26.9.

Converted existing scripts/s2_legacy_sdk_probe.go (unchanged) from accept_s2_native.sh:350-382 into an additional clean installed Scoped N2 lane. Preserve the independently pinned previous official SDK v1.6.1, actual 23-tool discovery, negative legacy calls and real Create/Update/Start/Exec/Stop/Delete with exact recorded stdout. Independent read-only verifier binds six exact fixture idempotency key digests to six distinct original and native receipts, common actor/layer/UUID, terminal success, matching retirement lineage, released binding, exit zero, native absence, unchanged controller/Root/guard/Incus and no unresolved operation. No retry, administrative cleanup or new product tool. Original S2 producer and release gate remain unchanged until conversion proven. The old SDK is client protocol compatibility, not a reason to preserve Legacy runtime.

Box independent module build b1dbd46c7a66733d4aedd31ec2bf85ae exit 0, module version v1.6.1, binary SHA256 e36453258855dd14e9ccdf556410b1d8650641728733541987fe3667ac120a2e. Exclusive source patch 4bcdb8f6c39636276e144690fb58d9b2 SUCCEEDED, 10980 bytes readback SHA256 19bf54fe3e886e2a40d84e9a4e4b81b707bf242eef14a6f4e0a1b231d06f2591. Source gate 9ad964460b9c10e0023f7c8f57a1889f exit 0: four private SQLite witness tests (including eight inconsistent receipt/lineage subcases), shell syntax, staged diff check, format 1300 files, YAML47. Local compiled old client and private witness tests are not actual installed acceptance; new N2 SDK lane CI_PENDING. No product runtime changes. Learning: reused L014 exclusive source transfer and fixed existing verification route; formal linked feedback pending; no new BDH asset or efficiency promotion.

Remaining P1b-06: preserve current official SDK native probe/receipts and output/paging/cancel/GC/auth/quota; generic final-commit/Update ACK and ENOSPC evidence; S2.5 independent dual-layer startup and existing native witness mapping. P1c/d/e gated on equivalent evidence. No merge, release, Legacy removal or Host mutation.

### P1b-06b2 — current official SDK shared output and catalog assertions

P1b-06b1 published as `36678d7f5747e44426148142d31b5a2e8f59a740`. Box synchronization 73e83f6cafac0be0950cebdd24a20172 exit 1 exposed only an unintended script mode difference, confirmed by 973e9bf5881d503eee2a6a1d7963bae6 exit 0. Restored original 100755 mode (unchanged blob) in `0826793a96a25abfdfcdc50d21a1311e73f69a43`; clean exact fast-forward a9bc5792ed9fe598654ee0bdfc4b4c39 exit 0. No force push. Original 36678d7 N2 45762727953 and corrected-head N2 45762785063 pending; old quality cancelled as superseded, not product failure.

While CI runs, extracted exact existing s2_native_probe.go discovery/annotation and seven Exec/output assertions into shared functions. Original full S2 entry continues calling both, retains original physical spool rendezvous, all other assertions and final verdict. New hosted installed Scoped subset runs on an independent owned fixture after previous SDK cleanup in the same N2 lane, avoiding another install job. Requires actual official v1.8.0 SDK, 23 tool annotations, recorded nonzero exit, 70K pagination, dual-stream 16MiB read cap, delayed/quiet/continuous guest timeout and cancelled read/late cancel invariants. Ten exact original/native receipts, seven recorded exit codes, distinct failed cancel, retired lineage and native absence independently read from fixed ledger; no full S2 PASS, no physical quota/spool claim. No runtime code changes. Generic final WAL/Update ACK, actual storage/GC and other S2/S2.5 witnesses remain open.

Box patch file 3324278edaefb63d8e58cb3c55ebce7d SUCCEEDED; 11726 bytes readback SHA256 d2bc74e4e2c85e43f5f978811911db9e21740cb21734b6a92c6934591aa4ade3. Source gate 98c351a27e5ab5ae1ef895102a360500 exit 0: gofmt, official v1.8.0 Go probe build SHA256 9d205f33fd0daa76ef83bc5bdce61425362296b067956a8218ec61bb4a35bf91, five private witness tests, shell syntax, format 1300, YAML47. Exact original discovery and output bodies (except moved session-close defer), unchanged original tail independently compared by 6bd4f9c588086ec8b5eb72c680b00999 exit 0. New native CI_PENDING. Learning: L014 reused; include file modes as well as blob IDs in future publication comparison. No new tool or efficiency claim; formal feedback pending. No retirement waiver, merge or release.

### P1b-06b3 / P1d preparation — generic Exec WAL and incompatible schema

Exact `36678d7` [old SDK installed job 139716258759](https://github.com/vanishs/boxweave/actions/runs/45762727953/job/139716258759) SUCCESS, six original/native/tracking receipts, UUID 72146aca-2237-46c2-a8ac-1dd590958709: Create 621d9e2cba522ed5bb1230e80c06f19e, Update 8c2953dba91b8b82cbbf430f62dcd254, Start c17e34cfa87559323cb40d564565b7a3, Exec ad43e7fd76ab1a2fa820968e088dea6a, Stop 75487442fef0330d4ca309c2739a60dc, Delete 0131630496570a428aa4fb1edbb2ff81. Mode-corrected 0826793 job 139716444607 also SUCCESS. Current SDK candidate 4e72199aa144ae4f0d47950f84ff017f046ea10c N2 45763098828 pending; exact Box fast-forward 9363fbb63566e2835d39395ba9b47923 exit 0.

Extend existing installed WAL fixture to generic Exec terminal transaction (before/after sync), preserve four lifecycle modes. Exact original/native receipts, atomic output/event rows or none, unchanged ownership/fences, retained UNKNOWN claim, independent guest effect once, restart/replay/refusal, no UNKNOWN output or fresh dispatch, fresh harmless command only after full terminal success. Post-sync requires live and cold complete commit. No generic UNKNOWN settlement or cleanup. Original BH018/full S2 witnesses retained. Private oracle and selector tests 7/7, source gate f3f16cea5775493de16e593161769007 exit 0, file 52737df7e4bf528d9686a7facaa759af. Native CI_PENDING.

P1d preparation: private real SQLite Go regression five incompatible schema variants across controller, recovery and read-only audit (15 subcases), exact DB bytes and synthetic original UNKNOWN/native/tracking/claim preserved; no archive. Not actual v6 refusal after final schema change; no runtime/schema removal. File 61126a694ae3da471987489466f56618, focused CGO race ce1585260f4c866b1c810351557c9b6b exit 0. Dependency audit 3973f856ae923080e17fb736321fd7e9 identifies 15 state files referencing legacy retirement tables; shared paths need disposition before removal.

BDH L014 revision 42 c8a63e4bf879c96459b6406e78375c90 and work rev6 4c8f598b93e41518c4f77cf287939735 exit 0 FEEDBACK_RECORDED; four transfers and mode correction recorded, no promotion. Remaining current SDK/Exec WAL CI, generic Update ACK/ENOSPC, quota/GC, remaining S2 and nested S2.5. P1c/d/e gated. No merge/release.

### P1b-06b4 — generic Update ACK durability candidate

Exact parent `0842da7b45814d35f01a7ad9d79cbfc86a79b9f2`: S1 45764023268, quality 45764023293, Forward 45764023269, candidate 45764023334 and [N2 45764023332](https://github.com/vanishs/boxweave/actions/runs/45764023332) SUCCESS, N2 14/14. Exec WAL before job 139720482571 original 65a8227c58867de7dcd865ca49be212f native d29c6631-d6bd-428d-9e21-7a1728ee8bcc; after 139720482786 original 8ee8703e3df224dc34debde70daaf3d2 native 3fe4514f-04cc-41c7-bc84-3f06750895a3. Both SUCCEEDED with native/ledger crosscheck, after-sync live/cold SUCCEEDED. Process interruption, not power-loss proof. SDK job 139720482913 both fixed-ledger markers PASS. No full S2/S2.5 or physical quota claim.

Clean Box resume 680b2ccc69bc1662fa4b17741f975743 exit0, exact parent, Go 1.26.9, UNKNOWN query 2039d82d0a3d198e91425eead791ec12 empty. Reuse installed ACK fixture, two clean CI modes for CPU Update: normal ACK event rollback with retained native receipt, and both durable ACK/fallback refused with no durable native receipt. Require exact UNKNOWN original, claim, event rollback, unchanged binding/fences/NIC/UUID/boot, actual CPU change, replay and conflicting/fresh request refusal across restart. No recovery apply or cleanup. No runtime changes. This is SQLite ACK durability, not native transport ACK suppression, ETag or postread loss; retain original S2 proxy/count witnesses.

Patch 28b0256e6bc55f28f0c43e43bb3f2af5 SUCCEEDED, 13198 bytes SHA256 b43f5be823bb497582d02e78e0370ff866ee8ded65a10fa336aaf907389b5259. Gate e080f6e1edafb5602aba54c62d11cf90 Python 14/14, YAML47, py_compile, staged diff, registered format PASS; exit1 only dirty-source inspect REVIEW_REQUIRED. Targeted Go race and format1305 5645ee62dd3aea52a56673b3d1561db8 exit0. Private selector calibration covers rollback, both fallback outcomes, unrelated operation isolation and invalid selectors. Manual targeted fallback (no registered installed ACK check). Two new native modes CI_PENDING. L014 rev43 ace6da94845e1525786c3c7a8fb1166d and work rev7 ca354bcf7f7ec6ded5ef6b80a2f502e2 exit0 FEEDBACK_RECORDED; no promotion.

Remaining transport Update ACK/ETag/postread, ENOSPC, physical quota/GC, remaining S2 and S2.5; P1c/d/e gated. No merge/release.

### P1b-06b5 — full original S2 protocol/packet probe on installed Scoped

Parent `0b0b0fb6267e8892ab4adee6fd083fbd2a0c8ceb` five workflows SUCCESS, N2 45765608697 16/16. Update retained-receipt job 139725666461 original 85edb5640c687e22d6864e840807d2ff tracking 29551dbf98a7e74ad17318ce47e5ed08 native 8c8043e4-1449-449c-96ea-488bc451b3e6; no-receipt job 139725666760 original ade777451dc447e3d225f0722ec55581 tracking 67c262a874390cabddcbaaeb1c33a30d. Both original retained markers identical across restart and durability PASS. Transport loss distinct and OPEN. Resume 3d46a13e5db1d08d94467fe869cf82a0 exit0 clean head; UNKNOWN query 37068d834594e8511edeb14960047fe2 empty.

New scoped-s2-native N2 candidate runs entire existing full official SDK probe on real fixed installed Scoped ledger, unchanged assertion bodies, hosted/exact-candidate guard. Reuse original queue/history seed under exact nonroot service UID and fixed DB, original concurrent append handshake, three real credentials plus revoked/expired, same-name foreign fixture, original simulated WAN, actual spool size observation bound to original, independent fixed-ledger crosschecks. Retain native foreign fixture rather than admin-clean it. No block-device or EDQUOT proof from default pool. Previous subset and release gates retained until equivalence observed.

Patch receipts c5e85ef5d170fde6c619950e0cdc944a (18559 bytes SHA256 c3eeca4db39821ab0db4c4c123d091334d313c2aec839bb1467d185e0db3cdcb), 34d6f35297890ab7e93ae60d62c77f28 (7313 bytes SHA256 cafff8d75744e780cebba834a9da8ef2ee59b40cbd3cca677d6ee687ed280238) verified. Build 3b127bfe005e7132575cbf5fc2b7fb1b exit0, current SDK v1.8.0 probe SHA d032f6ce7517d3e3f82948aa6da11a3ec51d4f4e0e695054da9ad18fe2f1fc4c; seed 478812ed7301361793f8497a0cb9477c4f37edfc10ea8eac18d9e9a6e437c9d5. Private 18/18 tests, shell syntax, YAML47/format1310 b982bf0ed0fd95172525037427682ba3 exit0; bound fixture stage b34118644e481ccb0cc459d1a006b58e exit0. Source only; full probe CI_PENDING. L014 rev44 59627692af788f93fc48d5fd1d33482e, work rev8 4f5b2cec32906dee56a59c2b77ec1b9d FEEDBACK_RECORDED exit0. No promotion.

Remaining exact transport faults, ENOSPC, physical quota/GC, S2.5, P1c/d/e gated. No merge/release.

### P1b-06b5a — foreign inventory refusal and reviewed-recovery diagnosis

Exact c843166fc9662caf6d41be28fa20b590289c4986: quality 45768446860, S1 45768446823, Forward 45768446911, candidate 45768446830 SUCCESS; N2 45768446903 15/17. Scoped full S2 job 139734997208 passed actual catalog, typed negatives, authenticated HTTP private cache, queued cancellation, 120+12 history, Actor/Layer/cursor, revoked/expired tokens and 12-client same-key admission. Original Create 477861e6f4c90e578b58c469cf5ba54e tracking 6f228c8fd58607877172e43bf467ef46 native 261bf888-ce09-4a19-9ebd-aaa853460b4c UNKNOWN (POSTCONDITION_UNVERIFIED). The fixture had a real foreign Project instance before Create; nativeScopedLifecycleInventory requires all-project inventory count equal to owned+pending. This violates the installed Scoped dedicated-host contract; no production guard loosened. Preserve the refusal evidence, move foreign read-only UUID negative after successful owned lifecycle; check ledger unchanged and independent native foreign preservation. Do not replay the failed original. CI_PENDING for corrected lane. Legacy job 139734996695 original 78fe373e73ba75ed76dc53b4cf1fbcf1: fresh exact review same witness then idempotency conflict at apply. Root cause not established; added stable redacted stage-specific error context preserving errors.Is and all predicates, not a claimed repair. Re-evaluate exact next failure before further changes.

Box resume 229efd52826db4cce539774cd26c0f29 clean exact parent; no UNKNOWN operations in pr191. Patch receipt bec482ce9b3c6d5f4dcb8a9d9acd7d2c 5431 bytes SHA256 f53894268a89ced26ac29512be34d9f211d92b36d18fdea15d85b43b65767b40. Gate be93a7bbbb7ddd93a7f26e5f41f999b0 exit0: 18 targeted Python regressions, focused CGO race recovery/state/CLI, vet, gofmt, shell, YAML47, format1312. No new native PASS. No new BDH asset: existing resume/verification/transfer guidance used; capture actual CI failure and correction. Physical quota, GC, remaining transport faults, S2.5 open.

## Update this record at material checkpoints

Record current phase/slice, base and head (avoid self-referential commit hashes), worktree/Box UUID and layer, actual installed build/digest, original operations and evidence, code/test changes, test inventory and results, CI run/attempt/job/SHA and current status, failing case and repair or genuine blocker, remaining dependencies, next safe work, learning disposition/link. Keep prior evidence append-only or link to the execution log; never overwrite failures. Preserve distinct Box/source/native/CI status. While CI is pending, state what safe work continues. A phase can be submitted to CI before its CI closes; mark COMPLETE only after required acceptance. Final review binds exact HEAD.

Learning disposition for this planning task: owner clarified pre-release compatibility retirement and CI overlap; recorded as task policy in plan, not a claim of general BDH efficacy. No new Skill or automation justified without observed development friction. P0/P1 successors should use existing learning mechanisms, not fabricate an improvement to satisfy a metric. Preserve existing inbox and reconsider relevant candidates with actual evidence.

## Repeatable resume instruction

Continue this PR from its actual latest checkpoint. Reconcile prior work and CI results; advance the next safe unfinished slice Box-first, iteratively verify/fix and submit coherent changes to CI. Continue independent development while CI runs; investigate and repair confirmed failures, keep evidence and handoff current. Follow repository BDH and phase gates; do not merge or publish without separate authorization.

### P1b-06b6 — installed Scoped Exec retention and private SQLite ENOSPC

Parent ff77fab99bcfb25434a2e714d88650499f1c26d6. BOX_VERIFIED (source only), CI_PENDING (actual native). Reuse the original S2 SQLite FULL witness without a second live controller or changing the installed DB; it uses a separate private disposable filesystem and synthetic original NativeAck, not an actual installed-native ACK or power-loss proof. The original S2 consumer calls the same extracted helper. New scoped-exec-retention N2 lane reuses the original SDK GC and quota-create/cleanup probes, normal installed fixed ledger, exact original native IDs, aged test-only receipt, two real output stream removals, distinct lost-ACK original with sibling preservation, reviewed exact GC receipt and restart truthfulness. The standalone offline test uses the original test proxy, not a change to controller transport. No production recovery policy or UNKNOWN operation release changed. Old staged-clone GC review remains available. Actual physical quota enforcement and S2.5 are still open.

Box evidence: exclusive patch d380d555c2f6d1e1b0573f60235c701a, 14692 bytes SHA256 8ef0d208a7e4bb19664bcfb2fca0f74d8dc818931eef7827f887f4deca0323ac, verified readback; c35f4049fe4a3d8b92f50af44812da54 exit0 four probe builds and shell syntax. State GC race and guard negatives, YAML47, format1314, diff check 69527e1ed56cc46f1f1e590b53c1d96d exit0 (control and CLI packages no matching tests). Original GC code assertions retained; native proof requires actual runner logs. L014 revision45 observation ebd25f89b33e56612bee04a6416634d0 and corrected work boundary revision9 5bcd29ccc74279c3ac81b13682f03756 FEEDBACK_RECORDED, no issues; private feedback draft 11c6e622b75cb2371247387fbffa0ce4 was unresolved until exact evidence references were added. No new asset or independent efficiency claim. Previous ff77fab full S2/Legacy review CI 45770012042 remains pending at this checkpoint. No P1c/d deletion, merge or release.

### P1b-06b7 — Scoped reserved IPv4 contract and private ENOSPC fixture correction

ff77fab N2 45770012042: 16/17 passed, including Legacy recovery rerun (intermittent conflict root cause remains unproven). Job 139740127059 confirmed ordinary native Create, update, disk growth/shrink denial, IPv4/IPv6 gateway and peer isolation. Old simulator 198.18.0.2 TLS host control succeeded, guest timed out; native_scope_common.go intentionally rejects 198.18.0.0/15 except exact gateway. Scoped probe now requires denial in ON/OFF/ON and independent zero hits, with gateway toggles and peer denial preserved. Legacy positive TLS and two-hit witness remain unchanged. Public IPv4 TLS positive control was UNVERIFIED on that runner; the Scoped-specific probe marker is not full public WAN acceptance. No production ACL change.

df951f9 N2 job 139742412406 failed before ENOSPC: nonroot service account could not traverse test-only sqlite-full parent created under umask 077. Correct only that parent to traversal-only 0711; isolated mounted ledger remains owned by boxweave with 0700. Actual ENOSPC and GC remain CI_PENDING; no fault PASS inferred.

Source build and YAML gate ed517910345947d795ff82380280d996 exit0; further source validation below. Remaining P1b evidence is not retired. No P1c/d removal, merge or release.

### P1b-06b8 — independent S2 and physical spool PASS; scoped GC review correction

Exact 5d513eb52e4a96f4d30debd94eb45bf69ab37601: S1 45771368432, quality 45771368330, Forward 45771368396, candidate 45771368462 SUCCESS. N2 45771368470: 16 PASS, one in progress (Create WAL-before), one FAIL at observation. Full installed Scoped S2 job 139744533253 SUCCESS: independent reserved IPv4 zero-hit denial in all Internet states; native gateway and peer packet, protocol/auth/history, actual lifecycle, output, actor/idempotency and foreign same-name read-only preservation. Original Create a874c567c4de48398da8ac0af1beb593 tracking d7ce09404350a2999ba3128a98893d66 native 78656336-c993-488b-ab8d-aafe8fc648a0 UUID 08d3e7fd-77ed-43d8-9bc0-35e9863d6076; Delete 5eccbda9af64435b6deea1a8d8bcd138 native 9e463f89-b403-4ed3-a578-1a270f0694ee. Exact native spool a74ddb0d-1d3f-4a8d-be31-de9a88be8dc7 both files 16777220 bytes PASS. This is physical recorded-output size, not actual storage quota enforcement. Public IPv4 TLS UNVERIFIED. No full S2/S2.5 retirement waiver.

Retention job 139744533150: actual isolated ext4 FULL original b8f0632bafd2cbebfc150a64b8385b09 PASS (synthetic native receipt); actual native 7-day two-stream deletion/404 and MCP expiration original 8b588e8aebdb2097eb3111c80912d451 native 05b040e7-288b-4fcf-873a-18bb4df34234 PASS; real native stdout DELETE accepted/lost ACK and sibling locked original f4679d29dd330a6b75a2390ecc4619ef native 963b754b-a68c-4827-9788-06a55a872b22 PASS. Failure at gc-inspect: restricted recover-worker repeats install.Status, which attempts the root-only Scoped fresh journal proof. Corrected only three existing GC review actions: local root full Status under existing maintenance lock and fences, exact reviewed manifest sent through the existing kernel-authenticated Root peer channel; restricted worker rechecks fixed identity, manifest, installed executable inode/hash, unit and exact fixed Root-owned unit bytes without reading the private root journal; existing DB only, original native GC observation and challenge/release semantics unchanged. Generic scoped legacy UNKNOWN release/bind remain refused; no new native mutation or root authority. Box source verified, installed native CI_PENDING. No claim reviewed release or final postrestart GC pass yet.

Box resume 772724ac2c8899c7ffacdafe0553940d exit0 clean head, no pr191 UNKNOWN. Exclusive patch 79c1dae56863b3ffb9e1d5ee9fc56d57 8202 bytes SHA256 f3ecd10d9eb2f50a8753c61bca312377ab0a338dc4eb2f599c8f43274ee8b358 verified. Focused CGO race install/state/CLI, vet, build and diff 8759047ad6c40cc8411cefe2750a5e49 exit0 (recovery package had no matching tests). L014 revision46 and linked GC work history revision2 FEEDBACK_RECORDED via beb1381ae2c8954342800bb43b1cdadc exit0; original lifecycle work rev9 preserved, private boundary bd189bc1557c4f516dd77e078311226e exit2 due 32-reference/feedback limits, rollover linked instead of deleting history. Existing BDH asset reused, no efficiency/promotion claim. Remaining actual native transport faults, physical quota, public IPv4 TLS and S2.5 gate open. No P1c/d removal, merge or publication.

### P1b-06b9 - real Update transport witnesses and GC single-lock correction

Exact c237e78eefd9185dee672f2309b918deb408e9c0 N2 45773971092 retention job 139753038251 FAIL: real isolated ENOSPC e1be8d943bec04bb4bdda701f440bf16 PASS; two-stream GC ead5b865a3256fa616d71d9f1d692823 native a0e7e88b-14d0-45f9-abfb-83a3be682a11 PASS; native lost ACK fb9983dc93cbce1de53a97a1084f0416 native 4e260e0d-f932-4287-a583-ef61a75cd06f PASS. GC review failed on the duplicate nonreentrant maintenance lock introduced by c237e78. Remove only the inner acquisition: main already holds the lock across recoverLocal and worker; retain fences, full root Status and restricted exact-identity recheck. Box focused race CLI/install/state, vet, caller check, format1322 and diff 78c6d4498d72c0ca247a712067c4fde1 exit0. Installed reviewed GC release and restart remain CI_PENDING.

New independent scoped-update-transport N2 lane adapts original S2 official SDK and native proxy for three exact owned stopped fixtures: real stale ETag 412, real 202 lost response, accepted 202 plus failed readback. Test-only transport is restricted to one exact fixture Update, with original claims, replay refusal, actual native state and unchanged root/guard identity checked across restart. No production transport change, no unknown cleanup. Box exclusive patch 6a1a18f01322b633c518969de0ec64c3 (11198 bytes), wrapper 37f74e44edf80f0fc1705bb3de091f5f (12504 bytes), verified readbacks. Gate f5bb710fe3cda2bc9513ab043fcfcbbe exit0: two Go probe builds, non-CI refusal, 18 Python tests, shell/YAML/format; final tightening and staged gate 9633e17bc23a903a06ca478d7b4003a7 exit0. Actual native transport CI_PENDING, not a PASS. No new product API. Prior full installed S2 and physical spool evidence retained; actual quota enforcement, public IPv4 TLS, nested S2.5 and remaining original witness mapping open.

BDH L014 revision47 actual reuse, no measured independent benefit; work history GC revision3 recorded via 34a9f49f72ebfce951a3a148e9a4ccfd exit0. Initial incorrect executable path 7e4d5d448e78408efbdf115efdd1355d exit127 corrected; boundary 4df526cf5865cfc0071e83f7ca7a7dcc refused changed target without feedback, retained original work target and explicit current checkpoint evidence. No P1c/d removal, merge or release.

### P1b-06b10 - GC and Update transport independently verified; physical quota candidate

Exact 426a9b62273c3854c59d98e9ba58c1a46a0a78f2: S1 45775201090, quality 45775200917, Forward 45775200970, candidate 45775200782 SUCCESS. N2 45775200961 at observation 17 SUCCESS, two running. GC 139757072286 SUCCESS: private isolated ext4 ENOSPC cdde27581f05223a9320194e8ca8527c (synthetic native receipt); actual native two-stream deletion/404 and expiration original 4df4f3feafa9bcbc9e7069d037de3e6c native 98e1431f-480b-4884-8121-e0bb9c7c7ec8; native lost ACK original 42ddf5670f3da3d12e267ebaf58d12f4 native 1fc185a4-4d78-41df-b96e-c697388bb026; exact installed root review/challenge/release, no second DELETE, durable audit, restart TTL/physical distinction, fixed ledger and native absence PASS. The single-lock correction is independently verified.

Transport 139757072237 SUCCESS: actual stale ETag original 7879dd27b6058b2805cf3d9f22fc1840 tracking a500c4dad3c64769dac4af0f3f4f2478 UUID 2a91971b-003e-4a9d-8920-66abaadfe6fe; accepted lost ACK b8e0ed4e49dcd542fc383196a8ac32f0 tracking 5d04ccf25199d04618408895d3bbee79 UUID e3ad3037-8a17-48b6-b131-a315b36b7cf0 upstream 5b7c4fda-c48c-4975-861d-e1a0776780ef; postread 401fab85235c951615092b4572d55198 tracking cb24f1a6b28c0364bee8adbfcccc99b3 UUID 49039cd3-58be-4e0b-887e-d5bc7de2cb64 upstream 58fa87aa-03fd-46a4-b484-bdc3b8cec77b. All three exact UNKNOWN original/restart/no-replay/independent native witnesses PASS. No claim of general UNKNOWN release.

Physical quota: extract existing S2 separate ext4 project-quota setup and public MCP quota/archive/missing-output witness into shared guarded files; preserve original assertion bodies byte-identical and original legacy caller. New clean N2 scoped-physical-quota selects the test pool before reviewed fresh installation; unchanged official SDK probe and original native quota assertions, exact installed fixed-ledger exit/lineage, normal owned cleanup and process identity. No production runtime edit. Separate direct-native quota probe in original S2 retained; ext4 acceptance does not certify Btrfs or public WAN. Source-only Box checks PASS, native CI_PENDING. No quota workload ran on retained Box.

Resume fc7cb02c567a93010c4e2d834e961efb exit0 clean exact head, original UNKNOWN empty. Exclusive patch aa86f09c0ffb252dd2462067f830dd86 9457 bytes SHA256 8407a1f42ea4024fad7241235c7f30236067858531f1aaf79ea1614be5114bc7 exact readback. Source extraction identity/2 checks/YAML47/shell syntax 16360bc1a199be83e89c35b287a1b740 exit0; unchanged SDK build SHA256 15e68d9ac45d4b5971791e49de28891b9715be99cb09f333d4ec2beed8b87d76, 20 related tests, format1328 and staged diff 09a3512c8e58d40a837f2d4a3a06eb04 exit0. L014 revision48 411768860b5896ed06cba17d10a03fff exit0 and work revision4 FEEDBACK_RECORDED 355db2cca51c1047de12cde801af3da3 exit0. Existing work-plan audit flags L014 IMPLEMENTATION_CHANGED_OR_MISSING; preserve candidate, no promotion or measured benefit claim. Remaining original fault mapping, physical quota result, public IPv4 TLS and independent nested S2.5 gate open. No P1c/d removal, merge or release.


### P1b-06b11 - physical quota verified; resume installed private-read auth races

Exact parent f1931614b8771eb6759b4568d0dcdb440382ba26: S1 45776554020, quality 45776554012, candidate 45776554103, Forward 45776553975 and N2 45776554177 SUCCESS. N2 20/20. [Physical quota job 139761484820](https://github.com/vanishs/boxweave/actions/runs/45776554177/job/139761484820): real ext4 project quota reached 2147483648 bytes (used and limit), native stdout 50327552 bytes, original c1e446bc09d847670d3a377accfeef1b native 1308796b-9481-44c2-9603-21a40b45edd6, guest exit one, bounded archived prefix and typed native missing output, exact cleanup and fixed ledger/native absence PASS. This does not attest host Btrfs quota or public WAN.

Resumed existing dirty work in Box 9d374e18-196a-493d-bf1f-2d582589f04d, /root/pr191-consolidation, exact parent (405b76fbb2db107e13720718c0c8ca12 exit0, UNKNOWN lookup empty ffdd4f0f7c99f759234051021e7e619d). Preserved all eight inherited files, reviewed diffs and previous build 5db287bcad0fb9a7d5320a750b0c54a6 exit0 (three Go SDK/proxy builds, non-CI refusal, YAML). New scoped-auth-races N2 lane reuses two actual compiled post-query auth cuts, four held native read revoke/expiry pairs and stalled-output disconnect/fresh-client recovery. Installed fixed ledger, restricted UID, exact candidate, read-only fault proxy, independent original receipt and native absence checks. Only terminal synthetic list rows; no synthetic dispatch or production policy edit. Tightened CI to require each individual witness and exact private-read original count. Source check 47b94cd0763fa524b660436a66164dcf exit0: gofmt, shell/py import, YAML47, format1330, focused CGO race state/control; individual shell syntax and review 05de39bd576591eb098bac5aa909381b exit0. Actual installed race CI_PENDING.

Learning judgment: resume and verification guidance reused; prior original builds recovered instead of repeated. Existing L014 candidate remains unpromoted; no new asset or measured efficacy claim. Initial inline edit admission refused without receipt; use exclusive source file transfer. Remaining: independently verify auth-race lane, direct-native quota and other original S2 marker mapping, public IPv4 TLS, nested S2.5 and P1c/d/e. No Legacy removal, merge, release or Host mutation.

### P1b-06b12 - real compiled auth cuts PASS; seed quiescence correction

Exact 77693ec4776c8134d6ad2162981b4d4a0b6af8be N2 run 45782418253 job 139780833610: actual installed compiled call cuts revoke original ed18fbe79f64ccca9195404316f4dc55 tracking be833a19f36030e26572c0efebf4d098 and expiry original 588fe3cf4b100f7aae568fd3d24867f7 tracking d15c1de7e57193953becac1199b841d4 PASS; exact two breakpoint hits, one service call each. Private-read fixture real native seed succeeded, then combined native names/idle assertion failed before the delayed reads. Exact cause not proved; likely transient retained Incus operation. Separate inventory check, bounded read-only idle wait using existing observer, recheck UUID/binding and exact native inventory, require explicit seed-idle marker. No redispatch, relaxed inventory, runtime change or private-read PASS. Resume a54185436060ea94b2a6825968e008b2 exit0, UNKNOWN query 98e4fc4b33b9daf77dd90135272239ac empty; source edit e1a7eca3adb82f1cd0b39f63b4f18e4b and checks 4d45346e33660664bc4e9498eb6f81a4 exit0 (Python import/compile, YAML47, format1332, diff). Corrected lane CI_PENDING. Reuse existing read-only convergence helper; no new BDH asset or promotion. Continue independent S2.5 conversion, no Legacy removal.

### P1b-06c1 - independent dual-layer Scoped startup/lifecycle candidate

Parent d2b809b91edeb7f28546d4e4880630e3a30f04a0 (auth seed quiescence correction); exact Box fast-forward 92be14d9146606f734a47361d44292e5 exit0. New PR191-only independent S2.5 startup/lifecycle CI lane uses exact PR head and the existing hosted disposable L0 and original MCP-created L1 fixture. Both layers use the product native-plan/bootstrap and digest-reviewed Scoped install.sh; each stale review must refuse before persistent installation, both guard/helper units active. SHA-bound wrapper and helper transfer to L1, no parent Incus socket, no inherited policy changes. Original dual-Agent receipts, real TLS IPv6 simulator, exact-key replay, lost MCP reply, revocation, controller restart/SIGKILL, L2 lifecycle, guest Git and normal owned cleanup retained. Independent verifier checks actual binary and installed status, per-incarnation ACL usage, fixed SQLite Create receipt and distinct socket identities. Existing historical and release S2.5 lanes remain intact; this narrower candidate explicitly leaves foreign-project packet verification and legacy upgrade/rollback disposition OPEN. No public WAN, recursive isolation, or full S2.5 PASS claimed.

Exclusive source patches 6d0275e1126521f2e6ea7406dc384855 (13355 bytes, SHA256 da7c448886e91d0f0e7c88b32e2de9b1bb2b5c45516332d26d69a10a391bf2c4) and 37488499fbe00ca36d25cfb15968c229 (9465 bytes, SHA256 84f969043dc6a0d4e44bec2a10c518c5ac8434b7c12f892f7482fc5f3c78d665) verified. Source application 5e17a05ef55efa0fbf3224bcdea45ed1 exit0; shell/Go format. Build/check e7af4a453495cebdaa28a4e310ee40bf exit0: source-only non-lab refusal test (3 subcases), shell syntax, Python compile, Go SDK agent binary SHA256 4f02499aa1615427d5f4d2bda279285934ff5ba93d3884b173de4d5c9699c87c. Source review dd86eeee76b0a434a66e2e9cb39724d6 exit0. Real nested CI_PENDING; existing retained Box cannot establish clean nested capability due prior inherited L2 limitations. No host policy change.

Prior exact 77693ec N2 job 139780833610 failed after two real auth-cut PASS markers; fixed d2b809b candidate N2 run 45783427232 job 139784188128 pending at observation. Remaining P1b auth private reads, original direct-native quota and other marker mapping, public IPv4 TLS, nested startup/lifecycle CI, foreign-project and maintenance witnesses. Learning disposition: reuse L014 source transfer, existing installation and independent verifier. Initial format gate 567552bf8c1c1956d557a0939fcafe02 exit1 missing gofmt on PATH; corrected 8dd7d2a66ca388208752ca5afca6e1a7 exit0. No new BDH asset or promotion.

### P1b-06c2 — auth-race recheck and S2 overlap contract repair (2026-11-26)

- Independent corrected-head N2 job [139784188128](https://github.com/vanishs/boxweave/actions/runs/45783427232/job/139784188128), commit d2b809b91edeb7f28546d4e4880630e3a30f04a0: PASS. Both held list cuts passed; revoke original b35332a834848538a181b36212683a67, expiry e9267d1aaf5d601aef317eb8ce6a7c9c. Actual seed became idle after bounded observation. All eight native HTTP-200 held-read revoke/expiry cases, cancelled recorded stdout and fresh read, exact ledger and native absence passed; original c361f276db0dcf5396ca842cffa3bd82, native b3ebf291-cd69-44e2-b4b4-739985897152, UUID 0544c0fd-9176-4380-8da0-bc4cb4e76416. No redispatch.
- S2 job [139780833270](https://github.com/vanishs/boxweave/actions/runs/45782418353/job/139780833270), commit 77693ec4776c8134d6ad2162981b4d4a0b6af8be: FAIL at legacy preatomic native Create/Delete overlap, after native UUID witness and before twelve refusal checks completed. Probe expected internal state.ErrNotFound text; current public ToolFailure redacts it as BOX_NOT_FOUND_OR_NOT_OWNED / AUTHORIZATION_OR_ABSENCE / UNVERIFIED. Repair consumes exact typed SDK text envelope. No production change, no weakened independent zero Delete ledger / unchanged VERIFYING original / terminal tombstone checks. The native rerun is still required.
- Box 9d374e18-196a-493d-bf1f-2d582589f04d: probe build and RC8 test exit zero (b1dfffb7874b0b1a18178ba75111ca0e); new SDK Delete-unowned public envelope regression and RC8 tests exit zero (5e60111a51db2dae53c73aaaf6a2bce4). First read while Exec was pending returned operation-output not found; read original again after terminal, no replay.
- Scoped nested startup/lifecycle source candidate c972ff1ec35c1cd18d60ed769d4e094b65a53973 independently queued as run 45784056084. Still OPEN: nested native result, cross-Project packet/foreign fixture, maintenance refusal/old positive rollback disposition, remaining direct native quota and public IPv4 TLS evidence. No Legacy removal, merge, release or host policy changes.

### P1b-06c3 - Scoped foreign/maintenance conversion and nested startup correction

Resume e9d15cb8af317be4bf206707432ecdd8287050ff, clean Box 9d374e18-196a-493d-bf1f-2d582589f04d, original 5dbc7b1323a002e8bf408e740f0c2202 exit0; UNKNOWN query 938c40ec46c806ae12922406101f1f8a empty. Previous d2b809b N2 run 45783427232 21/21 PASS. e9d15cb physical quota job 139787997515 PASS.

S25 c972ff1 run 45784056084 job 139786264964 FAIL: inner bootstrap Exec 13d1ec246e995669de19f538dcc66942 guest exit1 after inner native bootstrap and restricted foreign Project refusal PASS. Real outer Scoped install, original L1 a2a0e50a0dc1dca1a6158e3f1586b04e UUID 8c736296-e9e6-45c8-906d-dea99318ffe3, routed IPv6 ON/OFF/ON, dual Agents, real lost ACK/revocation, quiet wait and restart/SIGKILL PASS. Public HTTPS BLOCKED, package relay PASS is not public WAN. Likely fixture repeated broad mount test after local Incus creates devices/shmounts/guestapi; actual Box read-only mountinfo confirms all three (0450c988945ebcd33913b0a16be2c5d5). Strict pre-bootstrap refusal retained; post-bootstrap reject exact directory/socket mountpoints, require active local daemon and both sockets, independently compare distinct parent/inner sockets. New explicit phase markers and bounded complete original stdout. Exact CI cause still hypothesis pending recheck.

Foreign conversion: dynamic UID, original L2 PINNED ACL/UUID/Create receipt, exact gateway exclusion CIDRs, real bidirectional packet negatives and gateway positives, exact original foreign UUID cleanup and unchanged ACL. No MCP mutations while foreign fixture exists. Maintenance conversion: active/offline wrong/exact digest Legacy upgrade and rollback refusal, exact binary/unit/manifest/SQLite/ACL/guard before/after, normal service restart and original SDK readback. Successful stable upgrade/rollback explicitly #170, not a fake PASS. Existing historical S25/S3/S5 gates retained. Root Doctor guard observation required both layers.

Source patch originals 5500b4368adbbba92dd745046fd8b2c3, bd72035dd27bd5a23427ec79be9f1438, 8f53c74e0113abbbba123fe7a257cede verified exclusive file/readback; source application/checks efe6bbb6d8e52e8b45575a267aac0b79, e82c6a59f96f939872943299165d67c0, 1c6f8145d39925de827a3bd0d9442b14, dcf80f2b2f9b129dca357aadfe2bce2d, 1763c6730c9f8c0cb2d2b0dac10d14d7, 51f7f4508d214c6078311d27a7718d21 exit0. Three source tests incl exact CIDR and mount classes PASS; Go SDK compile, Python/shell, actual read-only Box mount classification PASS. Native CI_PENDING. Oversize argv explicitly refused before admission; reused L014. Learning: retain fixture regression, reuse existing assets; no new BDH promotion.

### P1b-06c4 - direct native quota conversion and atomic receipt refusal repair

Legacy S2 e9d15cb run 45784576105 job 139787997866 independently passed the preatomic twelve SDK Delete refusals and complete tombstone lifecycle, original Create a0a8364ea120e289cb0327c10cf785ac, native e52968db-ed12-4797-90f4-54717e882a1c, Delete 355f2f28d5b62107bd292eb5d2f53df2, UUID 2c7081ae-f432-443a-90e6-b6a250f69ef1. The same job passed independent direct native quota (native 789f0f1b-0eb5-4a82-99e7-6c7f87641da4, guest exit1, physical 50327552 bytes, native used/limit 2147483648), public MCP quota/archive/GC, real WAL/ACK, two controllers and private reads. Job FAILED later at atomic bound-Create rollback probe line148: outdated internal ErrResourceLocked text check. Repair asserts typed UNKNOWN_BLOCKER/EFFECT_UNVERIFIED/NOT_ADMITTED and exact original recovery navigation; unowned Delete/Get require BOX_NOT_FOUND_OR_NOT_OWNED and no leaked original ID. Original SQLite rollback/recovery assertions unchanged. Box build and control navigation tests exit0 (6469c99d8af2d59db098ed590289ae43); native rerun pending.

Converted independent direct SDK quota fixture: extract the original physical measurement and exact filler cleanup into common guarded script; historical native Create/Delete retained only in legacy S2, new installed Scoped path uses exact MCP Create/PINNED UUID/ACL, existing restricted native Reader.Exec for direct seed/flood, independent quota and physical spool checks, no fabricated MCP Exec originals, normal MCP Stop/Delete and fixed ledger, released binding, empty inventory/quarantine-only ACL, unchanged daemon identities. Dedicated hosted ext4 only, never run quota workload on retained Box. Source tests and both SDK builds exit0 (9cf229ee57d07d54e0807b2c4190224d). Exact native recheck CI_PENDING. Separate no-project-quota warning and independent spool EDQUOT marker still require installed equivalence, public IPv4 TLS remains open. S25 e9d15cb repeated inner bootstrap failure after restricted Project preflight, original 2274859a2e67eab467b9bec143a95475 guest exit1; corrected 6d7f0da run 45786193712 in progress at observation. No waiver or retirement.

### P1b-06c5 - exact S25 image identity and mirror reliability

Resume a42aa91, clean Box 9d374e18-196a-493d-bf1f-2d582589f04d, original dc6e666946890318ddfc7530a3223087 exit0; UNKNOWN tracking 5451b1a96423167f0cf358ee49dd8e7f empty. Exact-head CI six workflows PASS: N2 45787275208 21/21; S2 45787275176 job 139797124412 full 19 grouped TC/A PASS including atomic rollback; S1, quality, Forward, dual architecture. N2 139797125162 proves installed direct native quota: 50327552 physical bytes, used/limit 2147483648, Create 4fd24866032748f29b25a7fda12fb72f, native seed 1fd7f4f2-cf6a-4bdd-9616-4dfe08dd9110, flood 9b8094a0-0a05-4a97-b83e-6abc97bc0ed1, Stop 3dc3ccfbcb23380edc5b0dc76403d129, Delete f05e51a06c28d039ccec85af0e814f0f, UUID 12f1480f-1f67-4732-b416-b2822fb3c90f.

Prior 6d7f0da S25 run 45786193712 job 139793434921 FULL PASS. L1 original 74c79218217b3cc201315cee3f9789c0 native 869f91db-9327-4ba2-98d8-e5c9d47654fb UUID aa35471e-0c72-470c-840e-c645a3ed8968; L2 original dfbebe12d2d9208c92e84a5e052ebe5e native 7c6679d1-6a45-4336-92f0-9d4568bb0fa2 UUID 0ed32b2e-922f-4228-bedc-5cf3e150f836. Both layers install/root guard, IPv6, foreign Project packets/cleanup, original Git failure/retest, exact L2/L1 cleanup and online/offline maintenance refusal/recovery PASS. New a42aa91 S25 job 139797124607 FAILED original bootstrap 01d06c8577530b2172eee3322ecfb483 exit1: exact stderr remote image download from fra1mirror01.do.images.linuxcontainers.org Forbidden after native bootstrap. Mount fix passed prior run; latest failure is changing upstream redirect blocked by strict relay.

Correction: reuse exact L0 official image, native split export (Incus 7.6 source verifies combined metadata/rootfs fingerprint), per-file SHA256 and combined fingerprint before import; independent both-layer alias identity; foreign fixture imports same verified bytes into separate default Project. Existing private staging only; no relay expansion, parent socket, public bind or production change. Four source regressions and SDK build PASS (40e5e13af7b53640d1a50d75e7b6a613); patch ec5f2f19cf5a4674773579a727bdfa9f SHA256 c3b59622f94bfa5300eae267ed48cfab2978cce20ce3c491c9848f8945201025; apply 7e1a113003c4988cdb6b83b8489dab4b exit0; full fingerprint column fix fe1078afed7a7097396ec2c4f6123623 exit0; test quoting failure 845a5e61dac891ae0ddfb20570f5488e corrected. Retained Box local image inventory empty, no clean import here; CI_PENDING. Learning: reuse existing exact artifact and bounded transport, not broaden authority; no BDH promotion.

### P1b-06c6: noquota, independent EDQUOT, and public IPv4 oracle

On cdd9e0d, add a 22nd installed Scoped N2 job for a separately mounted ext4 filesystem explicitly created without project/quota features, selected before fresh review. Require exact installed root/restricted Doctor readiness, unverified-quota warning and post-lifecycle empty inventory. No runtime pool switch. The exclusive unmounted regular-file mkfs feature check in the dev Box passed (dd84e8d9626c465348f8a6bad48ad04e); this is not a native installed mount result. Extend physical quota witness with independent over-limit allocation after reclaim, exact EDQUOT rather than ENOSPC, capacity prechecks, CAP_SYS_RESOURCE dropped, original receipt/UUID preserved and reclaimed usage independently verified. Old witness retained. Native checks CI_PENDING.

Public IPv4: a42aa91 installed S2 job 139797125257 and original S2 job 139797124412 lacked a positive public WAN baseline. Reserved IPv4 denial ON/OFF/ON passed. Dev Box direct HTTPS example.com and pkgs.zabbly.com/key.asc HTTP 200 (894b20b6d06d0070adcfa3af0816685b) is environment-only, not product installed-child evidence. Harden public probe with direct IPv4, no proxy, HTTPS-only, exact 2xx, offline nonzero plus HTTP 000, same endpoint after restore, and independent SQLite original/native/tracking/UUID/guest-exit verification. No positive baseline stays UNVERIFIED. Separate runner-host control; no network policy change.

Source verification: noquota patch 2070d01e65f6be8c6c5234d2d6ffe59b; EDQUOT patch 9be1782ab7f274b81ec2e64dc7d028c2; Go build and 5+2 Python tests 32e8d530411aac6005a5c372035465b8; final host-control correction, regular-file check and tests dd84e8d9626c465348f8a6bad48ad04e, all guest exit 0. No native quota workload or host network mutation in retained dev Box. Existing S2/S25 gates retained. No retirement waiver; remaining installed fault coverage mapping and public WAN native positive evidence OPEN.

### P1b-06d2 - held operation_get and changing signed cursor

Resume exact 333b6b5: all seven workflows success (N2 45792289720 22/22, S2 45792289584, S25 45792289644, quality 45792289800, S1 45792289643, Forward 45792289559, dual 45792289603). Prior alternate-pool stage preserved unchanged in /root/pr191-consolidation tree 8a14a0e10979bfca5c54b7f82632aa0f579c7277, publication security refusal recorded at PR comment 7188235951. Original Box resume 536b31c3c8a20b7935afc6e1b85f866e exit0 and UNKNOWN list ef2376fffcc7691b4f7e29879345b569 empty. Independent worktree /root/pr191-p1b-auth-cursor original 19009ae44d1abaf41eb40d359d4fef04 exit0; no reset or reupload of blocked file.

Convert remaining original s2_p1_token_inflight_probe.go into existing Scoped auth lane: same fixed ledger and restricted account, normal installed listener, independently issued fixture actors, positive original read then bounded wait, revoke/expire and typed redacted AUTHORIZATION_REQUIRED tool refusal (not transport timeout/401), unchanged original receipt, exact signed cursor state change and complete identity set, durable cancellation of only seven deliberately undispatched test rows after proof. Independent read-only SQLite snapshots compare all prior operations/events/claims/outputs/audits unchanged, exact seven native-free terminal rows, exact key and actor lineage, no unexpected effects, unchanged daemon/guard identity, quarantine-only ACL. Existing legacy probe and all original gates retained; no product auth or policy change.

Box source SDK build 533c75df63c479d876abdb253fe7b335 exit0; independent oracle b77a13ed62bf6566f96f9c7e0ea278fc readback bd35bb63ad6d7311b87801a98399f474d529250df4e87620a686ec4dc7e9473e; three private oracle regressions and five targeted Go state/control regressions a2d876649d3bb25d40dd5eccb62bb2ba exit0. Strengthened typed refusal source test and rebuilt SDK 049e646b3a15ec8b35d2e010725c4c75 exit0. Actual installed acceptance remains CI_PENDING until published. Learning: do not equate HTTP transport failure with in-handler authorization proof; reuse existing fixture and metadata. No new BDH promotion; publication blocker remains separate.

### P1b-06d3 - real malformed terminal Exec

a11ce61 submitted and clean independent Box sync feaae4e71fb992b08622c78303daa226 exit0; auth lane 139824837922 running, N2 45795455806 three completed success and zero failures at observation. Original alternate-pool tree 8a14a0e10979bfca5c54b7f82632aa0f579c7277 remains separate and untouched. Convert historical S2 malformed Exec with unchanged original restricted native proxy, official SDK, exact installed binary and fixed Scoped ledger; seed normal owned Running Box before fault, actual accepted native 202 and matching malformed final native ID, fail-closed EXEC_TERMINAL_EVIDENCE_UNVERIFIED, no output receipt, same-key original only, typed competing-key refusal and exact original navigation, preserved claim and guest effect exactly once. Restart unchanged actual systemd controller, no altered guard/helper, no alternate ledger, no administrative cleanup of UNKNOWN. Independent native UUID, NIC, pinned ACL, original actor/tracking/native ID, original event/claim/output state, prior operations and no duplicate proof. Legacy script remains; new N2 mode CI_PENDING.

Box Go SDK build 61e44fd77a1d76b84e380c1c2c0de5de exit0; new independent oracle c17aed35a52836a3cdaf11ff7d518265 SHA256 3362e369b9d103a48ed32ca0b30aa17c219cc0a583459fac1680a06a5c8cac7f; shell/compile a4fabc1d96e3237f2bdd9a66dfb40faf exit0; five oracle regressions, unchanged native proxy build and two targeted native/state tests 648b0ab33faa83aadc38812ce0bb0fd1 exit0. Missing historical file path inspection 24af627c22efd76df8a69a2fea73fc83 exit2 corrected by inspecting actual box.go/worker. Source only; actual one-use native acceptance pending. Learning: reuse original fault transport and exact identity, not broaden authority; no new BDH promotion.

### P1b-06d4 - standalone Stop accepted ACK loss

a11ce61 N2 auth job 139824837922 CI_VERIFIED: revoke original e2e9885ad9f9024fd6cb82022dbfebcb tracking c417f9b478defb3e9db80cf785843bb5; expire original 4afb396be3a2ccd7c2f684d19d0a1750 tracking edc83e72a99822f35628458c93703914; exact five cursor originals and fixed-ledger no-native-effect markers PASS. 21/22 N2 passed, one pending at observation. 2d0cd3e clean Box sync 02fba51626b748c577b705c124a624b0 exit0; new malformed Exec lane queued 139827730788. No failure observed.

Standalone Stop still uses restricted native SetPower, unlike root-assisted Delete. Reuse original Stop/Delete SDK and unchanged confined ACK-loss proxy in new installed Scoped lane, only Stop in scoped mode. Exact fixed ledger, accepted 202 independently witnessed native ID, missing native ID on original UNKNOWN (no invention), same actor, tracking, box/incarnation, unchanged PINNED ACL, actual Stopped state, prior rows, no new-key admission, typed exact original refusal and same-key read after normal controller restart. No admin cleanup or replacement of guard/helper; stopped fixture retained on one-use runner. Original legacy Stop/Delete witness remains. Box SDK/proxy build ef787ca05048811bbea4cf508097f2ef exit0; oracle creation 5ed93a627525f83c2ff072fe68c917e4 and 240695e9960ce0d2ebc1926057e81873 exit0; six oracle tests f6c064be28494a380419765a8b7a5c32 exit0; workflow and shell validation f3e157ca6af28bd3161eee90d56e31b1 exit0. Installed acceptance CI_PENDING. Alternate-pool original tree preserved; no publication bypass, policy change or retirement.

### P1b-06d5 - pre-atomic installed Scoped Create/Delete overlap

Exact 2d0cd3e all seven workflows/N2 23/23 CI_VERIFIED, malformed Exec job 139827730788 original b57e90412766423b69c67642ca7f5b2d native 52eed8f5-6b2e-4a9f-8e80-5bc17c0e3104 tracking be87523f75ff169458f9ae42434373ff UUID 327a1eb4-1297-4f59-bbf5-8d45d6673429, retained UNKNOWN verified. ab00971 N2 last 16/24 PASS no failure, Stop job 139830829075 running. Retained Box resume b12de2778594729c3dd36b9b501eb394 exit0 clean ab00971; no UNKNOWN (526b7beaaab929dd519a12829ed2e1f6); blocked alternate-pool original tree unchanged.

Reuse original S2 overlay at exact shared atomic Create finalizer, confined to the scoped binding path and one-use named fixture. Existing legacy behavior retained. Build original and overlay separately; never replace installed binary. Normal controller stopped only in clean disposable runner, overlay uses fixed ledger/restricted UID/real Incus socket, Root helper and guard unchanged. Independent pre-atomic native UUID/Stopped/ACL/journal and five prospective claims, no owned/binding/fence/terminal row; twelve real parallel authenticated SDK Delete refusals (two actors), zero new admissions or altered original/native effect; independent release only after both observations. Then real atomic Create with exact UUID/pinned ACL/dual fences and separate authorized Delete tombstone, RELEASED fences, no residual native guest/ACL, unchanged prior rows; normal systemd controller restarted. No production behavior change or gate removal.

Box SDK builds 7ef27b3f06625f2263779c3555d23170 and c5b6aa8a4052c250bb884555418c66c3 exit0; independent witness 13329a5cf3d26af2f5d937fa5dc89db7; source helper c1d6e7c715da03ee2895e6e36e7ee735 and wiring e883ee29f2b493c96ad832fc61e88d33 exit0; initial oracle d77aaf2e227be951b99fffd36e24d772 exit1 incorrect SQL argument count corrected. Actual four oracle tests, both source-only Go overlay builds and targeted Scoped Create tests 0b4483f143978704bf1a0eecd5e16ce8 exit0. No overlay execution in retained development Box; actual native CI_PENDING. Existing native final-commit/WAL recovery tests remain; add explicit rollback refusal crosswalk. No BDH promotion; reuse L012/L014 and existing overlay.

### P1b-06d6 - original atomic rollback refusal crosswalk

ab00971 installed Stop ACK-loss job 139830829075 CI_VERIFIED: original 2ed05104bf3900aa2e1dcaab88f29a6b, independently witnessed native 5f1b7ec1-cfa8-4397-84f7-ab4093b82525, tracking 64bc302287e4ac5b790d87a926547beb, UUID 55c0be34-b395-41ab-ac05-90905e094648, true Stopped and retained UNKNOWN after restart PASS. N2 22/24 PASS no failure at observation. d32939b clean Box sync c206ef2af93a0253af2ec444cb3727fc exit0, N2 45799271822 queued.

Reuse already-existing real installed Scoped final-commit fault, not a redundant new Create or CI job. Before existing reviewed recovery, original official SDK now requires exact same-key UNKNOWN receipt/tracking/native ID; typed different-key Create UNKNOWN_BLOCKER navigation and unbound UUID Get/Delete ownership refusals. Independently check unbound ledger, no ownership/binding/fence or split terminal event, all five original prospective claims, no recovery audit, exact unchanged original/other rows, native instance, ACL and Root journal before/after all refusal requests. Then existing actual authorized reconciliation verifies original/no native replay, pinned binding and both RESERVED fences. Old legacy copy-only UUID bind is obsolete for Scoped; existing TestP1ScopedCreateCompletedRecoveryAtomic/legacy-bind refuses it, real installed reviewed recovery is stronger than copy-only, not a reason to remove a still-unproven gate. Existing running fault fixture retained; stopped exact native identity addressed by overlap fixture. No production change.

Box SDK build 51a9f572d8cbd3c6f2dff29890af69fb exit0, snapshot b8d83a2f6a35d1228942d5de5d5c900f and integration fd3c4073d858a29ebf6e29dc42a63e98 exit0. Five snapshot tests and eight state proof adversarial subcases 70581ca397828af5b10ad00072a9dee8 exit0. Oversized edit admission refused (no dispatch), readback 96d1e8c5a72f2b8de2a8a92ba32a46fe confirmed untouched source; subsequent small edits succeeded. Initial CI wiring anchor d78ce831ffd48b9379bc90858938a9fa exit1 (existing env prefix), corrected 3fb12e48e7c42be65f6cc1f5b6153040 exit0. Actual new installed SDK negative gate CI_PENDING, prior native recovery PASS remains valid for prior source. Reuse existing assets, no BDH promotion.

### P1b-06d7 — Legacy CI receipt-window repair (2026-11-26)

- ab00971: six workflows successful; N2 23/24, failed Legacy job 139830828501. Original completed Stop d5c07f688ca4ef4d1e38eeb02f75190e at 10:16:06.818; positive plan 10:16:09.155, guided decline 10:16:10.214, fresh plan 10:16:11.159, refused apply 10:16:11.862. The exact review gate correctly refused; no recovery admitted. Incus 7.6 upstream Operation.done removes completed operations after five seconds (https://sources.debian.org/src/incus/7.6.0-1~exp1/internal/server/operations/operations.go/). This strongly explains the timing; failure log does not independently prove native 404.
- Test-only scheduling repair: parallel independent authenticated UNKNOWN navigation and local review; four invalid envelopes and guided decline run before approval, with all witnesses joined and original DB/journal/PID unchanged. Apply the first exact envelope once, retaining product fresh proof and transactional gates. No automatic retry, no runtime or retention change. Failure path read-only plan diagnostic, with original evidence retained.
- Box checks: 52580f35265e8c2ca5785c2803dd1f43 / 2731dc3801425bd93edfe283e1151c0c successful edits; 8c3243615ea68b95f1cdcfa6967fce1d exit 0, six installed recovery fixture unit checks, py_compile, diff check, format 1347. Native CI must confirm timing repair. Prior upload-blocked alternate pool remains unchanged.

### P1b-06d8 — independently installed dual controller isolation

- d32939b scoped-overlap CI 139837920373 PASS, Legacy 139837920216 PASS. The new actual preatomic witness supersedes the modeled-only status for 06d5; previous 06d7 failure timing repair still requires its own CI.
- Scoped-only S2.5 conversion reuses already-installed L0 and nested L1 plus their original Boxes; the inner original now deliberately shares the outer name, while UUIDs, originals, native sockets, installed ledgers and derived ACLs remain independently observed. Legacy S2.5 distinct names unchanged. Official SDK in each layer verifies its complete owned list and original, typed foreign Get/Delete/OperationGet denial, and identical own list afterward. Separate read-only oracle checks no foreign UUID/original or refused work admitted to either ledger, then re-verifies actual native and installed guard identity. Existing foreign same-daemon Project/IPv4/IPv6/native socket proof and original cleanup gates retained.
- Disposable CI-only cross-token auth witness issues two new three-minute fixture credentials through each installed restricted CLI, confirms own HTTP initialize 200 and other controller 401 in both directions, revokes both in finally. No existing Agent credentials moved; no credential bytes in argv, console, evidence archive or staging files. Real independent native CI pending. This is not support for two arbitrary Scoped controllers sharing one native daemon.
- Box checks: 0b8607d3107a999292d9410b95073b0a same-name conversion, c8fca2a4e9930a2dba074ec8d1e161da SDK build, 4caeb953b17fd9172b0d0945bb1c0382 list/marker correction, dfc5f23e8037e5a66d82f85659c37a49 independent verifier dispatch fix (first anchor mismatch found by staged diff), 2384236361225267212f4166cafac04c final exit0: six new auth/oracle tests, four existing S25 setup tests, compiled SDK, Python and shell syntax, format 1349 and YAML 48. New installed native markers required by v1-pr191-s25-scoped workflow. No alternate-pool publication bypass.

### P1b-06d9 — signed package index mirror-race fixture repair

1fbbd07 S25 job 139846416445 failed inside original L1 bootstrap Exec beb7a5fcdc441009bec0f85e6da6a58a, guest exit100 before inner installation/dual checks. Ubuntu noble-updates Packages.xz size 1491848 vs signed expected 1491924; mirror sync in progress, APT refused. Outer real IPv6 on/off/on, lost reply and service restart/SIGKILL original recovery PASS. S1/Forward/quality PASS. Native dual-controller acceptance still pending, no product failure inferred.
Only the signed index refresh is bounded to three attempts for size/hash mismatch, retaining HTTPS, By-Hash where supported, strict update Error-Mode=any and normal APT signature/hash validation. Signature/certificate failures and other errors fail immediately; never retry package installation, reviewed bootstrap or the enclosing original Exec. No relay scope or runtime change. Both pre- and post-Zabbly source refresh use the helper.
Resume 49f298c54e8f2e04b919e277d740b42c confirmed clean 1fbbd07 and unchanged alternate-pool staged tree; later grep returned 1, not environment failure. UNKNOWN list empty. Box edit efa590979c3ede5cd02065b576442011 exit0; initial mocked test 345fff54261ef7cdbdd8017b9bf024ce failed because POSIX sh does not accept hyphenated function names; corrected with private fake executable on test PATH. b6c65aa57825b650d1fcaf86db4bce86 exit0, five index tests plus six dual and four setup tests; syntax and diff check PASS. Registered verify plan discovered; no registered S25 index check, targeted manual fixture test. Native CI pending. Reuse existing failure-learning and S25 fixture, no new BDH asset promotion.

### P1b-06d10 - C11 marker crosswalk and installed refusals

a65e5e2 N2 45799861890 25/25 PASS; actual rollback job 139839963173. Legacy receipt-window repair 5a2c7fb job 139842514033 PASS. C11 consumed marker crosswalk 72/72 (48 unchanged, 24 explicit changed/open), plus six ordered debugger witnesses: see CONSOLIDATION_P1B_WITNESS_CROSSWALK.md.
Add five installed Doctor private invalid-schema refusals and old generic UNKNOWN inspect/challenge/release refusal against retained Update original; no live ledger mutation or runtime change. Box 80229d70937296ed4faed4f099a42acb exit0 entrypoint refusal checks; 0a98851192706efb2b9bb493306454fc exit0 6 new + 5 existing tests; eafdca12c4857424d570aeb8ce74af9a exit0 mapping, six tests and locally built Doctor five layouts. Actual installed proof CI_PENDING. Original accepted native 202 Update NativeAck WAL before/after sync process cut still OPEN, distinct from terminal WAL/trigger faults. Prior alternate pool publication blocker untouched. No retirement waiver or new BDH promotion.

### P1b-06d11 - exact native Update 202 + NativeAck WAL sync cut

5a2c7fb N2 45800601350 all 25 PASS. 69ac198 independent S25 45803277116 job 139851764466 PASS, both same-name native fixtures distinct: L0 original 9e2365b2a30c6297e8fc416799b1238e UUID ec3c9f7d-fe42-4c81-b07c-71371689f7b9; L1 original c3ba945a6ce93a80f687abe791e9874a UUID f7506191-d70d-4205-8846-0d849ae42eab. Actual cross-ID, cross-token, fixed-ledger/ACL/guard, foreign packet and maintenance gates PASS. Mirror-race repair allowed clean inner install; no claim retry branch was exercised by actual APT. 06d8/09 native verification complete at that head.
Convert remaining combined P0 boundary using original restricted native 202 proxy and official SDK plus existing precise WAL marker interposer. New independent hosted modes before/after sync; exact installed binary/fixed DB, test-only marker on original Update VERIFYING event, native accepted ID independently observed, actual original controller SIGKILL, cold/private and live readback require all-or-none ACK and after-sync durability; native CPU2/UUID/ACL/fences and prior rows unchanged. Remove only test DDL; restore original service; original UNKNOWN/claim preserved, typed fresh-key refusal, duplicate original and eight independent read sessions. No native cleanup, replay or release. All old gates retained. Box b0a5b52 clean sync 5a044547b6829c9c5d1cf14479eb12a6; SDK build ca4fa0d4b65ff64fb77f88cc8fedefa6; marker edit 38c54fd6c4536eca4ecb22eac598b113; new harness create 24dfbc18248049ce2b03e26f490d0fc2; 01634b3c0d95bc0d05747e7148670571 exit0 13 oracle tests and proxy build. Actual native CI_PENDING. No new BDH promotion; alternate pool untouched.

P1b-06d13: 3cebdb6 exact-head all seven workflows PASS, N2 45809383716 27/27. Actual combined Update NativeAck WAL both cuts and restart-retained proof PASS (before job 139872941323 original 3411a02aa86dd9310abf8513c34c691a; after job 139872941372 original a0181ebb3035983bb247da51d629c526). P1b installed S2 job 139872941654 host TLS PASS, four guest DNS times nonzero (0.001404-0.002771 s), TCP/TLS connect zero and total ~5 s; all four original curl28/HTTP000, timing correlation PASS. Legacy S2 139872940617 independently also has all four public attempts curl28/HTTP000; simulated TLS toggle PASS, not a real WAN substitute. This narrows the failure to pre-TLS transport; not proof of environment-only cause. Development Box exact sync e41c963a2d75cbded17958190cf953de, guest direct TLS 200 and old inactive inner install with no image (718dd99591ce6b74f1f0f66aa010a413, 6d4279801082946ced12d1233bae9828); not a clean Scoped lab. New installed-only read-only host/NIC identity, NAT/forwarding/base-chain and guest-source TCP/443 conntrack observations bracket the same public attempts. No network policy, proxy, extra destination, or gate relaxation. Build, format, shell syntax, 61 scoped oracle tests and counter parser calibration PASS (13aa81d90b43a9f7ce9115f6de289389), native CI_PENDING. Public WAN stays UNVERIFIED; alternate-pool publication remains BLOCKED, owner backup noted, preserve original tree. P1c/d/e held; no merge or release. L012/L014 existing learning REVIEW_REQUIRED; no promotion.

P1-06d14: [real public WAN and runtime/schema consumer checkpoint](CONSOLIDATION_P1_06D14_CHECKPOINT.md). Actual public IPv4 BOX_VERIFIED at 92ffeda, hosted positive UNVERIFIED; P1c/d source BOX_VERIFIED, CI_PENDING; alternate pool BLOCKED, no waiver.

P1-06d15: [completed-original pin and maintenance separation](CONSOLIDATION_P1_06D15_CHECKPOINT.md). Parent 3df436b 7/7 CI, N2 27/27, all 11 Go race packages PASS. New Delete separation, Scoped pin repair and maintenance snapshot/drain BOX_VERIFIED, native CI_PENDING. Alternate pool remains BLOCKED; no DDL/Legacy runtime removal or gate waiver.

P1-06d15b: [exclusive recovery observers and installed pin witness](CONSOLIDATION_P1_06D15B_CHECKPOINT.md). Scoped PRS callback removed; actual completed-original pin repair witness added to existing N2 lane. Source verified, native CI_PENDING. No retirement waiver.

P1-06d16: [source separation checkpoint](CONSOLIDATION_P1_06D16_CHECKPOINT.md). See exact evidence and remaining gates; no phase closure.

P1-06d17: [source separation checkpoint](CONSOLIDATION_P1_06D17_CHECKPOINT.md). See exact evidence and remaining gates; no phase closure.

P1-06d18: [source separation checkpoint](CONSOLIDATION_P1_06D18_CHECKPOINT.md). See exact evidence and remaining gates; no phase closure.

P1-06d19: [source separation checkpoint](CONSOLIDATION_P1_06D19_CHECKPOINT.md). See exact evidence and remaining gates; no phase closure.

P1-06d20: [source separation checkpoint](CONSOLIDATION_P1_06D20_CHECKPOINT.md). See exact evidence and remaining gates; no phase closure.

P1-06d21: [source separation checkpoint](CONSOLIDATION_P1_06D21_CHECKPOINT.md). See exact evidence and remaining gates; no phase closure.

P1b alternate-pool: [integration checkpoint](CONSOLIDATION_P1B_ALTPOOL_CHECKPOINT.md). Recovered original source; Box-verified candidate, installed CI pending. C03/C05/C11 retained. No waiver.

P1-06d23: [Create availability separation](CONSOLIDATION_P1_06D23_CHECKPOINT.md). Full state/control/recovery Box PASS, independent CI pending. No P1 closure.

P1-06d24: [retained-address reader separation](CONSOLIDATION_P1_06D24_CHECKPOINT.md). Box regression PASS, independent CI pending. No retirement waiver.

P1-06d25/d26: [controller status and Legacy lifecycle isolation](CONSOLIDATION_P1_06D25_D26_CHECKPOINT.md). Box verified; independent CI pending. Legacy bodies retained, not yet retired.

P1-06d27/d28: [pin repair and exact schema DDL isolation](CONSOLIDATION_P1_06D27_D28_CHECKPOINT.md). Box verification PASS, independent CI pending. Schema 6 unchanged. No retirement waiver.

P1-06d29: [Scoped protocol fixtures and alternate-pool proof](CONSOLIDATION_P1_06D29_CHECKPOINT.md). Alternate pool exact 738b269 native CI_VERIFIED job 140222397266; supersedes earlier BLOCKED/PENDING. Hosted public WAN and final exact-head gates remain OPEN. Fixture conversion Box PASS; independent CI pending.

P1-06d30: [generic Scoped admission fixture checkpoint](CONSOLIDATION_P1_06D30_CHECKPOINT.md). 17 files, 57 calls, original assertions unchanged; full State/Control BOX_VERIFIED, independent CI pending. No legacy-specific witness retirement.

P1-06d31: [Scoped lifecycle pin and generic recovery fixtures](CONSOLIDATION_P1_06D31_CHECKPOINT.md). Keep Legacy assertions; staged verification and independent CI evidence tracked in PR.

P1-06d32: [Scoped recovery receipts and worker faults](CONSOLIDATION_P1_06D32_CHECKPOINT.md). Source candidate; retain all prior witnesses and native gates.

P1-06d33: [recovery observation isolation](CONSOLIDATION_P1_06D33_CHECKPOINT.md). Legacy witnesses retained; independent CI required.

P1-06d34: [host forwarding diagnostic comparison](CONSOLIDATION_P1_06D34_CHECKPOINT.md). Read-only evidence candidate; public WAN remains UNVERIFIED.

- D35: [Explicit Scoped admission and recovery fixtures](CONSOLIDATION_P1_06D35_CHECKPOINT.md).

- D36: [Pure Scoped Peer lineage and Legacy decoder isolation](CONSOLIDATION_P1_06D36_CHECKPOINT.md).

#### R2/R3 continuation: exact audit and Create provenance

- Converted shared audit positives to explicit retired-reader refusal; 11 drift negatives run against a valid Scoped baseline. Exact 1,024 identity boundary covered by existing Scoped witness (Operation c2b1b9f5804730415d9c5317293166a8, guest exit 0).
- Missing native ACK remains UNKNOWN, no invented ownership, recovery audit or lost claims. Recovered Create/Delete retirement validates exact event, linked recovery receipt and audit; independently corrupted rows refuse (aa7f4613228a40104a1a83d99386176b, exit 0).
- Found genuine fail-open in active Create lineage: native ACK plus arbitrary SUCCEEDED event accepted a recovered Create with damaged recovery audit/event before Delete admission (519d4027408121378852de30849abaa3, exit 1, two expected negative failures). Replaced legacy no-ACK predicate with exact native ACK + unique terminal event + exact linked scoped recovery receipt/audit for recovered Create. Valid recovered Create and both corruption negatives, related Create/Delete recovery tests pass (5e939d6e5b385d043f6b3f382cf558db, exit 0).
- These are SQLite/guest source witnesses, not native CI. No final regression count yet; R2/R3 remain open. Preserve old uncommitted schema/BDH work and no remote publication before coherent gate.

## P2 R2 checkpoint — stopped Scoped entry (2026-11-27)

R2 remains IN PROGRESS, not C07 PASS. The settled stopped-controller
slice reuses the existing exact Scoped drain and final retirement,
instead of reactivating the Controller or restoring Legacy execution.
The offline review binds its own entry intent and requires inactive
Controller, already-active Incus, current Scoped identity and unchanged
native/ledger/guard evidence. Populated scope uses durable business
originals and approved-prefix resume; empty scope uses the existing
Scoped archive/restore/purge chain. Offline ledger reads use the
bounded cold-copy proof. Distinct terminal approval and original-mode
resume remain mandatory; historical offline receipts refuse and require
their matching historical artifacts.

Box focused regression: 104 tests PASS (original
65337d87a4e766ac86322607bfbcd321). Includes stopped/active capture,
schema7/cold read, mixed and foreign scope, unknown refusal, changed
review, distinct approval, resume and historical archive tests.
These are synthetic source tests, not installed native acceptance.

The inherited bounded power UNKNOWN archival obligation is still open.
maintenanceRetirementScope already classifies exact owned power intent
without settling it. ReadScopedDrainWitness and scopedDrainRoot require
no nonterminal originals; ReadScopedRetirementLedger requires an empty,
terminal ledger, released bindings and exact Create/Delete lineage.
Therefore ordinary drain or changing only schema/mode cannot implement
the exceptional archive-only authorization. Do not weaken these guards,
release/replay an UNKNOWN, invent Delete receipts or restore Legacy.
Next R2 slice requires a distinct archive-bound proof of exact original
claims, native ownership/preimage and quiescence, with monotonic durable
retirement-prefix evidence and independent poststate. Reuse common
Scoped archive/restore/purge only after that proof. Keep existing
negative tests, old N2 gates and C07/C11 open until legitimate equivalent
proof. R3 native acceptance of the stopped entry remains required.

R1 exact candidate ceee966: Forward 45990043049 PASS (fresh job
140475255122 verified Controller Doctor, stale refusal and normal
retirement preimage). S1/quality/dual-arch actual failures remain
the two inherited H tests. N2/S2/S25 were still running at this check.
No P1 H/C04/C01 waiver or release authorization.

### P2 R2 — archive-bound bounded power UNKNOWN candidate

Source candidate BOX_VERIFIED for the minimum exceptional stopped scope:
all current instances independently proved stopped and exact-owned, terminal
Peer/Forward policy, unchanged original Root Create/retired Delete receipts,
cold schema-7 ledger and retained exact power-stop UNKNOWN claims. Separate
root-local review and archive-bound intent/ACK/verified prefix; no new
business Operation, claim release, outcome rewrite or blind replay. A pending
intent without independently verified exact next absence refuses; an
unrecorded or non-prefix native change refuses. Exact archive receipts join
common retirement, resume, final native restore, cleanup and purge proof.
Ordinary drain/empty retirement still require their original settled gates.
Foreign, mixed, unproved or unsupported scope refuses. R3 must test actual
installed positive and negative lifecycle, including interrupted effects,
archive bytes and original UNKNOWN retention, purge/reinstall and LXC.
No native effect executed during this source slice.

Box focused checks 7153e2177a585efcfd328b3eae6bbd3d exit0: new state
eligibility and drift negatives, native prefix/journal/no-replay tests,
related existing Go witnesses, and 64 Python uninstaller tests PASS.
Full affected Go regression 078d9442c84a37ca453f73d7cc790f28 exit1:
State PASS (141.549s), CLI PASS (1.530s), Install only inherited
TestS4B2ConntrackScopeMustComeFromFullSQLiteLedger and
TestS4B2RootAuditRefusesSQLiteAndNativeInterleaving FAIL; vet and format PASS.
No skipped H checks or altered old tests. Independent CI_PENDING.
Previous exact 818c982: S2, S25 and Forward PASS; N2 26/28, two
historical legacy-only lanes remain until R3 disposition; S1/quality/
dual arch still fail inherited H. No C07/C11 acceptance, P1 H/C04/C01
waiver, merge or release. Next R3 installed witness and equivalence
review, with real gaps repaired before advancement.

## P2 R3 — installed offline witness candidate

Baseline 3ba1029 exact CI: S2 46032710003, S25 46032709975,
Forward 46032710011 PASS. N2 46032710024: 26/28, actual normal
retirement 140620280975 and generic Scoped recovery 140620281284 PASS;
legacy 140620280987 and legacy-offline 140620281007 reject unsupported
fresh Legacy installation. All five failed source/dual-arch jobs contain
the same two inherited H tests, not new R2 failures.

Add three distinct hosted-only lanes (empty, settled populated, exact
power UNKNOWN with recorded real Native ACK). Reuse the existing
disposable installed D4 fixture with a Scoped power mode and exact
restricted Native receipt, withhold business finalization only.
Bind exact candidate/host/project and compare original cold bytes,
claims and complete operation set before/after archive. Exercise active
and declined refusal, foreign ACL refusal, unrelated empty project
preservation, verified native prefix interruption, post-native cleanup
interruption, explicit resume, purge interruption/reinstall. Extend
the existing actual missing-ACK Stop lane with a read-only offline
refusal; never change its retained original. No native uninstall in
the development Box, no fabricated ACK, no production test hook.

Focused local candidate checks 21e1f14dfa6a02d56e324356dab0b0a6
exit0: 4 Go offline journal/prefix tests, vet, Python 64 uninstaller,
4 identity, 14 S2 mapping and 3 CI gate tests. Dedicated installed test
correctly skipped outside hosted lab; not native evidence. Earlier
d01b9795d47c0eb52b30901449c2072c caught missing three-lane partition;
explicit separate C07 lanes added without changing 72/23 S2 coverage.
See acceptance/inventory crosswalk. New lane actual CI_PENDING.
Old Legacy gates retained pending evidence and explicit disposition.
Historical LXC normal/empty evidence remains bounded; exact new
exceptional inherited-policy native proof needs authorized clean
nested fixture. BDH plan b93b46ef1e81d3abcb4bc4f7e4001874 reused
installed-lifecycle route; L012/L014 review-required remains open,
no BDH promotion.

### P2 R3 continuation — N2 trigger and fixture calibration

Head 5a3af5a: exact-source Forward 46055735367 and S25 46055735392
PASS; S2 46055735366 pending at observation. S1 46055735388,
quality 46055735328 and dual arch 46055735426 retain the same two
inherited H failures (actual logs checked, five failed jobs). No N2
PR-triggered run returned by the available commit-scoped interface.
Actual PR diff 527 files; matching workflow path is present (position
2 in local sorted diff). Pinned actionlint 1.7.7 validates the original
workflow (f746a1559334a68406ef1c13b523a3e8); missing run cause not
established. Do not infer workflow invalidity, skipped gate or PASS.

Consolidate 143 individual N2 trigger patterns into 10 dependency
families/exact files, programmatically checking each old pattern is
covered; retain same-repository/base and all 31 jobs and assertions.
Pure documentation paths remain excluded. This conservative broadened
trigger is a candidate, not proof of the missing-run root cause.
Adapt the exact trigger tests to evaluate patterns rather than literal
filenames; preserve existing receipt ordering and 72/23 S2 coverage.
Tighten offline active and foreign refusals to require the specific
oracle and unchanged native state; restart observation waits only
within a bounded window for full coverage plus exact retained original.

Local regression a61b1c8cbbe799a35ef57acdd987b74a exit0:
14 S2 map, 4 identity, 3 CI gate, 64 uninstaller Python tests;
actionlint, format (1415 files), YAML (48), import PASS.
Actual Native lanes remain pending. Existing LXC labs preserved;
no target installation or destructive action undertaken. No legacy
gate retirement, H/C04/C01 waiver or R4 advancement. Learning:
literal-path tests can miss equivalent wildcard coverage; keep
behavioral trigger checks and distinguish missing CI metadata from
a confirmed trigger failure. No new BDH asset/promotion.

### P2 R3 — N2 pre-job failure diagnosed

Repository-level all-event read found exact 82c41e7 N2 run 46057428761
(push, failure, zero jobs) and 5a3af5a N2 run 46055728195 (push,
failure). The PR-filtered listing omitted these. Both candidates' N2
run block is 21899 characters; prior 3ba1029 successful-start N2 block
was 20479. GitHub documents a 21000-character limit:
https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-syntax#jobsjob_idstepsrun
Concrete violation matches the pre-job failure; platform annotation
unavailable, actual corrected launch still required.

Extract exact original marker assertions to scripts/pr191_n2_installed_gate.sh
(21417 bytes, SHA256 7ec0fb370148d70ecd5b3c72d0f1ee6c2fc77661773ff7c22b5cd02c6ea31012).
Preserve fail-fast shell, transcript, identity check and gate-before-receipt
ordering. Remaining inline blocks 482 and 533 characters. No removed
lanes, assertions or changed authorization. Add inline budget regression.
Local bffee141821127d97088ba4b54113e10 exit0: byte-for-byte extraction,
bash syntax, 22 evidence tests, actionlint, format and YAML PASS.
CI launch and native proof pending. Learning: check all event types
for missing PR runs and platform limits beyond syntax validation.
No BDH promotion, no R4 or retirement gate waiver.

### P2 R3 — actual N2 feedback and clean LXC fixture

Exact bf36c5f N2 46058382872: 27/31 complete successful jobs.
140706362774 settled populated offline full retirement PASS, actual
identity b62c440b2f6255a87e8c3dba16de4d60d539681a305dd11d101d7cfc581dee83;
140706362809 missing-ACK original retained and offline refused PASS.
140706362693 exact UNKNOWN review, foreign ACL refusal PASS, archive
created then progress refused before first new native effect.
140706362716 active refusal PASS, cold original fingerprint changed
after empty review. Both Legacy lanes still fail at unsupported fresh
Legacy installation, not at a current supported runtime scenario.
140706362734 and 140706362475 retain actual normal retirement,
interrupted resume/purge, current scoped reinstall and generic reviewed
recovery/no-replay/decline/partial original refusal.

Correct two concrete source defects. Settled retirement previously used
the live read-only SQLite connector even with controller stopped, which
can modify WAL-index sidecars. Select existing strict validator on
exclusive cold private capture when inactive; keep active live reader,
same exact settled-inventory requirements, no changed authorization.
Exceptional archive file inventory incorrectly classified the required
ledger.sqlite.serve.lock as an unsupported sidecar. Bind and verify
the exact persistent lock bytes alongside main/WAL/SHM; refuse missing,
duplicate, rollback journal or unknown ledger-prefixed file. No original
settlement, replay, lock release or weakening of root archive binding.

Focused regression ee44ecb7c9fb475b314f005b173c89cb exit0: four
state tests (incl new byte-preservation cold capture), five install
tests (incl archive inventory positive and negatives), vet, format.
Full affected state/install/CLI d5061d3b83172cdd4c030bef78e9a624:
State and CLI PASS; Install FAIL only the two inherited H tests
(6 subcases); no new package failure. Native exact-candidate rerun
required, do not infer C07 PASS.

Owner authorized a new Box for R3. Create 28269867c4c385cce54cf1eff940088e
SUCCEEDED, native f054c452-0897-4eb2-bc39-c8a95bb17e49;
pr191-p2-r3-lxc 91bd9fb7-ae4d-46c4-ab35-7ca6b7f069f9,
user-1001, standard nesting, 4 CPU/4 GiB/24 GiB. Preflight
5372fa4d8c80a95558fa1cc7192fd655 exit0: Ubuntu 24.04.5 LXC,
no installed Incus/BoxWeave/retirement state, mapped UID/GID,
AppArmor unconfined (functional-only, no recursion/isolation claim).
Signed dependency installation 75bf430d8640d0a1f4f41a9cadb3c879 PASS;
recognized inherited policy SHA256
fcbd512cb8212c6f5720f28398f01e85ff96246aebb73e09cf449d2f3aa2c5b0
unchanged. Exact previous source and Go 1.26.9/module cache
aa6a5ff4109f1a8d32437848b3838936 PASS; signed Incus 7.6 package
catalog 71f6c43499ac7922fae4f27d21f19188 PASS.
Do not reset previous labs or spoof hosted-only fixture identity.
Current candidate installed LXC witness not yet executed.

### P2 R3 — offline positives and normal handoff regression

Exact 8786dc2 N2 46059894552 completed 28/31; candidate SHA256
e1dd4cdf76315490eb321cd2a1ea3d3e467a997667c604f1f2bd8b21b2502db9.
Empty 140711332479, settled 140711332302, exact UNKNOWN archival
140711332393 and missing-ACK refusal 140711332606 all PASS.
Original lane reached verified native prefix resume, unchanged original
UNKNOWN/claims/all operations, independent native poststate, post-native
interruption/resume, foreign preservation, interrupted purge and fresh
ledger/new workload. S2 46059894513, S25 46059894537, Forward 46059894575
PASS. Quality, S1 and dual arch remain red (inherited H).
Normal retirement 140711332374 passed live plan, foreign ACL refusal,
retired history, then blocked at SCOPED_DRAIN_EMPTY_RETIREMENT_BLOCKED
after approved drain, before archive/restore. Exact lower-level blocker
not printed. The 8786dc2 cold-reader change affected this normal stopped
handoff as well; the preceding bf36c5f normal lane had passed.
Restrict cold capture to explicit settled offline entry; keep normal
read-only audit contract and original validation. Bind cold mode to
native review digest and use it through every wrapper phase. No cold
fallback, unverified scope, altered archive/UNKNOWN or safety bypass.
Add bounded blocker detail at handoff. Focused Go tests (10), CLI, vet,
65 uninstaller and 15 evidence-map tests, format/diff check PASS in
7ed22391ae50c0a4abcb0fac13eb628a; exact corrected native rerun required.

Actual fresh LXC witness on 8786dc2 (see PR comments 7227025109 and
7227076542): owner-approved 91bd9fb7-ae4d-46c4-ab35-7ca6b7f069f9.
Exact installed candidate b85ef75340a41140ba63cb8c1a57d844a16e777b781f9d72df5207f3f598d4a6.
Native bootstrap 5242c2ed2c79cf8b681e4f1d56e5a33d and install
38a5b86d435983f376dcbd0000b0cdb5 exit0. Cold plan/active refusal and
effective LXC review b7ca2606a2965e0ffe9483df9da4840e PASS. Declined
approval unchanged and actual offline empty COMPLETE
3ada5ec4dfc358444a10535700440cd9. Independent archive/native postcheck
9a3abbab77f7c42a7baa5c68a16b74f3 PASS: original five file digests,
Incus PID 7257 and exact inherited LXC hash unchanged. Preserve
retired-xp38o5al; no reset/purge of lab. Functional-only scope,
not strong isolation or exceptional populated LXC certification.

Legacy gate audit: I3 native Project description drift is a generic
safety oracle, not retired. Add equivalent installed Scoped empty
Project drift refusal/no-effect fixture and required marker to normal
retirement lane; preserve every earlier gate and fixture. Owner decision
for obsolete-only Legacy semantics remains required. See acceptance
crosswalk; no lane removal or R4 advancement.

### P2 R3 — owner-authorized obsolete N2 gate retirement

e674d15 N2 46062127023 normal 140718649872 including new I3 equivalent
PASS. Three offline positives 140718649957/912/140718650010 PASS.
Thus normal handoff correction verified on actual installed candidate.
Owner approved retire Legacy-only gate requirements after equivalent
proof; see acceptance crosswalk. Scope limited to matrix/gate consumers
and fail-closed old selectors, not historical fixture or product code
deletion. Local 6afd9027929909a199d935cb60b0ac27 exit0: exact remaining
29 lane order and unchanged active assertion bodies, retired selectors
refused before setup, 24 evidence/quota tests, actionlint/shell/format.
Learning: distinguish retired product requirements from retained
generic safety proof; verify exact retained gate set. No new BDH asset.
Current quality/S1/two-arch logs still fail only inherited H tests;
retain formal acceptance blockers. No R4 advancement until R3 exit.

### P2 R3 signed / P2 R4 exact release consumer candidate

R3 signed on exact ebf1d9c6e0d724ac1474c9e179789bf36255d5f6,
N2 46063354636 29/29 PASS, S2 46063354608 (72 markers/23 original
producers), S25 46063354620 and Forward 46063354635 PASS.
See [signed review](https://github.com/vanishs/boxweave/pull/191#issuecomment-7227714246).
No full PR/P1 acceptance. The earlier local LXC functional evidence
remains bounded; preserve archives.

R4 replaces the obsolete direct-serve release S2 invocation with
the existing installed Scoped producer and exact CI consumer model.
A controlled RC push starts the same 29 independent N2 lanes only
after exact protected-v1 HEAD and VERSION/ref checks. RC candidate-a
and S25 use the distribution build flags and version stamp; retain
candidate-b distinct negative and all existing native assertions.
S25 is still a real independent release job on its own clean VM.
Release S2 depends on S25 and the actual two-arch manifest job, then
observes exact same-source, same-release-ref push N2 run/attempt and
the current release run's completed S25/manifest jobs (not its own
unfinished run). Require the 23 mapped receipts and seven additional
identity-only receipts after all original N2 gates: 30 distinct job,
artifact, transcript identities and exact amd64 asset SHA256 for each.
PR receipts cannot authorize RC. No automatic rerun, dispatch, release
branch or tag creation. Verify the full manifest checksum; pass its
digest to the existing publish dependency and compare again before any
tag/publication effect. Existing source, dual-arch, immutable assets,
readback and stable-signature policy unchanged. Keep historical direct
S2 scripts and non-191 PR route for P4 caller consolidation.

Box 9d374e18-196a-493d-bf1f-2d582589f04d /root/pr191-p2-ready:
0a4789a2c63ceaf5482e52298aaa6d95 exit0, 68 focused/adjacent tests,
four workflow actionlint, shell/format/diff checks PASS. Two actual
local RC-stamped amd64 builds at parent Go source ebf1d9c both SHA256
32669bb5f8552d6fee9ce14df7cfeda1071b173545cae99a2de2e11b3b0e65da,
build-info exact. This proves same-Box rebuild identity only.
0ab4c1dda61df9975522943dcc0d819e exit0 adds complete 30-lane synthetic
consumer success + wrong unmapped digest refusal (10 gate tests total,
69 distinct focused/adjacent tests); original map, N2 assertions, S25
checker, manifest and stable-gate code byte-identical to parent.
Initial actionlint caught unused undefined inputs.release_tag in original
release YAML; removed that stale reference and reran PASS. Oversized
source-edit argv refused at admission; bounded source edits verified.
No native operations replayed or new BDH assets. Learning: exact
release event and artifact identity cannot be inferred from PR PASS;
independent rebuild divergence must fail closed.

Independent PR exact-head CI pending. Actual release-only runner/
cross-run build identity and current-attempt release artifact provenance
remain UNVERIFIED until separately authorized RC execution; do not
manufacture release context or publish to test. Inherited H/C04/C01,
final C11 and release-only proof explicitly open. R5 not started.

### 2026-11-27 — P0/P1/P2 and BDH preservation audit

P2 scoped development signed at 09152ff14b0fb75e73b0492d2a1d171bcc87499d:
N2 46066576792 (29/29), S2 46066576966 (72 markers/23 jobs), S25
46066576692 and Forward 46066576878 passed. See PR #191 comment
7228099703. BDH framed export delivered at 720f3ba9d1b5ad24d64858e880794049dcbcb2ce
(comment 7228231141).
Reconciliation recovered the D37 pure-peer regression and committed-range
BDH helper without replacing newer functionality. Historical source deltas,
phase histories, handoffs, verification and research are preserved under
[BOX_RECONCILIATION_20261127](BOX_RECONCILIATION_20261127/README.md).
Earlier acceptance obligations H, C04, C01 and C11 remain open; P3 R0 is
not started. This inventory is not a release or merge authorization.

### P3 R0 complete / R1 CLI contract candidate

Owner defers H until the OpenAI Support response and a new instruction;
C04 remains P5. See [R0 inventory](https://github.com/vanishs/boxweave/pull/191#issuecomment-7229390282).
Baseline d9a1eddc5f174ae1e7175257e424a4b262432690, tree
129947b3dc91df6d214489cde584e64d6291a684. Current owned Box/worktree
reused. R1 replaces the shared 41-flag parser with a classified 45-entry
active CLI inventory, command-specific accepted flags and successful
help. Unknown names and three retired BoxWeave mutation names fail
before generic parsing; legacy mode=false still refuses. Six fixed
internal entries, root/restricted workers, generated unit arguments,
all 16 Incus dependency leaves and P2 offline/normal scoped callers
remain. Existing explicit scope assertion stays for current service/
wrapper consumers. Installer help omits five obsolete selectors but
preserves their refusal before candidate access. Existing legacy helper
bodies and historical uninstaller recognition remain for R4/P4
caller/evidence disposition; no historical source/test deletion.

Local CLI full package and vet PASS (70ac6b8ca5553298b30ba5149622c588);
actual CGO binary help/refusals, eight adjacent Install tests and 102
Python installer/uninstaller/manifest tests PASS
(afc0780ff378adce6bd4f428be2eea6d; initial Python invocation lacked
PYTHONPATH, corrected). Final parser/installer-help revision CLI package,
vet, 2 wrapper contract tests and shell/diff checks PASS
(5ba865dda0be0a79acb20e86fb56049d). Actual help probe build is a local
candidate, not installed/release proof. Old generic recovery action
description corrected, reviewed plan/apply/guided and scoped GC
authority unchanged; R3 owns recovery-specific contract. Final
exact staged-tree and independent CI evidence are separate. H two
tests untouched; no installation/recovery/native maintenance effects
or waiver. R2 Doctor and maintenance classification remains next.

### P3 R2 — current Scoped Doctor and Maintenance candidate

Baseline 644a2fbb4ab7a40745aeecdcef9543ffd9c7487f; owner H freeze
unchanged. R1 S1/quality/dual-arch failures are the same two inherited
H tests (five jobs), Forward and S2.5 PASS; S2/N2 still running at
inspection. No H change or waiver.

Doctor distinguishes a proved historical installation (UNSUPPORTED_LEGACY)
from incomplete and conflicting observations. Schema-7 preflight now
requires verified current manifest/runtime/scope, active guard/lifecycle,
ledger and local HTTP observation; restricted Root-only coverage remains
separate from a fully proved mode. Legacy healthy projection removed.
Read-only independent ledger/native observation can still report evidence
when the exact Scoped controller is stopped or partially observed. Runtime
argv agrees with R1's accepted flags and fixed loopback listener; infer
the current Scoped default only from the same installed observer binary.
Root classification requires the existing static unit contract.
Scoped SQLite capacity is available to the restricted observer without
pretending to observe Root policy; the cross-capture digest binds peer/
identity capacity and foreign row count. Shared redacted count detects
foreign ledger scope without exposing its identities. Missing Root policy
cannot produce HEALTHY_OBSERVED. Maintenance retains historical/pinned
adapter and artifact observations but refuses to claim current Scoped
coverage for legacy or mixed identity; no retirement authorization or
approval is derived from observation.

Initial full CLI/Diagnostic/State (140s) and targeted Install/vet PASS
(operation 7cd8c83395424a699419240d77df25d9). After refinements, full
CLI/Diagnostic, selected D1-D3/maintenance/foreign/epoch State and
Install regressions, all four package vets, format and 13 scope
tests PASS (56d9de514abe21263dcce82a05b736b9). Full State initial
revision and final targeted tests are distinct; full final candidate
independent CI remains mandatory. Existing N2 H3B/H4B/postflight and
retirement observations must revalidate affected behavior. H tests
not rerun locally or modified. R3 owns reviewed Recover/navigation
convergence. No product maintenance effects or deployment.

### P3 R3 — current reviewed recovery and R2 postflight follow-up

Baseline ba2da1ffc810accbdc3b02048c07aa83947e7acc. R2 exact CI:
N2 all 29 jobs, Forward PASS; S25 fails at installer CONTROLLER_DOCTOR,
S2 crosswalk consequently reports missing/failed S25. S1/quality/
dual-arch five failed jobs contain only the two inherited H failures.
N2 scoped job 140775130050 proves H3B/H4B capacity parity and churn,
H4A navigation/busy/replay, D3 positive and partial refusal, guided
decline/apply and D4 no replay; original cb305e47a214cc4f1149e59520906c8e.
SDK job 140775130401 proves previous/current SDK and exact 23 discovery.
Prior-source installed evidence, not R3 exact signoff.

Local reviewed plan/envelope/writer agree on SCOPED_V2 and settle-only
supported leaves; historical local bind/pin leaves cannot authorize a
new review. Scoped Create/Delete retain existing MCP route, not local
apply. Preserve current same-key receipt lookup and historical
read-only links, no replay or authority expansion. Align fresh ledger
mode with Doctor; reject foreign count in plan, service and final
transaction. Restricted worker rechecks current identity. MCP
description reflects Scoped originals and unchanged 23 tools.
No H work. Existing GC and P1 ACK/WAL proof retained.

S25 actual failed Doctor JSON was discarded by wrapper; precise
failing predicate remains unknown. Standardize restricted Doctor
environment using service_command, emit bounded typed failure reasons,
allow at most three identical read-only Doctor observations for
verified current identity/schema/ledger and specific startup/runtime
gap. No install retry, no gate relaxation, no acceptance claim until S25
and S2 independently pass. Retain failed originals.

Source test 0ef058844131e3c3b32416f1d23d0172: full CLI/Diagnostic/
Recovery/Control, selected State D2-D4/S5/P3R3 and five vets PASS.
Python old positional mock failed for new env prefix, corrected exact
caller fixture without changing assertions. Recheck
21d14575697c59b0158331d995cd07aa: 106 Python, format, 13 CI scope
tests, shell and diff checks PASS. R3 installed verification pending.

### P3 R4 — caller convergence and S25 postflight repair candidate

Baseline 52f1ce9013208470f10b07c8cf26e277cd46ac8a. See
[P3 caller disposition](P3_CALLER_DISPOSITION.md). R3 Forward PASS;
S1 failure log contains only the two frozen H tests. R3 S25
140781584191 fails at outer CONTROLLER_DOCTOR: manifest, identity,
runtime, dependencies, guard, lifecycle, SQLite, native and MCP observed;
Scoped artifacts UNVERIFIED. Source shows the helper uses umask 077
and scopedWriteExclusive used MkdirAll(0755), so the new systemd
drop-in parent can be 0700. This explains the inaccessible Lstat;
the exact failed runner directory mode was not separately captured.
No startup retry or readiness weakening solves that predicate.

Fix only newly created root-owned systemd metadata directory to the
intended 0755 mode; preserve existing directory modes, file umask,
exclusive write and private journal permissions. Root temp-fixture
regression under 077 proves new 0755, file 0600, existing 0700 and
exclusive original unchanged; no actual host installation. Current
S25 outer/inner entrypoints now reject old or omitted mode before
effects, existing PR/release Scoped callers unchanged. Add explicit
obsolete CLI refusal calibration, not live audit/purge proof.
Align six live docs with schema 7, current recovery and native
dependency distinction. Historical bodies/workflows remain P4
inventory, no deletion or H changes.

Local verification 762a88127e6c0a295e8b1901dd6d56d6 exit0:
root fixture PASS, CLI/Diagnostic full packages, selected Install
D1-D4/P3/H1A/unit checks, vet, shell, 48 Python regressions and format.
Actual S25 and dependent S2 exact-head independent proof remains
mandatory. No formal R4/P3 signoff yet.

### P4 R1 — bounded obsolete implementation retirement

Baseline 13fee95b6c03c3e0d5b70c39806594c99d47cfe3. R0 ledger
[7230209749](https://github.com/vanishs/boxweave/pull/191#issuecomment-7230209749).
P3 exact N2 46082177305 29/29 PASS, S2 46082177119 PASS, S25
46082177116 PASS, Forward 46082177167 PASS. N2 scoped 140783414980
actual D1-D5, H3B/H4A/H4B capacity/recovery and partial refusal PASS;
SDK 140783415189/191 exact 23 discovery, old/current SDK PASS.
These close P3 installed regression within scope at predecessor,
not final release. C01/H and C04/C11 final obligations remain.

A01-A03: all Go references to BackupReceipt/Upgrade/Rollback/
CommitUpgrade and subordinate helpers were contained in lifecycle.go
and finalize.go; remove dead product upgrade bodies, retain exact
Activate and shared helpers. Retire install.sh uncalled guided upgrade,
old fresh review and their private helpers. Keep old-selector refusal,
metadata/receipt checks, installed fresh/Doctor/OAuth, native dependency
lifecycle and historical recognition. Eight old wrapper test methods
were exclusively direct retired-upgrade tests; preserve source/history
in Git, current generic decline, stale, identity, schema and refusal
oracles remain. Retain test_guided_upgrade.py current metadata, Scoped,
LXC and reader prerequisites tests; historical filename still used.

A08: remove only unreachable Legacy branches in s25_inner_setup.sh
after existing mandatory Scoped entry selector. Preserve exact live
commands, checks, markers and setup helper (including stale negative
and exact image). Outer/N2 transitive historical callers move with
R2 mapping; no producer/checker/marker edit. A04 shared
nativeScriptletAcquireLock still referenced outside family and old
workflow calls; A05/A06 old archive/uninstall consumers need explicit
reachability/disposition. A07/H remains frozen. Do not bulk delete.

Local operation a732c74812efe0bb2647af91e6917277: CLI/Diagnostic
whole packages, selected Install, vet and 98 Python tests PASS;
aggregate exit1 only format saw unstaged deleted file. A second
format invocation lacked gofmt PATH (b9ac68d202a1b360b6dfcdc0f7462544);
corrected staging and toolchain in d74873d52e28049f5912668f5fbedceb,
exit0: shell, 24 S25/crosswalk tests, full tracked format PASS.
No complete Install package or independent new-head claim.

### P4 R2 — native CI routing and release evidence convergence

Baseline 17afed0fe6b39365e1d2aad2392b2126f98a7db3. R1 quality
job 140787860760 exposed one new Python error: the old S25 ordering
oracle still searched the retired direct inner stage path. Move the
assertion to the actual wrapper stage/activate/status/Doctor order,
helper exact apply/active proof and both outer/inner helper/token/
Doctor ordering; preserve readiness checks. No H changes.

Unify N2/S25/S2 PR path filters to complete source/script/wrapper
families and VERSION, not docs. Replace #191-only special routing with
the same-repository v1 PR Scoped contract and actual positive PR ID,
exact source/event SHA, PR merge ref, consumer and producer
workflow/job identity and run/attempt. Historical S2 non-PR direct
route remains only on its original branch. Reject zero-size locator,
wrong repository ID, missing/wrong producer identity; retain all
existing artifact and receipt checks. Future PR context is modeled,
not an actual different PR run. Both old S25 manual workflows limited
to original historical branches. Archive obsolete guided-upgrade
workflow and its sole historical script at parent 17afed0f; current
Forward fresh/refusal coverage remains. See [workflow ledger](P4_WORKFLOW_DISPOSITION.md).

Operation a35daccafe16320f07af136132446dbe exit0: 50 targeted
tests, YAML and format. 582e6eeefabb3114ea034324ec16a6f8 /
native b52020d1-7ce0-4206-9c5f-4e548de28fa8 exit0: all 477
Python tests, 47 YAMLs, 1437-file format and 13 CI classification tests.
RC complete modeled consumer, stable and manifest checks remain;
no actual release or native execution here. R1 S25/Forward PASS,
N2 28/29 at observation, S2 pending; other Go failures remain
frozen H. New-head CI required; do not certify P4 or release yet.

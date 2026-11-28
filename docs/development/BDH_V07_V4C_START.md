# BDH v0.7 V4c  actual utility feedback and orchestration feasibility

Status: V4 functional PASS; utility gate NOT MET. V4c investigation
started, real witness outcomes archived; independent terminal reopen
review required before selecting and claiming the next improvement.
No new production code, runtime or product capability, promotion,
parent change, merge or release.

## Reconcile the real V4 outcome

Exact starting child f8d24629641df1b9dbc87d21673dc114625f88be,
parent #191 3494ae8a35c50c386686c9d9d675c1ae37e1b1b7,
canonical v1 9d07665dcfc5ae6345d6bfcd0d71de7b6510b51d,
scoped implementation fcceaf11ec271a6ed7662580f65c59e8c8f164dd.
Applicable delivery CI 45604806376 and 45604806453 SUCCESS
(native S2/S2.5 and release skipped).

- [Lane A original report](https://github.com/vanishs/boxweave/pull/193#issuecomment-7171335596):
  old Skill, normal/interrupted 5/8 parts, 32/49 case MCP calls,
  17 common, total 98 BoxWeave calls. Both exact bytes, collision
  and controlled recovery PASS. 16 orchestration batches includes
  setup, not a comparable manual-artifact primary score.
- [Lane B original report](https://github.com/vanishs/boxweave/pull/193#issuecomment-7171771903):
  actual later ordinary task uses planner, 3/4 parts, 89-90/109
  case MCP calls (198 confirmed, at most 199), 8/10 manual evidence
  objects, 9/9 glue submissions. Overlapping categories must not
  be summed as the pre-registered primary metric. Both bytes,
  collision and controlled recovery PASS. Distinct logical work
  actor, not blind, not separate authenticated human identity.
- [Originating comparison and fresh whole-byte readback](https://github.com/vanishs/boxweave/pull/193#issuecomment-7171854519):
  all four final files independently reread through actual MCP,
  matching both original sources, four collision failures verified,
  original recovery operations and final streams inspected. Case-only
  calls 81 vs 198-199. Different chunk sizes, additional safety/evidence
  work, different manual counting, nonblind exposures. No valid primary
  score or causal efficiency PASS. Actual reuse and stronger explicit
  continuity are valuable but not sufficient for promotion. No time or
  token estimates. Controlled inter-call pause only, not lost ACK/crash.
- Entry reconciliation d1b418ec1ab6b42c51cf5d71ace334fc exit 0:
  24 learning and 15 episode revisions byte-identical in canonical
  external and repository histories. Exact clean main/scoped/parent
  checkouts. No edits to trial lanes.
- Real witness import cd5a74dfc72c4cd7c63cd53e34b21860 exit 0:
  [L014 rev25](../../dev/learning/BDH-L014/00000025.json) old baseline
  helped + residual (98 reported calls, inclusive common); [rev26](../../dev/learning/BDH-L014/00000026.json)
  actual B use, uncertain overall utility + residual (198 confirmed case
  calls, upper bound 199). Preserve differing denominators. B attempt
  uses its actual second transfer reference pr193-v4-b-2, report covers
  both. Imported after performance, no fabricated advance claim.
  [Episode rev16](BDH_V07_EPISODES/BDH-E-343f1501118ad8d8c3dfe6615c718dc0/00000016.json)
  claim imports actual B actor, [rev17](BDH_V07_EPISODES/BDH-E-343f1501118ad8d8c3dfe6615c718dc0/00000017.json)
  records actual outcome, not positive benefit. State
  OUTCOMES_RECORDED_REVIEW_REQUIRED. Eight of original twelve attempts
  used. Existing resolver requires a terminal reviewer distinct from
  implementation and benefit actors. Do not invent one or bypass
  outstanding selection. An independently reviewed reopen should bind
  exact rev17 and the matching actual B observation at rev26.

## V4c initial research on the actual surface

The current Code Mode connector surface has fresh isolated execution
per invocation; no persistent local filesystem or dynamic imported
repository module is exposed there. The guest cannot be assumed to
possess connector access or credentials, and no authorized persistent
client-side connector host or Agent delegation capability was discovered.
Do not use guest/host secrets, exposed sockets, new product MCP tools,
or alternate transport to simulate a persistent client adapter.

The existing offline helper can be reused without weakening receipt
checks. A candidate *remote-surface* improvement is to combine a
successful previous receipt append with a separate durable next-action
claim in one bounded guest Exec, return the exact next request in that
same original output, and standardize a small caller adapter for
pagination, digest/identity checks, original receipt import, exact call
counting and explicit stop/resume. Keep manual fallback. Not yet a
selected proposal or reusable implementation. A true local client with
authorized tool invocation and durable storage may avoid more round
trips, but must be separately verified; do not assume the same adapter
works on this surface.

### Discovery probe

Separate fresh worktree /root/pr193-v4c-research, exact f8d2462,
operation ec7e2e22d4545ca99bde4840952c2600 exit 0. Five ordinary
operations queries (transfer a file through MCP; copy a source file into
owned Box; upload a large source file in chunks with SHA256; bounded
guest file transfer MCP chunks exclusive create interruption resume
verification; resume interrupted file transfer) surfaced tool.transfer-plan.
Three control queries (inspect guest file status; diagnose peer network
timeout; read-only guidance) did not. Entry d1b418ec1ab6b42c51cf5d71ace334fc
also surfaced planner for matching harness and operations queries.
This does not disprove B's reported different query/ranking failure;
exact B query is not established. Context ranks lexical overlap, caps
recommendations at six plus profile entry. Improve measured missed
relevant cases without hard-coded L014/tool preference or irrelevant
mandatory discovery. Do not claim a proven routing defect for every query.

### Actual bounded remote composition probe

Owned existing RC14 Box 9d374e18-196a-493d-bf1f-2d582589f04d.
New isolated fixture, not ordinary-task witness. 40960 deterministic
bytes, transfer f5b329c94c283d94dea52bf6355dab20,
source /tmp/pr193-v4c-probe-source.bin, target
/tmp/pr193-v4c-probe-target.bin, canonical bundle
/tmp/pr193-v4c-probe-bundle. No modification to prior trial artifacts.
Use original planner functions via short explicit Python argv;
no tracked code edits or new API. No direct source-to-target copy.

| Test/action | Actual evidence |
| --- | --- |
| Prepare maximum chunk request | ec7e2e22d4545ca99bde4840952c2600 exit 0; request 54892 bytes |
| Original stdout request bytes vs complete MCP file read | a1dc8a0302387f41803a265cf77a15a exit 0; stdout 8e742f934fbb5b4933cd2f4fd96bdcf6; file read ca8746af65666e45feb35798fdc02082; exact bytes, no truncation |
| Durable part begin | 534836bc8dd34062b2b57b90d05c9e4f exit 0, rev2 |
| Actual part transfer, independent readback | 1301213cb7b67e22c50221205a74b2a8 SUCCEEDED; 17d7f14be04e054b27b361686f691fbe matches source d54df5c52d7c3b52b21b6a1d62716143 |
| Original full-byte part receipt file | 8d5aed8fad9e60299b8500da7c53cfc9 SUCCEEDED |
| Combined record previous + begin next + emit exact request | da65d6f21b5a64ae4a4b7505e3c64ac7 exit 0; rev3 part verified and rev4 manifest claimed; stdout 109271215e14729fe407db8680cc1527; extra research comparison 0b215ab6d5782326bf5023bf40945e18 matches emitted bytes/digest |
| Manifest create/readback and receipt | ca0e3ac286bfb14dd829ec9d99c9e38b SUCCEEDED, c74bb6d21f52c8df897e4e533fbb8dae; f3a4e4fd9b7bb8b8911d5a90a4a2c110 SUCCEEDED |
| Combined manifest record + assembly begin | 210cdabba966ddbae5a220e53773b77c exit 0; rev5 then rev6, assess RECONCILE, duplicate begin refused |
| Assembly and actual final byte compare | 7dc7ef740f1ffdd83011636eedb12563 exit 0; final read ead736d298d5b81bd8d3bdc85341f0f4, source 7b8686cd168b9805a909ddc0383a457a identical |
| Final receipt and assess | 226e25bc1dc54cbbeb22f2aecad5f4f0 SUCCEEDED; 679e57eb1910e97a2d043f8dbe574aed exit 0, rev7 VERIFIED_READBACK and cmp/sha256 |

Full source/target SHA256
cb2ab7ec5cd0405b3dcaf55c74d70b2158a136168dd6e27c2ce09f6f1d7e92ca.
All original receipts retained, no native replay. Two combined calls
saved one separate Exec per transition in this controlled structure;
each Exec normally also needs poll and both stream reads. This is a
structural savings opportunity, not an observed matched total cost
reduction: the research added extra comparison calls. Raw request
output at 54892 bytes worked; nested base64 envelope for maximum chunk
may exceed 64KiB output page and was NOT tested. Implement exact
canonical bytes and bounded output paging/digest verification rather
than assuming one page. The manifest envelope was verified against a
separate file read, not yet a fully independent surface hash adapter.
Three manually constructed receipt files remain. The actual caller
adapter, full crash-window regressions, no-clobber retest, matched
efficiency, and fresh independent witness are pending.

## Proposed V4c work sequence (not selected yet)

1. Independent terminal review of original paired evidence and actual
   rev17/L014 rev26, record separate reopen recommendation with exact
   evidence; originating recorder then runs existing resolve-episode
   --outcome reopen at expected revisions. Do not use benefit-confirmed.
2. Use normal opportunity review/selection on exact clean child source;
   choose bounded next scope based on real residual friction. Preserve
   existing planner/Skill safety, original records and attempt budget.
3. Develop and verify a bounded receipt-to-next-intent composition,
   standardized client orchestration and pagination/call-budget
   handling. No implicit next dispatch. Each receipt and next claim
   remains a separate append-only revision; interruption after receipt
   before next claim may be READY, after next claim is RECONCILE.
   Stale output is not replacement permission. Fail closed on missing
   or uncertain original Operation, partial/truncated output, conflicting
   keys, wrong path/box/digest, existing target or concurrent claims.
   No reduction of byte verification, original exit checks or fsync.
   Keep exact source authority and caller evidence; no bypass.
4. Reproduce a specific missed relevant discovery query if available;
   improve actual asset relevance metadata or bounded routing with
   negative controls. No mandatory extra discovery for irrelevant tasks.
5. Test normal, interruption and collision with the same chunk size,
   inputs, authority, comparable surface and full safety oracle. Count
   actual comparable manual constructions separately and all logical
   MCP/other calls, including failures, discovery, evidence and handoff.
   Aim to reduce both manual overhead and total calls vs V4 B, assess
   against A and report any safety/cost tradeoff; a smaller regression
   is not by itself benefit-confirmed. No positive result forced.
6. Independent review/delivery and held-out ordinary task witness,
   preferably with no pre-registration exposure, then terminal
   disposition. Only then consider V5; parent gates unchanged.

Current actual blocker: independent terminal reopen review is not
available on the originating surface. The owner may ask the existing
independent V3c reviewer (not Lane A/B or originating implementer) to
review the recorded result and recommend reopen or raise findings.
No automatic assignment, scheduler, merge or publication.

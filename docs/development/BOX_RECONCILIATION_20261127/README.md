# Box development preservation audit — 2026-11-27

Owner-authorized preservation of P0/P1/P2 and BDH development on Draft
PR #191. Baseline 720f3ba9d1b5ad24d64858e880794049dcbcb2ce, tree
8b54a9c636c6b3cc08a0f35cb6ebbed11976b948. No merge, publish, force,
reset, native fixture mutation or gate waiver.

## Observed inventory

Eleven owned Running Boxes (list c45b7e356817827db4e9e59d0230e629).
Bounded source discovery in /root, /tmp and normal workspace roots:
72 Git checkouts: 64 on main Box 9d374e18-196a-493d-bf1f-2d582589f04d
(22 clean, 42 dirty), eight in secondary Boxes, two Boxes without Git.
Main inventory 87f37bc7e42e84a3e94e9320e4cd5fd8 and content audit
e5a11e3bf6125291c7da0d43d7df4b0f: 248 modified/untracked occurrences
equal current content, 222 exact historical remote blobs, 95 generated,
cache, deleted or excluded, 53 raw transfer evidence, nine distinct
nonmatching working-file blobs. Compared #191, v1, #193 and three
published #193 checkpoint refs. Separate index audit
4cf0dfe4a772b0f5b8d7e96c7836a786: 195 historical/current blobs,
four audited working bytes, two staged-only obsolete failure-index
blobs superseded by tested run_batch implementation. Secondary dirty
clean-startup lab: 43 surviving files equal current, four historical,
16 deletions already delivered; no novel surviving source
(78ebc170daa5764d3908273f99bb5c11). Other seven checkouts clean.
See [secondary-box evidence](secondary-box-evidence.json).

## Recovered work

- D37 local-only d2338b3: add missing pure Scoped regrant UNKNOWN
  recovery test; retain recovery-only proof, forged digest refusal,
  no redispatch and sibling history. No runtime edit.
- BDH local-only 60448ba: add clean linear committed-range helper and
  four regression methods, retain newer indexed framing, append
  optimization and ordinary staged mode. Test combined framing.
- Original 85784330fc2b04e89f256cc9cbaad8c0 exit0: 13 staged-batch
  tests; three pure history tests and nine subcases; vet/format PASS.
  Final affected package and harness regression separate.
- Earlier fast-local policy a3a01ea represented in 363ee54. Earlier
  alternate-pool workflow/script superseded by verified Scoped lane;
  retain historical note, not old active workflow. P1 replan four
  divergent docs' substantive added lines already in current docs.
  Old index-only failure helper superseded. PR188 H4B/H5 docs byte
  identical. Never apply old patches wholesale.

## Historical archive (not active source or acceptance)

Five commit-delta bundles: 14 local-only commits, 538151 original
patch bytes, commit/parent/tree and SHA256, source text preserved.
Three P1 phase-history bundles: 28 original revisions and exact text
digests. P0/P1 handoff bundle, source-verification bundle (28 records),
historical altpool notes, P0 inventory generator, historical WAN
witness and PR193 lane B report. Main loose-evidence index: 2174
named file locators, sizes and SHA256; an index, not raw byte backup.
Old source SHA IDs are not claimed to be new remote Git commit IDs.

Do not upload credentials, tokens, runtime ledgers, private archives,
raw captured requests, unreviewed logs, toolchains, caches or built
binaries. Selected archive text checked for high-confidence credential
patterns, not a blanket secret-scanner guarantee. Raw/scratch material
remains in original Boxes; bounded search cannot prove absence outside
scanned roots or in unregistered/deleted Boxes. Keep all labs intact.

P2 scoped signoff 09152ff: N2 29/29, S2 72 markers/23 receipts,
S25 and Forward PASS. See PR comments 7228099703 and 7228231141.
H/C04/C01/final C11 and actual RC proof remain open. P3 R0 not started.

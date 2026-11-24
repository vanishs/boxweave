# PR188 H1A — typed read-only maintenance observation

## Scope and outcome (2026-11-24)

Box acceptance PASS; independent exact-head CI required. H1A is observation only: no approval digest, all-blocker coordinator, mutation fence, uninstall/purge permission, or recovery apply. H1B owns planning decisions. Normal Doctor and MCP contracts are unchanged. Historical artifact directory counts and metadata are not proof of archive ownership or authorization to delete.

Source workspace: H0 lab A, topic parent `6591015bcea937ac45b4628f42f252dcc988d82b`. Candidate `1.0.0-dev.h1a` built with uncommitted H1A source on that parent; embedded commit identifies the parent, not a released candidate. Final tested binary SHA256 `e3f40917f484b0fe6a2425f1d796b8f554e61719d49c413f6676e5a65f8e2e18`. This delivery uses the exact Box source bytes and CI harness. Clean VM CI rebuilds from its exact HEAD.

## Implemented boundary

- Root-only `doctor -maintenance -project user-UID`, fixed installed scope, approval/override flags refused. Supported immutable RC13/RC14 exact hashes and current matching observer/installed identity; other identities remain unverified.
- Bounded typed read-only ledger capture: exact create lineage and incarnation, owned/retired counts, original UNKNOWN and nonterminal records, historical recorded claims separately from nonterminal claims and claim protection, peers, forwards, foreign counts and orphan counts. Redact endpoint and credentials; no source database write.
- Observe fixed installed paths, metadata, units/effective drop-ins, root journal/preimage, root policy and bounded directory evidence. Metadata or entry names alone never establish ownership.
- Read independent native project, profile, bridge, ACL consumer and proxy evidence, exact UUID/create markers, pending operation witness and unrelated project count. Double-read with canonical unordered UsedBy references, reject changing source witnesses; no daemon activation if Incus inactive. No cross-source atomic guarantee.
- Reuse root-policy and native Forward predicates. Scoped root/ledger comparison; partial coverage stays explicit, not a healthy maintenance verdict.

## Actual Box evidence

| Witness | Operation | Guest result |
|---|---|---|
| RC13 offline repeated observation, installed binary/ledger/manifest/journal/unit hashes unchanged, scope and approval refusal | `cae52aee34b045040df0a4ebf22f6e70` | exit 0 |
| Final candidate build and repeated H1A Go tests | `8e7257eadb4911312802afbea6bab735` | exit 0 |
| Final RC13 candidate observation | `61f15e105310e4604a8bd533258cc13f` | exit 0 |
| Final RC14 live three repeated captures, same native digest | `007f36c25fb43d8b37186f82df858bbe` | exit 0 |
| CI Python harness on actual RC13 offline fixture | `830802a1e4e9050275e0cf47f3f22733` | exit 0, PR188_H1A_OFFLINE_EMPTY=PASS |
| Same harness on actual RC14 live fixture | `e418d4af349006781841c4777643654a` | exit 0, PR188_H1A_LIVE=PASS |
| Unrelated empty Project counted but not named/modified; nonroot refusal; HTTP 200 | `8eea8f52ccef936297c8d10b96123b0b` | exit 0 |
| Six Go package targeted regressions (H1A/PR177/PR178/PR182/Forward/Diagnostic), Python compilation, shell syntax, diff check | `1a59492d5812f702b52fa4cac27629a6` | exit 0 |

A: staged original RC13, Controller/helper inactive, exact legacy preimage and effective inherited LXC drop-in observed; no activation, uninstall or binary replacement. B: original RC14, 4 owned/4 retired, 3 confirmed directions, 1 Forward, Root sequence 5 agrees with ledger, 78 historical recorded claims, 0 nonterminal recorded claims. All 4 native UUIDs/create markers matched, no unowned native instances, Forward references observed; HTTP 200 and all three services active. Native canonical digest `42c289e21ef2bd542e91a16a7d55cd73941c52bbda9b7c3314f41726035c5c6d` repeated three times. Harness independently compares project/profile/instance config, PID and installed file hashes before/after.

Corrections: fixture assertions initially confused terminal recorded claims with live protection; corrected model and tests. Actual RC14 run `af9d94f8c47d0071f0bcc78f53346f75` correctly returned partial/exit 1 on differing native digest; independent read `58910d94f059cbece03f66b79ba1d8e1` showed Incus UsedBy ordering only. Sort unordered reference lists, preserve ordered configuration and effect witnesses; repeated actual observation passed. Do not weaken drift detection.

Temporary lab-only B-to-A Peer grant `1a8eb0859d8ffb996e4f75e64425c8b9` transferred only candidate, verified SHA, temporary bounded server stopped `ce8a167ba7d06f299616b7acac494207`, grant revoked `a6e71097627ad9bdd06d795123ef8d76` SUCCEEDED. Both original fixtures retained. Temporary unrelated empty Project removed after unchanged-state test. No physical Host configuration edits; the ordinary outer grant/revoke adds history.

## Independent CI and handoff

Existing clean Ubuntu VM N2 matrix now asserts H1A active-empty and offline-empty in all three lanes; scoped lane also asserts live observation. It builds the exact checked-out HEAD, stops/restarts only its disposable Controller for offline test, and retains existing N2/N3/N4/D4/retirement gates. Actual Box runs used historical pinned adapters, not the new installed exact-head adapter, which CI will validate.

Unit tests exercise absent/unsafe artifact metadata, symlink/hardlink and parent link, immutable adapter match/refusal, preimage classification, retired rows, protected/released UNKNOWN, foreign redaction and credential absence. Not exhaustive H1 acceptance: mounted/unreadable mixed artifacts, complete archive provenance, pending-create ambiguity, parallel writer/foreign consumer effect tests and full blocker/approval/retirement contract remain H1B/H2 and independent CI. No physical legacy activation or destructive retirement performed due known inherited unit drop-in. No new CI result claimed until verified. Keep PR draft; next H1B after H1A CI review.

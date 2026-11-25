# bwdev v0.1 Box development and handoff evidence

PR #189, work_ref boxweave/pr189-bwdev. Starting branch head
acf7017a5716a5011c8a04cafad9758ce3ec39c7; v1 base
a38ef26e047f64bb09490be35d9aa39048b320ac.

## Environment and scope

Box 9d374e18-196a-493d-bf1f-2d582589f04d, retained authorized
lab, isolated clone /tmp/bdh-pr189-v01. New dedicated Box admission
failed SAFE_NATIVE_PREFLIGHT_DENIED (dab2374c02b77d1d2e75deab58581a05);
no bypass or replacement of the existing #188 worktree.
Clone operation a3859c60a98595358707d0d2ca517d3f.
Python 3.12.3 and Git 2.43.0 available. Make and Go were not on PATH;
no dependency installation or product runtime changes attempted.

## Evidence

| Observation | Original Operation / result |
| --- | --- |
| Initial CLI unit tests, including clean/dirty checkpoint, exclusive capture and original-operation review | e7a3ffa35193279c7378dbaf7f8e5894: 7 tests, exit 0 |
| Make unavailable | 5aaed2e8a6e98b6089779ebf1366bd9e: exit 127 |
| Direct Python tests and actual doctor negative prerequisite checks | 46197b71acc1db5a5e3b50b550ddcc49: 7 CLI + 6 fixture tests, exit 0; required missing Make/base correctly blocked |
| New local-guide relation regression found missing Path import | 99b26ebbb02ebe59a1e65201c885eecd: exit 1; corrected |
| Final focused tests, module-linked skills, exact v1 ref fetch and doctor | 710d903b49afc371bd41930710f88f14: 8 CLI + 6 fixture tests, exit 0; v1 ref observed |
| Actual task checkpoint captured outside worktree | 10d689b154458876f979379b4ee6c099: exit 0; /tmp/bdh-pr189-v01-handoff.json |
| New process and review attempt read same checkpoint | 14324a73d2cf2e8cd42f845e0e3c7e4c: expected REVIEW_REQUIRED for dirty tree, original Operation and Box identity; wrapper exit 0 |

The separate-process fixture also proves that a second capture cannot
overwrite the first, that a dirty tree does not silently pass, and that
a retained original operation triggers review. This is a simulated
handoff, not a proof of autonomous Agent takeover, cross-writer atomic
coordination, or an independent authenticated reviewer. The actual
checkpoint records the source head before commit; its dirty observation
does not preserve uncommitted bytes. The submitted Git tree preserves
the tested source changes. Reconcile the current PR head and CI before
continuing. Do not repeat original effects based on a receipt alone.

## Prior CI issue

Original run 45547041460, Legacy job 138979037296, failed on
ConnectionRefusedError at the MCP handshake after a fixture-induced
controller restart. Added a bounded loopback listener check and
revalidated controller PID, original database state and journal before
the unchanged authenticated check. Offline tests cover refused-then-ready
and timeout. Timing race is a hypothesis, not proven native root cause;
the new independent installed CI must confirm. No native test weakened.

## Learning and next action

Learning candidates BDH-L004 through L007 are captured in
[Learning Inbox](../../dev/LEARNING_INBOX.md). The new capability follows
[Script First design](BWDEV_DESIGN.md) and uses the existing
[BDH entry](../../dev/README.md). Full Go, full installed lifecycle and
independent candidate acceptance are left to applicable CI. No product
runtime change or productivity improvement is asserted. Retain the
isolated Box clone; no deletion authorized. Review exact submitted SHA,
CI and remaining gates before merge; use v0.1 on the next normal task.

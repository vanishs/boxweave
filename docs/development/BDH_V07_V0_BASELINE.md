# BDH v0.7 V0  Box baseline and defect reproduction

Status: local baseline and gap reproduction complete; CI and later phases
are separate. This is an explicit harness development investigation, not
an independent ordinary-task witness or an implementation of a new asset.

## Exact environment and baseline

- Canonical v1 9d07665dcfc5ae6345d6bfcd0d71de7b6510b51d.
- Paused parent #191 3494ae8a35c50c386686c9d9d675c1ae37e1b1b7.
  Original five PR-triggered runs 45588862102, 45588862517, 45588862206,
  45588862382, 45588862396 reported success. No claim that native skipped
  gates or P1b ACK/WAL are closed.
- Child #193 planning head 4e75b78b8451574d1faeabb17673c5e03021b5ee,
  tree 3bb402f0f0f7e96abf3c1511e0d164ac559188b0. Child PR-triggered runs
  and combined statuses returned empty at inspection; push checks may not
  be represented by that API. Do not equate empty with PASS or no CI.
- Owned Box 9d374e18-196a-493d-bf1f-2d582589f04d, RC14 controller
  3fb71a6ddf47d13d165e5c039144aa66fad0eb05; Git 2.43.0, Python 3.12.3.
  Original parent checkout /root/pr191-consolidation inspected clean,
  separate clone /root/pr193-bdh-v07 at exact detached child HEAD.
  Inspect ea972ea529af116e360afa0ff47066bc and clone
  3e654547c120697cb5486df55bc13373 both guest exit 0; no parent edit.
- Original baseline 7f6213b85c1fdaee10eefaf11829c04b guest exit 0:
  work plan, learn next, learn audit (no findings), Eval and 73 tests.
  Summary ccc405decef6bb45d8142a17f6ebe3b0 exit 0; 14 Eval cases PASS.
  Concurrent source-read Exec was explicitly refused while original
  baseline claim locked; original was polled, not replayed.
- Histories L012 rev3, L013 rev2, L014 rev11 validated in
  e164c0355522f7cecd6fd040c71805e1 exit 0. Read original stdout receipts
  80aa43b9f95b46f394c67fcf4d77eac4, a613d026dc4e98e64cfb1d37720adb2e,
  2d88e9945cde2ac0e16de275a1a6e52a, 27487bec36beb5c71213f07adb730edb
  (all observed exit 0; some bounded output pages). See prior research.

## F1 and F4

Actual old plan and learn next both surface L014 rev11, with
next_action "review linked implementation", despite seven distinct
recorded observations (not seven independent Agents). Old routing
already finds the asset. Copied, explicitly synthetic state/observation
variants were evaluated without changing live history:

| State | Old selector with additional helped + residual toil |
| --- | --- |
| CANDIDATE | review linked implementation |
| DEFERRED | review linked implementation, even with new-observation |
| STALE | review linked implementation |
| VERIFIED | excluded |
| ACTIVE | excluded |

Probe 6cefd30dae7ed43749d402929e45c28e printed those results but exited 1
on an incorrect plan JSON key. Corrected probe
b8ac84ee5a6f70507f129b8d9f949846 exited 0 with identical selection results.
Historical P1b protocol-1 feedback review at exact child checkout
069801d2a4344d19abc0a45d6ab7127e exit 0 returned FEEDBACK_RECORDED
with no issues and exact rev11 match, without an opportunity decision.
This is F4, not evidence that P1b lacked useful feedback.

## Actual manual transfer baseline

Before executing assembly, the comparison oracle was exact byte equality,
whole digest, refusal of missing/wrong-order parts with no output, and
refusal of existing final path without changing bytes. Primary future
metric: manual coordination/program construction actions; count logical
MCP calls separately, including admission, polling and retries. No
elapsed-time or token estimate; one controlled baseline trial only.

Two distinct deterministic 40960-byte ASCII byte sequences were
prepared externally: byte i is 32 + ((i + offset) mod 90), offsets
0 and 17 respectively. This is actual RC14 transfer, not production
source or an SFTP interruption. Retained paths in the exact owned Box:

| Part | Create Operation (SUCCEEDED) | Whole file SHA256 |
| --- | --- | --- |
| /tmp/pr193-v0-transfer-a.bin | 7da5282027a13219934f80f3eb91e7fc | 4c4bddc83ce13b83a298f26d68648c1c4ccc98f875e124202756a884352d5646 |
| /tmp/pr193-v0-transfer-b.bin | cc1ef4255df09131dcc93a037e948892 | fb2bb99e6c63ced8e84a4a1db7b4effc1a893d2d820de91e856532d056630930 |

Separate actual whole-file reads d74ea906f53deff08ce24bc3547e901e
and 2138f22accb65f36d9b1b4ea68499139 matched the original base64 bytes
exactly, not just reported size. A bespoke short Python assembly program
compared each chunk with expected bytes, then opened final path with
exclusive xb, verified assembled bytes, and exercised negative controls.
Original 9496e3878b0d140f63fd4f2c2456bf2b guest exit 0, stderr empty:
missing path refused (FileNotFoundError), wrong order refused (ValueError),
81920-byte final output at /tmp/pr193-v0-transfer-final.bin,
SHA256 24d11909c0680cbf0db506ffbfd26ec838e4bd16ee39b6afd2085415b4d48874,
existing destination refused, bytes unchanged.

Observed orchestration: two separately constructed chunk payloads,
two exclusive creates, two terminal operation reads, two explicit
readbacks, one bespoke assembly program and one Exec, one terminal
operation read and two stream reads. Ten MCP calls including polling
and both output streams (2+2+2+1+1+2), no refused admission on this
transfer. Negative controls executed inside that one Exec. A new
planner/helper should reduce hand-authored coordination while keeping
the same integrity/refusal oracle; raw transport count need not fall.
The numbers are a single controlled investigator baseline, not an
independent Agent comparison or quantified time savings. Actual SFTP
mid-write interruption, transport/daemon crash and power loss untested.
No files or Boxes deleted; retain fixtures and do not blindly rerun
exclusive creates.

## Learning and continuation

Actual baseline observation appended through learn observe at expected
rev11 (operation 7af2ea3ae3645c3bfc3a91009d77c09a exit 0); [L014 rev12](../../dev/learning/BDH-L014/00000012.json) remains CANDIDATE. Historical
implementation preserved; no new tool or promotion. [Protocol-1 work
history](BDH_V07_V0_WORK_HISTORY/00000002.json) records material use
and handoff. V1 should use explicit typed residual evidence and
review-readiness (legacy unknown, no inferred friction from helped);
V2 owns durable episode selection/assignment and independent review
continuity. V3 must select from actual evidence rather than hard-code
L014; V4 requires a separate ordinary-task Agent and matched comparison.
No authority to merge, release, change #191 or alter installed runtime.

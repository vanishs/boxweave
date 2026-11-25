# BDH U1-U4 local acceptance and rollout handoff

PR #189, work_ref boxweave/pr189-dual-loop. Started from plan commit
df864c27ea08211a63d663e5eeecb167c1ab3ebf; exact base
a38ef26e047f64bb09490be35d9aa39048b320ac. Reused owned isolated checkout
/tmp/bdh-pr189-v01 in Box 9d374e18-196a-493d-bf1f-2d582589f04d.
Resume 9cb6b87f074c1ea4e277c866881e4579 verified clean source, Python and Git.
No Go, gofmt or Make locally. Prior v0.5 and plan-head four CI runs passed;
plan runs 45556615187, 45556614993, 45556614989, 45556615070.
No product runtime, host, other worktree or MCP catalog change.

| Phase | Original terminal evidence | Local result |
| --- | --- | --- |
| U1 | c9956e610e9d02a1300d19285ecf8d7f | 45 regressions, 4 Eval, portable plan/check, exit 0 |
| U2 | 5e6c5795df369283951dd1ef266a54ff | 47 regressions, separate process boundary/export and stale revision, exit 0 |
| U3 | aebfd523e4b862b1ea0b785dc59050bd | 51 regressions, capture/defer L014 revision 2, exit 0 |
| U4 | d8e6361ceba51dde865195afb52aa34b | Related deferred item rediscovered, new real observation linked, exit 0 |
| U4 | 63853c10651a7170dc6bed421be8653c | 54 regressions, 8 Eval, L014 implementation attached, learning audit clear, exit 0 |
| U4 | d824ecb283a7d27b2bcf549f27b35563 | RC13-RC14 guidance fixture from non-Git cwd; handoff and pending persistence, exit 0 |

Actual learning: historical L005 large argv refusal and this task's U2
refusals were linked, not counted as separate candidate assets. Verified
exclusive file create f9e86facbe28dff3fba4aed216c6b4a5 and execution
65bda5634d3a24637d5be3d92eba49a5 succeeded. L014 was deferred,
rediscovered via domain/signal in a later process, reopened and attached
to the bounded source-transfer Skill improvement. Five immutable revisions
preserve the evidence; candidate is not automatically verified/active.
Same Agent, distinct recorded work refs across earlier v0.1 and current U2;
not proof of independent Agent recurrence. No claimed numeric productivity gain.

Scenario coverage: portable operational guidance (fixture, no live host);
normal development (this change with prior CI and exact staged tree);
known failure (actual admission refusal, original receipts preserved);
deferred rediscovery (L014); interruption/continuity (separate process and
expected revision tests); tiny guidance (no unrelated automatic asset path
match); invalid/mismatched scope, sticky UNKNOWN, no-checkout unavailable
persistence and untrusted content (regression/behavior cases).
The remote fixture history and handoff are retained outside the checkout
at /tmp/bdh-u4-remote-fixture; not a product success or a real learning
observation. Existing development finish/CI and authority gates are retained.

Delivery choice: stage-by-stage local Box verification then a single
integrated CI candidate to avoid repeated full native CI runs. Previous
CI checked during implementation. v0.5.1 is a dev-only increment, not
a product release. All original observations and old histories remain.
Remaining rollout gates: exact-head independent CI and L4 review, actual
fresh Agent activation on each supported surface. GitHub-only manual entry
and deployable configuration are available, but no fresh independent Agent
session or automatic project configuration was performed. Do not claim
automatic activation on an unconfigured surface or a completed actual
Host migration. No merge/release without applicable authority.

Final offline gate: original operation 6879185fee08abab8fc1bab9c1cded33 exit 0, 54 bwdev and 109 existing Python tests (163 total), eight deterministic Eval cases, clean learning audit and diff check. Both original output streams read through terminal page. No claim of native or independent CI from these results.

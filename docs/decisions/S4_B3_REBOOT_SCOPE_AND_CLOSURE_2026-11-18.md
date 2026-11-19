# S4-B3 final acceptance scope — service-level outage in place of host power reboot

**Owner-approved on 2026-11-18 PT.** This is a signed-scope *decision record*, not a claim of test completion, production support or physical reboot.

## Acceptance boundary

- The original [six-stage B3 criterion](../development/S4_B_SIX_STAGE_PLAN_2026-11-16.md#b3--directional-private-peer-box_peer_set) demands restricted ownership, verified one-way allow/revoke, conservative unknown/failure handling and independent native/kernel readback; it does **not** make physical power cycling a unique B3-only gate.
- For **B3 only**, the owner approved replacing a **physical whole-host reboot witness** with a **real native Incus service restart and installed PRS/Root service restart**, preserving the same on-disk SQLite + original Root journal and independent native ACL/NIC/nft/packet readback. This is a *service-level outage/recovery surrogate*, **not** an actual kernel/host reboot, power-loss, filesystem journal crash-consistency or S5 proof.
- Retain a real persistent-host reboot/power-cycle and incomplete-original post-reboot provenance gate for **S4-B6/S5** before promoting the stronger whole-host durability claim. A failed mandatory B3 ownership, ACL, original no-replay, NIC isolation or ETag race cannot be waived by this decision.
- B3 requires a **precisely synchronized external native ACL contender** in the Root stop-barrier GET/ETag→PUT window. A competing metadata write must not be overwritten, a stale approved ETag must be refused, no grant confirmed, the journal/claims remain conservative, and protected NICs stay stopped.
- Reuse previously accepted **14 scoped B3 PASS receipts** and the original Incus 7.6 **9/9**; do not infer extra coverage from them, repeat them unnecessarily or modify the L0 shared host.
- Require separate final source/diff and original GitHub job-log/evidence audit after these focused jobs. If any test fails, fix and rerun only affected gates; sign formal `S4_B3_PASS` only after independent native readback and all mandatory reduced-scope B3 checks succeed.

**Execution:** sole Draft [PR #153](https://github.com/vanishs/boxweave/pull/153), `feat/v1-s4-runtime-integration-20261116`; no merge, second PR, RC or publication. All tests use disposable, independently installed GitHub Incus 7.6 hosts, not shared L0 host policy.

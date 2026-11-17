# S4-B2 Stage 1: kernel same-IP tuple ambiguity (2026-11-17 PT)

Status: Stage 1 evidence pending independent native confirmation. B2 PARTIAL; B3 not started.
Baseline source: e201f1a785a1e02839ade389b6c5ca32a846cc85.

## Why this one experiment is necessary

The latest successful native Incus 7.6 runner proved two non-target
A/C entries survived A/B exact-IP deletion. It did NOT exercise another
owner or network with the SAME source and destination IPs as A/B.
The production conntrack delete syntax is `conntrack -D -s A -d B`,
with no per-Project zone, peer UUID, interface or original 5-tuple.
BoxWeave can prove IP lease ownership for its own NIC, but a conntrack
record does not carry a BoxWeave Project identity.

## Isolated native experiment

In the existing temporary GitHub Actions root Incus 7.6 three-Box
fixture, AFTER all A/B/C stop, ACL DROP, original protected SQLite
root journal and native no-replay checks, create exactly two synthetic
TCP IPv4 kernel entries with the SAME verified A/B source and destination
addresses and distinct ports 47301 and 47302. Demand that native
conntrack -L reads back BOTH ports before a deliberately test-only
`conntrack -D -s A -d B`. Demand that both are absent afterwards.
The test-only probe is a separate incident reproducer, not another
authorized production journal operation. Cleanup is limited to the
two fixed-port tuples and the runner is disposed. No shared L0,
production ACL/Scriptlet, other Boxes or host-global flush.

Mandatory native marker:
S4_B2_NATIVE_ROOT_IP_ONLY_DELETE_COLLATERAL_SAME_IP_DISTINCT_PORTS=RISK_CONFIRMED

Interpretation if confirmed: root IP-pair deletion is not guaranteed
to preserve unrelated flows merely because the queried BoxWeave
bridge and ACL are owned. This experiment does NOT prove a real
foreign Project with overlapping IPs exists, nor does it prove that
all foreign Project flows would be lost. It proves that IP-only tuples
cannot be distinguished by owner on this kernel conntrack path.
Phase 2 must add a stronger verified scope boundary (such as safe
non-overlapping global native subnet admissions or an independently
proved per-bridge conntrack zone) and negative tests before enabling
installed peer revoke.

Existing installed S4-A PASS and separate Scriptlet native PASS are
not installed B2 acceptance. B2 remains PARTIAL until complete
controller-root integration, application live flow revocation and
cross-Project negative proof are obtained.

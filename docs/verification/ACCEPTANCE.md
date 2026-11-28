# Current acceptance and evidence map

An admission, terminal operation, actual guest exit, native readback, packet proof, physical Host and publication are different evidence tiers. Test exact candidates; preserve uncertain and failed originals.

| Scope | Required verification |
| --- | --- |
| Source/distribution | Format, quality/race/source checks, real amd64/arm64 CGO builds, immutable version/assets. |
| Lifecycle/files | Authenticated UUID/Project confinement, native create/start/stop/delete/update, readiness, exclusive file create/list/read and unsafe-write refusals. |
| Durable operations | Idempotency conflict/replay, separate original/native IDs, disconnect, exit/output retention, UNKNOWN recovery, claim protection and no duplicate dispatch. |
| Identity | Token/OAuth revocation/refresh, Actor/Layer boundaries, foreign denial, restricted socket, exact Root original checks. |
| Network | Schema 7 Scoped v2 only, independent UUID-derived ACL per Box, Legacy/shared-ACL/mixed refusal without effects, default deny, directional IPv4/IPv6, unrelated/reverse flows, Forward ownership/exposure, actual application packets, reboot and partial-policy fences. |
| Maintenance | Read-only preview, approval, drift refusal, full Scoped retirement/resume (normal and separately reviewed offline), completed-data purge, fresh install and old/mixed refusal with foreign native preservation. |
| Nested/physical | Conditional native compatibility and independent controller; distinct target-specific physical/external/storage tests as required. |

For the current Scoped-only v1 candidate, the [consolidation acceptance ledger](../development/CONSOLIDATION_ACCEPTANCE.md) and [workflow dispositions](../development/P4_WORKFLOW_DISPOSITION.md) govern acceptance. All 29 N2 lanes, independent S25, 72 S2 marker rows/23 producer receipts and source/quality/dual-architecture gates remain required. Historical Legacy positive upgrade/rollback evidence is version-bound, not an active product path; stable upgrade/rollback and #170 obligations require explicit final disposition, not waiver by documentation. H and C04 remain open.

Exact merged feature evidence is in [status](../STATUS.md). #184 passed full legacy retirement, purge, fresh Scoped ledger, and scoped retirement/purge preserving unrelated resources. #177/#178 proved original recovery, live directional flows and partial-native fault acceptance. No physical signoff follows from this.

The complete original A-/BH-/TC- case matrices, bounded owner substitutions, signoffs and failures remain in the [history archive](../history/README.md) and [pre-refresh acceptance matrix](https://github.com/vanishs/boxweave/blob/06d27434b11997e9254822c122483c9dacba12a3/docs/verification/ACCEPTANCE.md). Reconcile all mandatory IDs against the current exact candidate before stable approval. See [continuity](CONTINUITY_ACCEPTANCE.md), [risks](OPEN_RISKS.md), [versioning](../process/VERSIONING.md).

## #188 H5 integrated candidate

[H5 delivery and exact-source evidence](../development/PR188_H5_DELIVERY.md)
records H1-H4 and H5 Box results, four independent installed lifecycle
lanes and remaining proof boundaries. Quality CI exercises the complete
offline Python suite, full Go race regression and bounded capacity
fixtures. Candidate artifacts stamp the exact checked-out source on both
architectures. An earlier successful run is evidence for its own source
only; the final candidate must pass its independent CI again. No merge,
physical Host acceptance or release is implied.

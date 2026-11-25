# Native networking and directional Peer

BoxWeave manages exact owned Boxes in a restricted Incus Project. Private peer access is denied by default; internet access is independently controlled. No arbitrary IP/CIDR/port ACL or host firewall editor. See [security](SECURITY.md), [MCP contracts](MCP_TOOL_CONTRACTS.md) and [installation modes](../getting-started/MANUAL_INSTALL.md).

## Scoped Peer v2 (explicit fresh mode)

- Create uses stopped quarantine, exact UUID/incarnation and an independent UUID-derived ACL and static dual-stack binding before start. New Boxes default to boot.autostart=true; explicit false is preserved.
- A grant/revoke names two owned UUIDs and one direction. Controller stages the original and claims; bounded Root helpers independently verify ledger, ACL/ETag, addresses and kernel inventory. No arbitrary Root command.
- Grant prepares ACLs and directional conntrack cleanup under deny, then allows last. Revoke denies first, cleans that direction/ACLs and publishes verified policy before SQLite settlement. Reverse/unrelated flows and boot settings are preserved under verified conditions; no whole-Project STOP fallback.
- Root guard restores policy before Incus starts; partial originals fail closed. MCP read/recovery may remain accessible while mutations are fenced. Inspect originals; do not remove journals or replay UNKNOWN.
- Delete cleans only exact dependencies and verifies native absence/retired lineage; scoped addresses are held against reuse.

See [N5 delivery](../history/development/V1_PR178_N5_DELIVERY_2026-11-23.md), [N4 fault evidence](../history/development/V1_PR178_N4_INSTALLED_FAULT_ACCEPTANCE_2026-11-23.md) and [Doctor/Recover](../process/DOCTOR_RECOVER.md). No scoped migration, upgrade, rollback or backup commit. Full retirement requires original preimage and completed MCP cleanup; see [maintenance](../getting-started/MANUAL_INSTALL.md).

## Capacity

1,024 owned Boxes is a logical limit, not physical capacity. Retired identities are bounded at 8,192; held addresses can exhaust sooner. No safe automatic reuse or compaction. At most 32 retained directional grants, including REVOKED; same-incarnation regrant reuses its row. Root history is bounded at 8,192 with reserve for active revokes. Synthetic ledger tests are not 1,024 physical guests or 8,192 real transitions. The #188 candidate reserves a future retired identity record for each owned or unresolved Create before admitting another Scoped Create. This preserves ordinary Delete headroom; it does not release address holds. Doctor and host_status separately report live slots, IPv4 union upper bound, identity reserve, retained Peer directions and remaining grant-history upper bound with revocation reserve. LOW means at most 20% remaining. Address reuse and authenticated history checkpointing are separate [unapproved reclamation proposals](../development/PR188_H4B_RECLAMATION_DESIGN.md). Existing /24 bridges are not silently enlarged; reviewed fresh native bootstrap can provision a /21 under exclusive conditions.

## Legacy mode

Legacy uses a shared ACL and separately reviewed Root/PRS workflow, not the scoped guard. Peer may require Host-local approval and disrupt other consumers during older maintenance. Never apply legacy all-consumer/shared-ACL remedies to scoped installations. Historical IPv4 reuse, IPv6 NDP, anti-spoof, ETag and global Scriptlet experiments are [archived](../history/README.md), not current scoped failures.

## Native forwarding

An exact instance-owned Incus proxy device maps TCP/UDP Host bind to the guest port. Default loopback; external binds need explicit acknowledgment and approved service/native policy. New installer-created services default to all; existing units/manual serve keep observed policy. No managed DNS, TLS, Caddy, website routing or reverse proxy. Native device proof is not application listener, LAN or public IPv6 reachability. See [Forward policy](../history/development/V1_FORWARD_ALL_POLICY_2026-11-21.md), [usage](../getting-started/USE_BOXWEAVE.md) and [acceptance](../verification/ACCEPTANCE.md).

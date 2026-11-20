# S4-B2 Stage 1 closure: frozen findings and Stage 2 integration contract

Date: 2026-11-17 PT
PR: https://github.com/vanishs/boxweave/pull/153
Evidence baseline: e201f1a785a1e02839ade389b6c5ca32a846cc85
Stage 1 state: COMPLETED; IMPLEMENTATION_SCOPE_FROZEN, 2026-11-17 PT
B2 state: S4_B2_PARTIAL. B3 not started.

## Stage 1 purpose and boundary

The owner requires conntrack, live TCP/UDP revocation, old-client Scriptlet
compatibility, root/controller authorization and rollback all to remain in B2.
Stage 1 ends after the experimentally significant native behavior and the
installed topology/handoff boundaries are understood. Product implementation
is reserved for a single concentrated Stage 2 cycle. No new public network
MCP writer, release, new PR or shared L0 global configuration modification.

## Verified existing knowledge: no need to repeat

- Real Incus 7.6 A/B/C dedicated network, static IPv4/IPv6, controlled ACL
  grants, stopped/all-NIC barrier, stale ETag rejection, foreign raw NIC guard,
  lease/address collision denials, partial NIC PUT fault and restart fail-close:
  independent native Actions 44183845937 SUCCESS.
- Real Linux kernel A/B TCP and UDP application flows (both IPv4 and IPv6):
  ACL-only DROP left established traffic alive; separate pair-scoped
  conntrack deletion finally blocked all four flows. Same native run,
  markers S4_B2_NATIVE_LIVE_TRACKED_* and
  S4_B2_NATIVE_COMPILED_DROP_PLUS_EXACT_CONNTRACK_DELETE_TCP_UDP_DUALSTACK_DENIED.
- Four A/B TCP/UDP original kernel conntrack scopes were removed through
  the protected SQLite/host-root one-shot journal; two non-target A/C
  sibling entries survived (run 44183845937). It did NOT prove that
  an overlapping-IP foreign Project is immune.
- Restricted legacy TLS Project/Profile GET/PUT, real Files GET, real
  instance Exec and native terminal, cross-project 403, dedicated ACL
  GET/PUT and foreign ACL refusal all survived Scriptlet route and exact
  rollback. Root SDK original global config apply and restore passed
  independent Incus 7.6 run 44122934628.
- S4-A installed restricted HTTP MCP and original terminal regression,
  quality and controlled dual-arch builds pass on baseline source.
  They do NOT represent installed B2 acceptance.

## New Stage 1 installed topology and permission findings

Source-read (not production installation proof):
- internal/install/native_prereq_prepare.go nativePrepareProfile builds one
  exact owned NIC with native shared ACL, ingress reject, IP v4/v6/MAC
  filters, but does not explicitly set default egress reject.
- internal/native/box.go nicProfilePrivate intentionally treats absent
  egress setting as Incus 7.6's implicit reject; ownedPrivateNICEgress
  writes local per-Box reject when internet=false or allow when
  internet=true. Do not duplicate incompatible global egress logic.
- nativePrepareApply is already a separate reviewed root-only host action,
  not the same operation as install-stage or install-activate.
- native-scriptlet-plan/apply/status/rollback/reconcile already exist as
  root-only host commands with a durable original; successful route is
  ROUTE_APPLIED_PARITY_PENDING, not an installed parity acceptance.
- native-peer-purge-plan is a root-only read-only snapshot. No installed
  native-peer-purge-apply CLI exists; current protected root journal is one
  host-wide single-use attempt and cannot safely serve successive pairs.

Actual disposable L0 executable proof: Box
fe7cd681-80e4-48e8-8a6c-91e5258f0e06, original Exec
8d454a5c5e8ad412fa70d1db07e98e4d / native
4ce823a0-b26c-4114-abac-326a0c7f56a2 exit 0.
Nonroot root-plan and Scriptlet actions refused; root with no validated
database or original journal refused, no root intent created. This proves
fail-closed local entrypoint behavior, NOT an installed root handoff.

## Stage 1 native same-IP collision outcome: RISK_CONFIRMED

Independent native Incus 7.6 [Actions #44201216635](https://github.com/vanishs/boxweave/actions/runs/44201216635), job 134455484702, exact SHA 3071f25edfca5ab275b9da7a3d63333777fec35a, SUCCESS.

On the ephemeral stopped A/B/C host, two separate synthetic TCP IPv4 entries
shared original source and target IPs but had different source/destination
ports. The kernel proved BOTH present; a single deliberately test-only
conntrack -D -s/-d removed BOTH; subsequent native readback proved none left.
Original marker:
S4_B2_NATIVE_ROOT_IP_ONLY_DELETE_COLLATERAL_SAME_IP_DISTINCT_PORTS=RISK_CONFIRMED.
The native job also verified prior protected root journal gates and cleanup.

This is a genuine same-IP conntrack provenance risk, NOT a real foreign
Project guest test. Without a distinct source ownership discriminator,
IP-only deletion might remove another Project's matching IP flows. B2 must
fail closed for overlapping/unknown global bridge/route scopes or use an
independently proven zone-based kernel isolation mechanism. Installed
foreign-Project live session survival remains mandatory Stage 2 acceptance.

## Frozen Stage 2 product integration decisions (no more exploratory scope)

1. **Owned network admission.** Use one exclusive, restricted Project plus
   dedicated managed bridge and exact S1 profile. Fail closed if root/native
   global inventory shows foreign consumers, overlapping unmanaged
   routes/bridge address scopes, stale UUID/MAC/IPv4/IPv6 leases or
   per-NIC nft changes. Do not treat SQLite lease reservations as native
   ownership. Any overlap which cannot be independently ruled out is
   UNSUPPORTED, not approved under a blanket source/destination delete.
   Evaluate per-zone conntrack only if the native signed runner proves
   exact isolation; otherwise use exclusive non-overlapping admission.
2. **Restricted controller to host root.** A host-admin-installed,
   root-owned local helper is the only path to root operations. If using
   AF_UNIX it must validate SO_PEERCRED exact service UID, accept ONLY an
   original durable operation identifier and approved scope digest, never
   arbitrary shell, IP ranges or Incus objects, and independently re-read
   the protected SQLite ledger and real host-native Project/bridge/ACL/
   NIC inventory. Install activation cannot silently grant admin access.
   Separate explicit host-specific approval remains mandatory for
   daemon-global Scriptlet routing.
3. **Durable one-shot execution and recovery.** S4-A original idempotency
   and SQLite multi-resource claims protect Box/ACL/lease scope before
   any native change. Root-only per-original journal must be fsynced to
   UNKNOWN before dispatch, serialized with Scriptlet/ACL writers;
   repeat, crash, ACK loss and native drift may NOT blindly redispatch.
   A verified operation-specific reconciliation is required before
   safe reuse. No automatic global shell or conntrack -F.
4. **Native revoke barrier.** Quiesce every shared ACL NIC/Box with
   autostart off while holding leases; issue reviewed ETag dual-stack
   DROP and verify every effective per-NIC nft rule; only then remove
   verified owner-bound conntrack entries. Exact host readback must prove
   four directions cleared and unrelated third-Project TCP/UDP flows
   untouched. Resume selectively only when all postconditions hold;
   failures keep all claims and deny the network.
5. **Installed parity and upgrade.** Installer integration must
   preserve old restricted TLS Project/Profiles/Files/Exec permission
   matrix before/after Scriptlet routing, including tested root
   rollback and restart/partial failure. Explicitly handle existing
   SQLite schema mismatch (do not silently adopt/migrate v2 state).
   An isolated native SDK green test is not installed MCP OAuth proof.

## Stage 2 acceptance matrix (must ALL run on the integrated product)

- Clean installation and refused incompatible/foreign preexisting
  bridge/ACL/Scriptlet ownership; default denied vs reviewed internet.
- Installed controller actor and independent root caller with original
  operation ID + scope digest; no root arbitrary calls or wrong Project.
- Actual A/B IPv4/IPv6 live TCP+UDP allow and revoke, reverse/third-Box
  refusal and unrelated independent Project control surviving.
- Concurrent ETag conflict, failed native ACL/nft per-NIC update, daemon
  restart, lost ACK and process crash; exact persisted UNKNOWN/fencing,
  no replay and independent verified recovery/re-arm.
- Installed legacy restricted-client rights before/after/revert,
  exact global Scriptlet config and rollback/failed install recovery.
- One final SHA: gofmt, make verify, S4-A, dual arch, disposable signed
  Incus 7.6 native plus installed MCP/OAuth, independent reviewer,
  original operation receipts and exact cleanup.

Anything not established by these mandatory product-level tests remains
S4_B2_PARTIAL. Only after formal B2_PASS may B3 begin.

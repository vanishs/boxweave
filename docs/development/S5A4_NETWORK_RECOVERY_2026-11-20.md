# S5A4 — network and dependent Delete recovery

Status: **PARTIAL / BOX_SOURCE_PASS / NATIVE_PENDING**. Owner-directed continuation of PR #163 at exact base `c1773048bdbf184c074c7e1de26828c53896ee51`; no schema migration, deployment, host changes or new MCP tool. A3's signed lab is not actual trusted platform approval.

## Implementation

- Forward create/delete: independently read the original native success receipt and exact owned instance device (local and expanded config); compare persisted forward ID, UUID/incarnation, protocol, address/port, safe intent, create lineage and exact listener claim. Creation also rechecks current external exposure policy. Atomically settle existing original + ledger + independent recovery receipt; never issue another native effect.
- Peer grant/revoke: use existing root-scoped, read-only PRS reconciliation. Unix peer credentials authenticate the root helper; it selects its operator-pinned original and plan digest, never an MCP-provided digest. Require COMPLETE_VERIFIED journal with all native/kernel/conntrack proof, current exact peer binding/fences, active original credential and no conflicting Project operation. Incomplete journal remains UNKNOWN. Existing CLI path retained; shared MCP/CLI recovery uses the same bounded witness. A previous helper that lacks the read-only request mode fails closed.
- Dependent Delete: recover proven linked Forward Delete, Stop and Peer revoke children only while their exact parent is UNKNOWN and retains the Project barrier. No child replay, next child scheduling or automatic continuation. Parent settlement requires final original native success + exact absence, all children successful and dependency/fence checks in the final transaction.
- Existing generic claim release cannot release Forward/Peer or dependent-Delete parent claims. Approval-dependent partial compensation remains blocked, including until the actual A3 trusted-client evidence and production authorization mechanism exist. Do not infer absence equals success.

## Box evidence

Development Box `11bbc73e-2bce-4cf9-857f-799f768305f8`, detached exact base plus A4 patch at `/tmp/boxweave-s5a4`.

- Focused model/transaction tests: original BW `d3e6c4800c5b678391a921b94c3193eb`, native `b1c802f5-044c-4222-ab64-f3a42657d619`, guest exit 0. Forward create/delete, incomplete/drift/lock checks, peer grant/revoke, independent receipt/replay, atomic rollback, linked child and final parent settlement incl fences.
- Full regression + focused three-repeat race: BW `c1d7bad19dfd6a8e273e841eecd509c6`, native `bf96db7f-e546-472a-8606-53438bbac6cb`, guest exit 0. `go test ./... -count=1` all packages pass; `go test -race -count=3` A2/A4 and related state/recovery/control/native cases pass (native opt-in tests skipped). Additional final test-only changes scope the fault injection to the original pre-terminal transition and check root witness socket approval scope.

## Independent native acceptance

Activated S4-B4, S4-B2 controller/root and S4-B3 grant workflows on the S5A branch. Existing disposable fixtures and packet tests retained; A4 new test exercises real authenticated official SDK HTTP MCP handler, real Incus Forward create/delete on IPv4 and IPv6, TCP and UDP, exact original post-effect commit-cut, distinct settlement/replay and final dependent Delete. Root jobs reuse actual installed approved PRS fixture, completed grant/revoke lost ACK and incomplete revoke fault; new authenticated recovery handler must settle or block, preserve journal bytes, and old directional/foreign packet checks remain. These run in a disposable GH host; handler test binary is source-exact, not a claim of canonical installed MCP daemon parity. Require explicit new A4 PASS markers and native identities before signoff. Full installed daemon/client and physical integrated acceptance still S5B; no production use without review. No actual native execution is asserted by this checkpoint.

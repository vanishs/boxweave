# HOST_GO Round 1+2 scoped cleanup and cross-layer evidence follow-up

**Scope / verdict:** Owner requested completion of both previously blocked rounds. The two Round-1 owned Guests are CLEANUP PASS. Full Round 1 and Round 2 gate requirements remain BLOCKED on independently identified open evidence and actual development Agent connectivity. Round 3 physical host NOT_STARTED. No existing UNKNOWN was changed.

## Owned Guest cleanup: PASS

Lab: disposable managed boxweave-test-cli, nested Incus 7.6 restricted Project user-1002, original restricted nonroot actor and original persistent SQLite. Original v1 executable source 58626acb4d3ddf11f5bcd4e9fb0fd5d7fe87095f, SHA256 640cdd28d601b84ed3e1a35bdaf50322e13db46084417459d585a8d7ca1fb34f, dedicated temporary authenticated loopback MCP; original boxweave.service remained active.

- A name hgr1-58626-cli-a; original Create e89ce4335a89e3e1590497792708cc92; owned UUID 6b51eb00-fd33-4dfb-8d30-eadd55753a5f. NEW box_stop original 03df844d906f91aa7e66ee05814b216d, native Stop 5defce9c-536d-4be2-8898-f9dde8acb82f SUCCEEDED. NEW box_delete original 305e9e403b73f6d8e25fb371adb6203b, native Delete 60055c0e-967d-42b1-b4f0-5bfeffb8cbad SUCCEEDED.
- B name hgr1-58626-cli-b; original Create 02760b0a450a6951d24bdf4c8b53c5b3; UUID aa79c354-4019-405f-ad00-0731d1553bd6. NEW box_stop original a01cc38f55437880275283464cb26c45, native Stop 3989ace5-1836-4d4e-a6e2-66ecbd0b2b23 SUCCEEDED. NEW box_delete original 8c4c8cb61e01a08b964b175c8bd9ddf7, native Delete dc34a615-6af2-4768-8f4d-45bafea3b6bf SUCCEEDED.
- Both original owned UUIDs and Create SUCCEEDED were proven before any new mutation; internet was OFF with per-NIC ingress/egress reject and v4/v6/MAC filters. Each Stop and Delete was a distinct new supported official Go MCP SDK durable intent, not a replay of uncertain work. Independent SQLite original receipt read confirmed both terminal SUCCEEDED and owned_boxes state DELETED. Separate native Incus inventory found neither name; foreign bw-s25-l1a UUID 21767890-6d46-4867-889d-0069c1e4ac0d remained RUNNING. Temporary cleanup service stopped, original service active. Earlier blocked safety check remains recorded as blocked, never hidden.
- Scope: cleanup I installed official SDK, N native Incus and V separately read SQLite/native postflight on same test host PASS. Not a full L0+L1 independent BH-009/BH-015 verifier signoff.

## Same current-source Actions and independent artifact read

- PR #133 source d613737798219bf48c45efdf1afe9806eeb87542, quality run 43330914335 and 43330908378, S1 43330914221, disposable native 43330908322 and nested dual-agent/verifier 43330908376 all SUCCESS (5/5).
- Independent native nested job 131575066357 artifact 13995825832 ZIP SHA256 0586d01f4a5149b46b5ed479f9f643bac82a1f57f7e2e46d7562fc7a096e65b4 independently opened and hash checked. Its identity.txt records exact source d613737 and candidate executable SHA256 ec88dbc81f8c854a848381d9aea1a52a651e758433d7eb3e0eb36d629822254a, Incus APT 1:7.6-ubuntu24.04-202611072126. Raw original L0/L1 operation IDs/UUIDs, native NIC and simulated IPv6-WAN HTTPS packet evidence present. Scoped I+N+V PASS for separately routed simulated IPv6 WAN on both levels; two official SDK Agent clients; exact native/SQLite original operation readback; actual service SIGKILL/no replay; rollback and cleanup.
- CI itself explicitly printed S25_ALL_BH_AND_TC_FORMAL_SIGNOFF=NOT_YET. Its archived evidence does NOT by itself complete the whole current-source same-binary L0+L1 unproxied public IPv4 HTTPS + foreign/sibling bidirectional v4/v6, forged source IPv4/v6/MAC with legitimate controls, roles and separate full verifier signature. Previous source 58626acb local packet safety tests cannot be mislabelled as exact current source across both layers.

## Open mandatory gate

- ROUND1=BLOCKED: BH-009/BH-015 complete current-source same-executable L0+L1 I+N+V security packet matrix and formal independent signoff not yet available. The two owned Guest cleanup blockers are CLOSED/PASS.
- ROUND2=BLOCKED: TC-013 actual intended ChatGPT development Agent remains on legacy 0.50.x Gateway; no user-approved ChatGPT-to-v1 MCP authentication and private network connection. Two official Go SDK test Agents do not satisfy the agreed actual-Agent gate. No v1 BoxWeave plugin was found by plugin discovery. Remaining per-ID BH/TC matrix incomplete.
- HOST_GO=NOT_AUTHORIZED; Round 3 NOT_STARTED; public IPv6 real is separately deferred to S5 by owner approval; no such public IPv6 PASS claimed.
- Six original external gateway UNKNOWN, two original nested failed-start UNKNOWN and other historical native UNKNOWN never retried, adopted, deleted or changed. No physical host, foreign Guest or shared native Incus altered. PR remains Draft; no v1 merge or release.

**To close:** user approves and configures the actual intended v1-capable Agent and credential-preserving tunnel/auth path without public unauthenticated exposure; then run Tool list/read/one original-key write with independent I+N+V. Separately repeat and sign all missing same-executable L0+L1 BH-009/BH-015 real packet negatives, before formally promoting either Round 1 or Round 2 to PASS. Post-documentation same-head Actions recheck required.

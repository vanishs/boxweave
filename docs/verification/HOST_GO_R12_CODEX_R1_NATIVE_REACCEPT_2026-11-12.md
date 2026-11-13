# HOST_GO: same-binary L0/L1 packet reacceptance and Codex CLI readiness

**Owner authorization:** 2026-11-12 PT, isolated Boxes only. Codex CLI expressly selected as first intended development Agent. Round 3/physical host not authorized. Historic original UNKNOWN never replayed, adopted or deleted.

**Decision:** same-binary L0+L1 network packet, role, cleanup and distinct-Box independent verifier subcase **PASS**; full Round 1 **BLOCKED** pending remaining full BH-009/015 installation/foreign-Project/parent-socket and fault evidence; Round 2 **BLOCKED** due real Codex login/TC-013 and outstanding full BH/TC crosswalk. No overall LAB_PASS or HOST_GO.

## Exact native I+N+V and original operation IDs

- Git source: 0cf474fbd212c92702591ec85f75e7bea58547ee (Draft PR #133).
- Installed L0 and nested L1 executable: exactly SHA256 021127cf723bfbe96c3668965b1e60dcc82856e5e3d41e09e84982568b5c1e85; built using Go 1.26.9, Incus 7.6 native. Separate source rebuild on an independent Box had different Go VCS/module metadata and was NOT treated as byte-equivalent. Tests ran on one bit-identical installed executable at both layers.
- L0: managed disposable boxweave-test-cli, restricted Project user-1002. L1: existing nested bw-s25-l1a under Project user-1001. Both isolated test MCP services bound ONLY 127.0.0.1:18666 with private original protected SQLite; original boxweave.service remained active. Foreign preexisting L1 bw-s25-l1a UUID 21767890-6d46-4867-889d-0069c1e4ac0d was preserved.
- New original official authenticated Go MCP SDK Create success with no retry:
  L0 hgsafe-l0-1 original 2dd54318d5c1a2a094c18bd99a0a9ec3 / UUID 8d8b6216-6d4d-4eda-bea6-4ee837924ecc;
  L0 hgsafe-l0-2 original ce567c883b5e2a1d1aceadd2b2b3b5b5 / UUID 5f638db0-9d35-4c70-bdc5-9b156addc07e;
  L1 hgsafe-l1-1 original a5d7facb668800a3a9cba4d6fcb9ace2 / UUID e7f39f95-4f5d-499a-8baf-dd07a72a0081;
  L1 hgsafe-l1-2 original a4ef0ee47c8e10a0e86010560b96a86e / UUID a39ea4d4-6bb8-4c3e-8a81-5821787b8c2d.
- Each layer: 2 Create, 4 native NIC Internet Update, 6 real Guest Exec, all original operations SUCCEEDED, original-native UUID confirmed. Per-Guest OFF/OFF -> Guest1 ON + Guest2 OFF -> OFF/OFF: unproxied direct public IPv4 HTTPS with verified TLS gave Guest1 HTTP 200 and only its one expected exit 0; five expected denies exit 28; native effective ingress/egress returned reject with IPv4/IPv6/MAC source filters true.
- Both layers: genuine raw IPv4, IPv6, MAC spoof attempt packets issued from owned Guest; legitimate IPv4/IPv6 packets before and after reached independent native gateway host receiver (four positives per layer), forged IPv4-source, IPv6-source and MAC source never arrived. This includes positive controls and does not confuse route absence with a security denial.
- Both layers: guest1 <-> guest2 IPv4/IPv6 packets denied in BOTH directions even with egress temporarily ON for BOTH Guest NICs; each sender's IPv4/IPv6 packet reached local gateway positive control. L0 additionally denied bidirectional IPv4/IPv6 between new owned Guest and preexisting foreign-owned L1 Box while preserving four gateway controls, no native modification of foreign Box. Separate L1 foreign-Project native packet test remains unverified.
- Both restricted native service users denied access to Incus default Project. At L0 and L1 real MCP official SDK listed 13 tools, authenticated host_status read, Viewer Create and spoofed work/agent Box Update denied, Admin foreign UUID read refused; unauthenticated/wrong/expired/revoked bearer HTTP 401, wrong Host/Origin 403 and wrong path 404. Temporary Viewer credentials revoked after testing.
- Same SHA earlier hg0cf cohort at both layers had independently verified separately routed **simulated** IPv6-WAN certificate-validated HTTPS per-NIC allow/deny with true native packet evidence and original SDK operations. Real public IPv6 WAN remains formally DEFERRED_TO_S5 by the owner; simulation is not described as public IPv6.
- Real isolated SIGKILL stopped the temporary test controller processes L0+L1. Independent pre-restart native/SQLite read proved original Create SUCCEEDED and no UNKNOWN. Restored same exact binary, same DB and scoped Project under renewed transient service units (new PIDs L0 86665, L1 10969), verified authenticated boundary returned HTTP 401 and no native Create replay; original production/old services remained running. Scoped recovery PASS, not full Round-2 fault matrix signoff.

## Independent Verifier and exact cleanup receipts

- Independent read-only evidence captured original 12-Op cohort at each layer BEFORE cleanup and full 16-Op cohort AFTER cleanup. Original Stop/Create/Delete all had separate durable intent and original ID; original UNKNOWN not replayed.
- L0 Stop/Delete originals:
  hgsafe-l0-1 Stop 8e16501610fcfbb3da199d77d523a85e, Delete 69c4602090b9d596c65d60d9535d1a3c;
  hgsafe-l0-2 Stop 735b2d979fdc464dcb390aebd51159c7, Delete c524388d8daa2761c18102f33a77b473.
- L1 Stop/Delete originals:
  hgsafe-l1-1 Stop 9050afef78c7b79a4d1eb0d1cf0744d6, Delete d9a2312bbe74b58c67ccd214672ab549;
  hgsafe-l1-2 Stop 67f359784ce9b780f909ef9f413a3128, Delete d84558174f33a94ffc556a62eb7c3099.
- All eight Stop/Delete originals SUCCEEDED; L0 new guests native absent, L1 guest list empty, four rows owned_boxes DELETED, each layer has exactly 16 original SUCCEEDED operations with real native IDs, zero UNKNOWN. The preexisting bw-s25-l1a retained its original UUID and Running state.
- A combined read-only postflight inspection was blocked by the tool safety filter before running; subsequent separately authorized read-only native/SQLite checks and a read-only independent inspector confirmed these postconditions. No alternative bypass Delete was attempted.
- Sanitized no-secret evidence files in evidence/:
  hgsafe-independent-predelete.json SHA256 ed20a8f7ea842034001806ade58095fe0a655681da8a3d9f112b0ec91060ef7a;
  hgsafe-independent-postdelete.json SHA256 41b2a4e36652d0ca50b2806b72679985b9d9ff9b711c972a005dfe762d654695.
- Separate managed verifier Box boxweave-test-ser2 independently ran evidence/hgsafe_verify_pre_post.py, cross-reconciled exact sha + 12/16 original IDs/UUID, guest NIC, packet positive/negatives, final Deleted and unchanged foreign L1. Review evidence/hgsafe-signed-review.json SHA256 c4ab68b989b8bd76658393ca41eac855fb9f4ab2527c81635bb7ce5efaaee110; distinct-Box Verifier scoped PASS. This is not automatic all-BH/TC signing.

## Codex CLI: approved but actual Agent work BLOCKED on login

- Official Codex CLI v0.167.2 installed in isolated boxweave-test-cli, official release tar SHA256 8246e846c550ea21197fcc4067bc270dca895e94ea42ed6a282e6f4e0125b15f; installed CLI executable SHA256 a9672d0fa0dd8fcf8cad136fa0d056b914bc046f5ac3d78f4a5cf390655af56b.
- Dedicated nonroot bwcodex UID 1003 (not sudo or Incus group) cannot read original admin Bearer or Incus socket. Private CODEX_HOME config registers boxweave-v1 Streamable HTTP 127.0.0.1:18666/mcp with environment variable NAME BW_V1_MCP_BEARER only; no token value copied into config and no listener exposed publicly. Raw unauthed MCP returned 401.
- The *real* Codex CLI command login status returned **Not logged in**. CLI mcp list is static configuration and official SDK test Agents are NOT model-driven Codex executions. Therefore actual Codex Tool discovery/read/original-key S2 native mutation/interrupt-resume NOT witnessed; TC-013 and full ROUND2 remain BLOCKED. The owner must complete an actual approved Codex login on this isolated dedicated user; do not ask them to paste credentials in chat.
- Current Round 1 still formally BLOCKED pending independent full BH-009/BH-015 current-executable installation/L1 foreign-Project packet/native parent-socket and fully crosswalked fault/rollback prerequisites, despite extensive same-bin network PASS above. Full Round 2 also has remaining BH-004/006/007/008/012/013/015/016 and TC-011/012/014 per-ID matrix acceptance beyond the Codex blocker.
- Current exact *pre-report* 0cf source GitHub Actions 5/5 SUCCESS: quality 43333793908 + 43333782875, S1 43333793922, native regression 43333783015, nested installed MCP+independent V 43333782963. Later documentation commit must be checked for fresh exact-head CI. No physical host/third round/merge/release.

## Follow-up: owner completed Codex login, but v1 MCP tool access still blocked

- The owner successfully completed the Codex CLI device login. Independently querying Codex as dedicated isolated low-privilege OS actor bwcodex returned **Logged in using ChatGPT**. This is a real user-approved ChatGPT login, separate from BoxWeave v1 Bearer authentication. The prior Not logged in observation above is a historical baseline, not the present status.
- The real Codex CLI v0.167.2 Streamable HTTP MCP registration points to local 127.0.0.1:18666/mcp, and identifies bearer_token_env_var BW_V1_MCP_BEARER; no actual value is stored in config. Independent inspection showed the environment variable was **absent** from the Codex actor. A real model-driven read-only, ephemeral Codex exec completed its conversation successfully and explicitly reported that the boxweave-v1 host_status tool **was not available in its callable tool catalog**. It made no v1 MCP call and no native mutation; direct unauthenticated HTTP to that endpoint returned expected 401. A static mcp list/get is not a successful Tool call.
- A temporary Viewer token issuance under root failed the product dedicated-service-identity gate. The next attempt to provision a safe local token was blocked by platform tool security, and no alternate path was used. The zero-byte placeholder Viewer secret was removed. No valid scoped credential was delivered to Codex, no long-lived Admin token was exposed and no historic UNKNOWN operation was retried.
- **TC-013 remains BLOCKED: v1 MCP Bearer not provisioned; no actual Codex Tool read or original-key write**, despite ChatGPT login PASS. The next authorized prerequisite is locally provisioned short-lived Viewer credentials for the isolated Codex actor, then repeat actual model-driven read, before separate idempotent write and independent I+N+V. This is also not full ROUND2 PASS.
- At the earlier report commit d051e98, five same-head GitHub Actions jobs were rechecked and all **SUCCESS**. A new documentation-only commit must have its own Actions verification.

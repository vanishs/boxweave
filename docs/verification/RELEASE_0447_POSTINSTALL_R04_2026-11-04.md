# v0.44.7 exact-release installed R-04 follow-up — 2026-11-04

**Scope:** independently verified installed BoxWeave v0.44.7 in the approved isolated nested parent. This addendum supersedes pre-publication status only; it does not revise earlier historical test execution or claim a global cross-entry guarantee.

## Release and host provenance

- GitHub main and immutable v0.44.7 release commit: 4c1b88d06724ab93e4d997a99feefa54ced0bbe2; prior verify/publish CI #41896449530 PASS and official Release with 25 assets (release verification documented in the owner handoff).
- Actual parent: bwverify-0446-e2e-container, backend dev, 3 CPU / 6GB / 45GB, running in isolation. Inner Containarium daemon and native MCP both v0.128.0.
- The installed host CLI returned BoxWeave 0.44.7; independent `boxweave provenance` reported version 0.44.7 and source SHA 4c1b88d06724ab93e4d997a99feefa54ced0bbe2.
- `sudo boxweave doctor --mcp` returned overall passed, all ten required checks passed, authenticated MCP version 0.128.0 and REST inventory passed. An unprivileged doctor returned incomplete due to documented admin visibility requirements; it is not an installed health failure.
- Outer owner host was NOT upgraded. No owner-host Provider, SSH identity, network, stable selection or default release pointer changed. The default stable selector remains 0.43.1.

## Exact installed-release real-operation evidence

Before new mutation, the original private app database and claims database were backed up using SQLite backup into /home/boxweave/qa-0447-incident-evidence/ on the isolated parent; both original app configuration files were copied as mode 0600 within the same environment. No database contents, keys or secrets were exported. A temporary isolated-only lifecycle allowlist extension added bwweb-0449 for a new official-release acceptance run; the application worker was restarted and active. After successful cleanup, the original application configuration bytes were restored and the worker remained active.

| Gate | Exact live observation | Status |
| --- | --- | --- |
| Official 0.44.7 new app-owned lifecycle creation | Plan 691af7d1aca34a6297f107c29d06daa9, operation ef4f8c4f117546d8b1a57a4a242bcbaf; receipt succeeded / connection_ready / OK; Incus reported RUNNING | L PASS |
| Immutable Box identity | Application target bwweb-0449-container, user bwweb-0449, created 1793822887; independent native Provider get returned exactly that createdAt, 1 CPU / 1GB / 10GB and IP 10.203.0.5 | L PASS |
| Post-create released claim | The new operation reached coordination_operations observed_terminal; its resource claim no longer remained | L PASS |
| Protected direct execution | Real installed v0.44.7 nested MCP `connect(exec=true)` to bwweb-0449 refused COORDINATION_USE_TRACKED_TASK | L PASS negative |
| Original UNKNOWN write conflict | Installed nested Gateway task for bwweb-0447 with harmless `true` returned RECOVERY_REQUIRED | L PASS negative |
| Concurrent tracked-task deduplication | Two parallel gateway calls with the identical request key on bwweb-0449 returned completed / exit 0 and the same task ID a994cc603d89025e94b427c64dc947ba; bounded guest file read returned exactly one byte (base64 MQ==, SHA-256 6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b) | L PASS |
| Gateway lease vs Web application resize | On original successful bwweb-0448, a new v0.44.7 installed live lease blocked app operation 767f19df4e8b44a7b52d1bee58d2496b as failed / RESOURCE_BUSY; native Provider CPU remained 1, then lease release succeeded | L PASS |
| Post-lease normal resize | On bwweb-0449, app CPU resize to 2 (operation a1535be28ca140f1b1ec8cae596e9bcf) and back to 1 (operation abafd2b490f5491d897b78dc29fefb16) reported succeeded / resized / OK; final native Provider read confirmed CPU 1. No independent intermediate Provider CPU 2 read was captured in this run | L PASS for receipts and final 1; intermediate read U |
| Test resource cleanup | New bwweb-0449 stop operation eb48f581806240d399dce39ae60dc4cc succeeded / stopped / OK, then delete operation e4ec7db3fe034765bfdaea305907e7ac succeeded / absent / OK; Incus inventory returned only the three retained historical bwweb-0446/0447/0448 Boxes | L PASS |

**Trust readiness boundary:** The exact official v0.44.7 version demonstrated a new real successful create and trusted connection readiness. The spontaneous first-observation TRUST_FAILED condition was **not** observed for this new child. The bounded retry logic retains the previously tested source fixtures (36 focused and 738 complete Python tests at the source/candidate checkpoint), but actual release-installed failure-then-recovery remains **U**. Do not mark that narrower fault-injection path as live PASS.

## Original UNKNOWN incidents: preserved, not cleared

- Original application operations f654ae360e8e458485968c4b3bdabe54 (bwweb-0446) and fe71f6a4816e4fbab690a09fa83b3622 (bwweb-0447) still have **failed / blocked / TRUST_FAILED** application receipts, while the common coordination database still records **unknown** and one retained resource claim each. No active leases were present after the new tests.
- Native Containarium get returned both original named children RUNNING: bwweb-0446 createdAt 1793809108 and bwweb-0447 createdAt 1793819412. The application environment registry separately contains a **deleted** old bwweb-0446 target with created 1793807060, not the current bwweb-0446 Provider incarnation; bwweb-0447 has no adopted active app environment record. These conflicting identity records prohibit a guessed manual resolution.
- No original child was stopped, deleted, recreated or adopted. No direct claim database writes or blind Provider create retries were performed. Original private evidence and unchanged claims remain available on the test parent.
- **Recovery remains BLOCKED** until the original dispatch/receipt, current Provider incarnation and historical host-key/daemon logs are reconciled through an explicit safe recovery procedure. Merely seeing a running same-name Box is insufficient to convert failed receipts to succeeded or release a name claim.

## Remaining gates and external boundaries

- **Outer-host R-02:** `boxweave_task_list` still returns TASK_TRACKING_UNAVAILABLE on the outer Gateway. Provisioning requires separately authorized host root action; no outer-host action performed.
- **R-04 global matrix:** Other Web/REST/SSH/legacy MCP disconnection, lease-expiry, crash and UNKNOWN combinations remain U. Readiness race on official install remains U despite the successful create.
- **Containarium upstream:** Create-after-error and native atomic disk shrink semantics remain OPEN for evidence and tracking **only**; no upstream code or runtime was changed.
- **Real installed Web HTTP with administrator login, D4 positive diagnostics, external OAuth/TLS clients, Provider rollback, E2–E4 and OTel collection:** U or BLOCKED as in the current roadmap. An authenticated Flask fixture is not equivalent to real installed Web login.
- No BoxWeave product defect requiring a new version was established by this follow-up. No new release and no stable-selector promotion was performed.

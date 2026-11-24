# PR #178 N3 installed Peer acceptance (2026-11-23)

Status: implementation and isolated installed acceptance complete; exact-head CI pending. No merge/release authorization. N4 broader fault/traffic matrix and N5 release work remain separate.

## Contract

Fresh scoped-v2 installations admit exact directional Peer requests through MCP. Controller stages durable original and claims; authenticated Root applies independent ACL/Guard plan and returns a verified receipt. Controller atomically settles the original. Same-key replay precedes native lookup. UNKNOWN recovery requires completed Root journal and live kernel evidence; no mutation replay or blind unlock. Legacy manual pathway is preserved; no in-place migration.

Reverse allow and revoke coexist without a symmetric DROP breaking the authorized reverse flow. Stateful Guard controls directional deny; unrelated ACLs and boot flags are unchanged. Root validates canonical UsedBy inventories and tenant-scoped leases, not default project's gateway-only lease view.

## Local installed evidence

Parent b977764b9079ec8e428f70410e36395de6ffdad4 passed four CI workflows. Owned disposable nested Boxes only; no production Host/Box edits. Inherited outer nesting profile was unconfined; no security settings relaxed. Positive nested tests do not establish physical Host enforcement parity.

- Full Go suite and format/diff checks: d6078d149c416eb2f49374ec61101ce4, exit 0.
- First fresh installation 54e43774-c82f-44c7-83b8-2b4bc0309871: grant 9ae4afc39b14d22ae5d541d080eeea1b failed closed before Peer journal due to lease project scope. Original remains UNKNOWN/fenced, no replay or forced unlock. Read-only root scope/prefix retest fe39475e726c87c6ef779ba339ab3537 passed after tenant lease/canonical ordering fixes.
- Second fresh installation 8d93d497-95e7-46af-925c-c6e08f08eb5a: archive SHA256 a66be8480de24ed5ff8542d9e63ece45b1bfcf5bea12772b7f143a918025299a; installed candidate SHA256 581c0fc439fdebe292f5dadd9d584711c2ea9aa401a2af2672935b092591b0a7. Tested development source on parent, not exact new commit.
- Operation 196e233a311cfbbfd55564d2b69acdae recorded N2 lifecycle PASS, default isolation, C->D, A->B, B->A grants and A->B revoke. PR178_N3_INSTALLED_LIVE_FLOWS=PASS: established IPv4/IPv6 TCP A->B blocked; B->A and C->D original sockets still exchanged data. Outer wrapper timed out (124) after markers, not counted as exit 0. Regrant 31791cb89948870ad84ae802e12c08c9 verified SUCCEEDED by 46e1104e4c14da33b447eaddcded3efc. Separate postflight 5b6f8be6ba8a9fe79f43ba3b304b5ed9 exit 0: exact replay, regrant connectivity, extra Box create/delete with active policy, unchanged four ACLs/autostart/running states, Guard readback and lifecycle Root service restart.
- Actual completed receipt loss: lab-only restricted actor fixture d71c2fb9013037824dc1bd50062313cb exit 0. It deliberately omitted controller settlement after Root completed revoke a5289068d820d4ccf901a3623e4cd855, then restarted controller. MCP inspect/reconcile 531f1d4410eb1de2d1ec5d04682aab02 exit 0: UNKNOWN -> SUCCEEDED, recovery f82b1c0c63726bc9ba7115c5474834dd; native_retried=false, repeat same recovery receipt, unchanged Root JSON journal hashes.
- Cold stop cc77d62f7a09dbc470c26c220a0b202c / start 549eda288ffd551ba24e39c2c534442e succeeded. Postflight ef97c2563500b2672642cc2e568b1822 exit 0: Guard monotonic active timestamp 243151469409 before Incus 243155398627; all three services active; B stays stopped, A/C/D start; same ACLs and autostart flags; C->D allowed, D->C and A->C blocked.

## Exact-head CI

Scoped installed matrix runs N2 regressions plus real MCP directional grant/revoke/regrant, persistent dual-stack TCP sockets, replay, active-policy lifecycle and Root service restart/Guard readback. Explicit markers required; legacy matrix remains. Actual cold stop/start and completed-ACK-loss evidence above are local lab evidence, not hosted CI reboot claims. Go regression suite exercises scoped replay, concurrency and recovery settlement. Broader Forward/traffic and uncertain partial-write fault matrix remain N4; no claim of N4/N5 completion.

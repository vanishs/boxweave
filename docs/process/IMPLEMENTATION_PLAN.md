# Implementation plan — permanent narrow product

**Status:** no v1 implementation exists yet. The numbering below is an engineering dependency order, **not P0-to-P1 deferral of excluded products**. Deleted features are outside scope.

| Order | Implementation slice | Required real proof |
|---|---|---|
| 0 | Pin reviewed Go MCP/Incus SDKs and Incus daemon API extensions; freeze 20 MCP schemas and limited admin/viewer credentials | Go compile + real SDK calls; native security and restricted Project capability evidence |
| 1 | Host-local Incus-aware installer/status/upgrade/bootstrap; version/capability ledger; no Caddy; reuse compatible existing daemon | Fresh Ubuntu install and existing Incus reuse; shared upgrade approval; no foreign modification or automatic reboot |
| 2 | Loopback authenticated MCP, SQLite owned-Box records, universal Tracking/context, caller idempotency and durable async worker/UNKNOWN | Auth positive/negative, dedup and crash/restart original receipt, viewer cannot mutate/Exec |
| 3 | Create/list/get/update/start/stop/delete owned Box and native async guest-root Exec + output/readiness/cancel | Real guest test/build with nonzero/zero exit, disconnect/recover, native post-state and foreign denial |
| 4 | Native guest file list/read/create/conditional replace/delete with bounded safe hashes, and original Operation diagnostic summary | Real create->read->edit->fail->fix->retest, unsafe path/race refusal, no patch engine or diagnostic service |
| 5 | Fail-closed native private network isolation + directional `box_peer_set` and TCP/UDP native port forwards | Real A->B deny/allow/revoke, IPv4/IPv6, loopback/public mapping, TCP+UDP, collision/reboot, foreign state intact |
| 6 | Self-contained nested install/upgrade and same functional contract in L1/L2; no parent orchestration | Repeat full real Agent loop; inner Incus does not receive parent sockets/credentials |
| 7 | Black-box Finder/Fixer/independent Verifier, CI assets and immutable version publishing | 20 MCP tools plus local CLI leaves: positive/negative native evidence; no source-only PASS for installed function |

Officially support Ubuntu LTS first. Other guest distributions can be certified on evidence, without making the first product incompatible with future proven Linux images. No phase in this plan adds Caddy, general ACL editing, RBAC, snapshots, OAuth, file patch engine, bulk import, host automation, process supervisor or idle wake.

See [boundary](../decisions/PRODUCT_BOUNDARY.md), [Agent loop](../product/DEVELOPMENT_LOOP.md), [network](../design/NETWORK.md), [acceptance](../verification/ACCEPTANCE.md).

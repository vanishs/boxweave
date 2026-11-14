# RC.3 physical-host pilot: scope and evidence (2026-11-13)

**Subsequent owner-approved 2026-11-13 PT / 2026-11-14 UTC nested post-pilot addendum (latest scoped evidence):** The L0 RC.3 host created an owned L1 development Box, installed an independent matching RC.3 L1 controller/Incus 7.6, and proved direct authenticated **L1 `box_create` / `box_get` / `box_list` / guest `box_exec` PASS** against an L1-owned L2. L1 was reached through its own operator-managed Quick Tunnel and OAuth connector; no inner credentials were delegated from L0. In contrast, an indirect L0 `box_exec` → L1 Python → inner MCP **read `box_list` PASS** while an indirect inner `box_create` was **blocked before the outer Exec was dispatched** by platform safety checks (L1 retained the same Operation/Box/native inventory). This does **not** show an inner BoxWeave failure or prove the hypothesized missing second-layer ChatGPT approval. [Full chronology, exact IDs, security hypotheses and evidence](RC3_NESTED_L0_L1_L2_AND_MCP_BOUNDARY_2026-11-13.md). The original read-only pilot table and original 'Explicitly NOT passed' section below are a **historical earlier checkpoint**, not a reversal of this later Create PASS. No real packet/quota/fault/rollback/independent-Verifier signoff or `HOST_GO=PASS` follows from this addendum.

**Source distinction:** A human operator reported the physical-host command outputs during installation. Subsequently, the assistant **directly invoked** three read-only tools through the newly connected `boxweave v1rc3` MCP plugin and observed their replies. This is a dated scoped report, **not an independent verifier signoff**. Never copy machine-identifying hashes, private tunnel hostname or credentials into reusable installation docs.

## Immutable release / CI identity

- [v1.0.0-rc.3 Release](https://github.com/vanishs/boxweave/releases/tag/v1.0.0-rc.3): tagged source commit `1ce1774deca99fac4a3bb1031779ccacea0289d7`.
- Linux amd64 binary SHA256: `2d35fbaf778362ec15aab78f092c55c3a5ad85eabc00e578a73bf80389ca05be`.
- [Release Actions #43552351245](https://github.com/vanishs/boxweave/actions/runs/43552351245) succeeded for quality, S2 native, S2.5 nested, dual-arch artifacts and immutable publisher. CI is **not** the physical host.

## Observed physical-host sequence

| Area | Observed evidence | Scoped verdict |
| --- | --- | --- |
| Host/native | Ubuntu LTS real host, Incus client/server **7.6**, existing **Btrfs** pool, dedicated `incus`-only nonroot service, private per-user Incus Project/bridge | Operator-reported setup, not universal compatibility |
| Install | Exact-tag `install.sh --plan` then hash-bound `--apply`; manifest `version=1.0.0-rc.3`, expected commit/binary SHA, `service_active=true` | **PASS** operator-provided installed build |
| Local HTTP | `systemctl is-active` → `active`; anonymous `POST 127.0.0.1:18650/mcp` → **401** | **PASS** running loopback + unauthorized denial |
| Native prepare | `native-prepare-apply` → `APPLIED`, narrowed owned restricted Project, default-deny anti-spoof NIC, exclusive new ACL | **PASS** config/readback; not packet witness |
| Prerequisite | `native-prereq-plan` and SHA-approved `native-prereq-verify` → eight checks true, `config_ready=true`, `packet_witness_required=true` | **PASS** eight configuration checks |
| Nonroot doctor | `mcp_ready=true`, `native.service_ready=true`, `native.ready=true`, `ready=true`, same version/commit | **PASS** admission readiness |
| HTTPS OAuth | Operator-run Cloudflare Quick Tunnel, local OAuth approval secret, `oauth-status` showing matching public Origin/`Enabled=true` | **PASS** configured origin; secret not collected |
| Real ChatGPT MCP | `host_status({})`, `box_list({limit:10})`, `operation_list({limit:10})` actually called via connected tool and **all returned**; version/commit/Incus/readiness matched; inventories were initially empty | **PASS** authenticated remote read/tool discovery |

The status also reported `default_image_local_ready=false`, `selected_storage_quota_verified=false`, `selected_storage_space_checked=false`, and warnings `DEFAULT_IMAGE_ON_DEMAND` / `STORAGE_QUOTA_UNVERIFIED`. Those are truthful **unverified** capabilities, not install failures.

## Explicitly NOT passed

- No first on-demand Ubuntu Guest image download, `box_create`, Start/Exec/output/Stop/Delete, original mutation receipts or native UUID readback on this host.
- No real IPv4/IPv6 default-private packet witness/anti-spoof/Internet toggle, nor host-route negative tests.
- No native Btrfs size/quota and Exec output-spool bounding validation.
- No physical-host failure/rollback, real power-fault, or separately signed independent verifier.
- Therefore **`HOST_POSTFLIGHT_PASS`, `HOST_GO=PASS` and S5 stable Release are not signed**.

**Next bounded test:** one expressly authorized MCP `box_create` with preserved idempotency/Operation evidence and independently observed exact Incus UUID; never replay an `UNKNOWN` mutation. After that run actual packet, quota, fault and rollback acceptance.

This dated report supersedes **only** older general statements that RC.3 had never been published or that *no* physical pilot was installed. Historical lab and older PR/host evidence remain attributed to their own time/scope. See [manual install](../getting-started/MANUAL_INSTALL.md), [pilot gates](../development/V1_PILOT_DEPLOY_AND_RELEASE_CONTRACT.md) and [RC.3 decision](../decisions/V1_RC3_INCUS_NATIVE_INSTALL_AND_IMAGE_POLICY_2026-11-13.md).

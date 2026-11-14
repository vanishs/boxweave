# RC.3 physical L0 → L1 → L2 nesting and nested-MCP dispatch boundary (2026-11-13 PT / 2026-11-14 UTC)

**Status: L1 native Create PASS; L1 direct authenticated BoxWeave MCP Create/Exec PASS; L0 `box_exec` → L1 Python → local MCP read PASS; indirect inner `box_create` PRE-DISPATCH BLOCKED.** This is a **scoped, live operator-approved experiment**, not an independent verifier, unrestricted nesting guarantee, or `HOST_GO=PASS`. The exact evidence below supersedes only older statements that this RC.3 physical topology had never executed an owned native Box Create; historical failures remain as they occurred.

**Evidence origin:** The operator supplied interactive Incus CLI, expanded instance config, AppArmor and cleanup output. The assistant directly invoked the L0 and subsequently the independently OAuth-connected L1 BoxWeave MCP tools, read original Operation receipts, Exec output, and the L1 controller's own SQLite/native/journal **read-only** snapshots. Do not copy the disposable Quick Tunnel hostname, machine fingerprint, bearer/OAuth secrets, password, callback URL, or live grants into documentation. The original tool transcripts remain in the owner conversation; this report records the verifiable IDs and boundaries.

## 1. Topology, versions, ownership

```text
L0: physical dedicated host, Incus client/server 7.6
  BoxWeave 1.0.0-rc.3 (commit 1ce1774deca99fac4a3bb1031779ccacea0289d7)
  owned L1 Box: rc3-l1-nested-1113
    L1: Ubuntu 24.04.5, Incus client/server 7.6, independent BoxWeave RC.3
      independent local SQLite, restricted incus-only service identity,
      Project user-1001, bridge incusbr-1001, pool default (dir),
      loopback HTTP MCP 127.0.0.1:18650/mcp
      L1-owned L2: rc3-l2-nested-1113 (Ubuntu 24.04.5)
        no BoxWeave L2 controller installed or claimed by this experiment
```

- Same pinned RC.3 source and binary were verified on L0 and L1: release binary SHA-256 `2d35fbaf778362ec15aab78f092c55c3a5ad85eabc00e578a73bf80389ca05be`. L0 owns L1 UUID `cd1ea3d0-a24f-477b-b01c-08fb5acac53e`, original L0 Create Operation `ad6f3170600f933c564f6d9c79317929`. L1 has its **own** storage, identity, operations and credentials. This is **two active controllers / three container layers**, not evidence of an installed controller at L2 or an L2→L3 guest.
- L1 `boxweave` runs as UID 1001, Incus-only group GID 990, not Incus admin. Its confined user Project was prepared with exclusive ACL `bw-l1-nested-1113-acl` and 8/8 native prerequisite checks. The independent local HTTP `/mcp` responds 401 without authorization; an authenticated temporary local client previously completed `initialize`, `tools/list`, and `host_status`.
- The L1 `dir` pool is usable; `selected_storage_quota_verified=false`, `selected_storage_separate_mount=false` and `selected_storage_space_checked=false`. A requested `size=16GiB` is **not evidence of an enforced 16-GiB hard quota** or bounded Exec spool. Likewise, network ACL/anti-spoof configuration readback is not an IPv4/IPv6 packet witness.

## 2. Operator's Incus-native L2 control (not a BoxWeave-owned Box)

1. From L0, the operator entered L1's root shell and confirmed L1 Incus 7.6 and `id boxweave`; `runuser -u boxweave -- ... incus list --format=json` returned `[]`.
2. The first manual `incus launch` attempt failed **before instance creation**, with `Invalid device type "disk,path=/,pool=default,size=16GiB"`. Root/NIC device CLI arguments were formatted incorrectly in the assistant's proposed command. A following Incus list proved no instance existed. **This was a test-command error, not a nesting failure.**
3. A corrected CLI command inherited root/NIC from the restricted default Profile and applied device overrides `-d root,size=16GiB` and `-d eth0,security.acls.default.egress.action=allow`, with `security.nesting=true`, one CPU and 2048 MiB. It created and launched `native-l2-probe-1113` successfully as `boxweave` in Project `user-1001`. The operator observed `RUNNING`, IPv4 `10.210.99.2`, and a private ULA IPv6 address.
4. `incus config show --expanded` showed CPU `1`, RAM `2048MiB`, `security.nesting=true`, root disk `default/16GiB`, inherited ACL, ingress `reject`, egress `allow`, and `security.ipv4_filtering`, `security.ipv6_filtering`, `security.mac_filtering` all `true`. Incus generated `raw.idmap` containing the confined service UID/GID. This proves **native effective configuration**, not packet enforcement or per-Box disk quotas.
5. Guest `/proc/1/attr/current` and `/proc/self/attr/current` reported an enforced `incus-user-1001_native-l2-probe-1113_...` AppArmor profile; kernel AppArmor domain stacking reported `yes`. Stacking **does not imply policy-admin permission**.
6. The operator deleted this **manual, foreign-to-BoxWeave** probe with `incus delete native-l2-probe-1113 --project user-1001 --force`, then reported cleanup. No inner BoxWeave-owned UUID or Operation was created by this native-only path. Never expect `box_list` to register manually launched Incus objects or adopt/delete such a probe as owned.

## 3. Actual L1 BoxWeave-owned L2 via direct ChatGPT OAuth MCP: PASS

The assistant deployed `cloudflared 2026.11.0` **inside L1**, using Cloudflare's signed apt repository and a separate sandboxed `boxweave-l1-quick-tunnel.service` pointing only at `http://127.0.0.1:18650`. The service was **started**, not enabled for reboot; the ephemeral `trycloudflare.com` origin was registered with `boxweave oauth-set-origin`. Before consent-password setup, the expected policy denied the external Host (`HTTP 403 invalid Host`) even with the Origin set and `Enabled=false`. The operator then configured the distinct OAuth approval password via a non-echoing L1 terminal. Readback: `Enabled=true`, OAuth AS discovery `HTTP 200` with valid HTTPS verification, anonymous `POST /mcp` `HTTP 401`; no credential was published. L0's tunnel was not replaced.

The owner connected **a separate named ChatGPT MCP connector for L1**, distinct from the L0 connector. On that **direct** L1 connection, `host_status` gave RC.3, Incus 7.6, Project `user-1001`, `ready=true`; initial `box_list` and `operation_list` were both empty.

| Real L1-owned action | Original evidence | Verdict |
| --- | --- | --- |
| `box_create`: `rc3-l2-nested-1113`, `image=images:ubuntu/24.04`, CPU 1, RAM 2048 MiB, root 16 GiB, `nesting=true`, `internet=true`, `start=true` | Operation `24f581df75869ede94f6653e477122d1`, native Operation `15c8f7cd-ba8a-4a73-9647-ce76e468bab7`, `SUCCEEDED` at about 2026-11-14 05:40 UTC | **PASS** |
| Identity and ownership | L1-owned exact UUID `92008ee4-d06f-4664-8de7-cb2bf8f6f647`; `box_get` and `box_list` both return the same Running Box, 1 CPU/2048MiB/16GiB, nesting and Internet true | **PASS** |
| Actual guest Exec | Original `box_exec` Operation `aec2706c24c1a21a0b863197a6dd177c`: `SUCCEEDED`; **guest exit 0**, `command_success=true`; hostname `rc3-l2-nested-1113`, Ubuntu 24.04.5, enforced Incus AppArmor profile | **PASS** |
| Third-level Incus/BoxWeave at L2 | No RC.3 `incus admin init`, guest bridge, L3 guest or L2 controller tested | **NOT TESTED** |

The RC.3 L2 was **last observed Running and intentionally retained**. It is an ordinary L1-owned Box; it must be stopped/deleted later only with its original L1 owner and the usual exact-UUID/operation safety gates. No automatic cascade management from L0 is implied.

## 4. Controlled contrast: L0 `box_exec` → L1 Python → L1's loopback MCP

This is a **different client path**, not a hidden cross-layer feature. It executes ordinary guest software through an authorized L0-owned Box. The inner MCP has its **own** local Bearer credentials and authorization. Tests used temporary token files local to L1 with secret values not exposed in argv, stdout or shared logs; temporary test tokens were revoked.

```text
ChatGPT → L0 box_exec(L1) → guest-local Python HTTP client
                                     → http://127.0.0.1:18650/mcp
                                         → initialize
                                         → tools/call(box_list or box_create)
```

| Attempt / evidence | Outer L0 dispatch | Inner L1 evidence | Result |
| --- | --- | --- | --- |
| Earlier authenticated guest-local `initialize`/`tools/list`/`host_status` | `box_exec` ran | L1 localhost responded 200 | **PASS** local MCP availability |
| Earlier inner `box_create` attempt, before external L1 OAuth connection | Tool returned `此工具调用被 OpenAI 的安全检查屏蔽。请仔细检查你发送的内容。` | No outer operation receipt or inner create record; empty L1 SQLite/Incus at that time | **BLOCKED BEFORE OUTER DISPATCH** |
| Reattempt with token creation step | One temporary admin token was actually created by a separately accepted L0 Exec, later explicitly revoked/file removed; follow-up Python `box_create` Exec was blocked | At that time L1 operations/owned boxes/native instances remained empty | **BLOCKED BEFORE OUTER DISPATCH** |
| **Read-only `box_list` via Python**, 2026-11-14 ~05:52 UTC | L0 original Exec Operation `b5e7f94c2defe4c62fd098492f3a82d7`: `SUCCEEDED`, guest exit 0, `command_success=true` | `initialize` HTTP 200; `tools/call(box_list)` HTTP 200, returned existing L1-owned L2 UUID `92008ee4-...` Running | **PASS**, exact same nested client pattern |
| **Mutating `box_create` via Python**, separately approved ~05:56 UTC | L0 `box_exec` itself returned OpenAI safety-check block; **no L0 Operation ID**; no new Python process evidenced | L1 `operation_list` remained the **same two** direct-MCP Operations, `box_list` one Box, SQLite `operations=2`, `owned_boxes=1`, native list only original L2; no corresponding new service journal request | **BLOCKED BEFORE OUTER DISPATCH**, not an inner MCP refusal |

The first `box_list` test attempt also had a **test script bug** parsing its temporary viewer-token ID; its guest shell exited 1 before reaching Python. This failed attempt is **not evidence of MCP failure**. The erroneous viewer token was revoked, the parser was corrected, and the second `box_list` test passed. Subsequent read-only SQLite verification found zero active viewer tokens and both temporary viewer token rows revoked. No existing Box was changed.

**Key inference:** If L1's HTTP server had received and rejected an inner `box_create`, we would expect an actual L0 Exec receipt, guest output/HTTP status or corresponding L1 request traces. Instead the **outer** L0 MCP tool call was denied before the BoxWeave Exec could be admitted. The exact OpenAI safety-check classification is opaque; do **not** claim it ran Python and then failed on L1, or that L1's `box_create` is defective.

## 5. Identity, permission and second-layer approval hypothesis: keep separate

The owner proposed a reasonable hypothesis: the human approves an outer ChatGPT MCP operation, but an inner tool mutation initiated by guest Python has no second ChatGPT UI to ask the human for approval, so the platform might block it. This is **an unverified hypothesis, not an established root cause**.

Distinguish three gates:

1. **ChatGPT host-side tool safety/approval:** applies before dispatching L0 `box_exec` (or the separately connected L1 `box_create`). It may refuse a request with an OpenAI safety-check error; the exact internal rules and whether approval is available are not exposed here.
2. **Inner MCP identity and role authorization:** guest Python held a separately issued L1 admin Bearer token on the attempted create path (or viewer on reads). This is L1's own application authentication, **not a second ChatGPT OAuth approval**. A blocked outer Exec never reaches that check; it cannot demonstrate missing/failed inner authentication.
3. **Native Incus authorization:** L1 BoxWeave acts through its own restricted user/Project, with no access to L0's native admin socket. Successful direct L1 MCP Create shows the native path works under its approved permissions.

The L1 direct OAuth MCP connection separately obtained the user's interactive ChatGPT authorization and its own native Write succeeded. **No authorization automatically propagates across layers** and v1 has **no parent orchestrator, nested approval forwarding, cross-layer delegation or privileged bypass API**. There is no demonstrated supported mechanism that allows an outer `box_exec` to solicit a second ChatGPT tool approval in the middle of arbitrary guest Python execution. This observation alone does **not** prove the platform denied the operation specifically for lack of that approval.

Other confirmed public observations show pre-dispatch safety blocks even for **ordinary first-level read-only calls**. OpenAI Support acknowledged incorrect blocks of legitimate read-only app actions and reported improvements while investigation continued ([2026-08-05 statement](https://community.openai.com/t/chatgpt-app-mcp-tool-calls-blocked-by-openai-safety-checks-before-reaching-mcp-server/1386059/22), [2026-08-16 follow-up](https://community.openai.com/t/chatgpt-app-mcp-tool-calls-blocked-by-openai-safety-checks-before-reaching-mcp-server/1386059/24)). Separate developer reports include [authorized calls absent from Cloudflare Tunnel logs](https://community.openai.com/t/chatgpt-mcp-tool-calls-blocked-by-openais-safety-checks-read-only-requests-absent-from-tunnel-logs/1405342), [more frequent write than read blocks without server request](https://community.openai.com/t/custom-mcp-connector-valid-tool-calls-blocked-by-safety-checks-server-logs-confirm-they-never-arrive/1407832), and [one developer's dedicated-tool/annotation experiment with explicitly unproven causation](https://community.openai.com/t/chatgpt-mcp-tool-calls-blocked-by-openais-safety-checks-read-only-requests-absent-from-tunnel-logs/1405342/13). These are **external reports**, not BoxWeave's own verified causality. Rewriting tool descriptions or bypassing safety checks is **not** a validated fix.

## 5.1. Subsequent OpenAI Help AI-assisted response: product integration knowledge (not an engineering verdict)

**Evidence class: user-reported AI-assisted support guidance, after the RC.3 reproduction; not a formal policy commitment, authenticated engineering root-cause finding, or proof of a permanent restriction.** The owner sent the sanitized direct-versus-indirect reproduction to OpenAI Help and received two AI-assisted explanations. Preserve their **substance**, not chat transcripts or verbatim conversation:

- **Approval boundary:** ChatGPT may ask the human to confirm direct MCP write/modify actions; some higher-risk actions may instead be blocked **outright before tool dispatch**, without offering approval. This matches the observed L0 `box_exec` failure mode but does not reveal the exact decision rule for this request.
- **No documented nested/pre-approval mechanism:** The Help AI could not identify a public supported mechanism to obtain a *second ChatGPT approval inside an already running outer tool* or to **pre-authorize inner tool mutations**. This supports the owner's missing-interactive-approval *compatibility hypothesis*; it **does not prove** the classifier recognized a second-level MCP write or that all nested writes are intentionally prohibited.
- **Direct first-class tools:** The Help AI recommended making sensitive writes visible as **top-level, first-class MCP tools** so ChatGPT can apply its normal approval/safety policy. An outer `box_exec` running arbitrary guest code is not such a distinct registered inner write tool. Even direct first-class tools are not guaranteed to be approved: some requests may still be blocked.
- **Gateway option, not a BoxWeave v1 requirement:** A **separately built** single MCP front-end could selectively expose allowed operations from multiple private backends and surface each requested write as an explicitly approved tool. This shifts orchestration/policy to the front-end; implementing it inside BoxWeave v1 would expand scope and contradict the approved small, layer-local controller boundary. Do **not** silently build or require it.
- **Private-connection option:** The Help AI also mentioned **Secure MCP Tunnel** for connecting otherwise private servers individually to ChatGPT, alongside allowed-tool/action restrictions and write confirmations. Treat this as a **platform-proposed integration option to evaluate**, not as tested availability, a BoxWeave component, an auth bypass, or evidence that the ephemeral Cloudflare Quick Tunnel used here was Secure MCP Tunnel.
- **Unanswered by Help AI:** Whether this exact block was a false positive or intentional; whether a future approval/authorization capability will exist; whether any particular ChatGPT plan, tunnel, tool annotation or gateway configuration eliminates the block. Do not describe the response as an OpenAI engineering team guarantee or infer that a user can override safety checks.

**Product decision / future Agent guidance:**

1. **Do not provide cross-layer orchestration in BoxWeave v1; do not impose an artificial ban on cross-layer operations either.** Each controller manages its *directly owned* resources and authenticates/authorizes locally. An independently authorized Agent, user, or ordinary in-guest MCP client may compose controllers via their standard interfaces when their client platform permits it.
2. **Separate protocol feasibility from ChatGPT client behavior.** This experiment proves nested local MCP **read** feasibility and direct L1 MCP **write** feasibility; it leaves indirect ChatGPT-mediated nested writes **pre-dispatch blocked**. The public MCP protocol and BoxWeave design should not be described as intrinsically forbidding nested writes.
3. **Keep three approvals distinct:** ChatGPT host-side safety/write confirmation, the target BoxWeave's MCP OAuth/Bearer role checks, and Incus Project/native rights. Successful authorization at one gate neither propagates to another gate nor obliges ChatGPT to dispatch an indirect action.
4. **No workaround by weakening controls:** Do not change tool descriptions to conceal mutation intent, insert privileged parent-child proxies, share Incus admin sockets, weaken anti-spoof, or move secrets into `box_exec` command text. A dedicated external integration remains a possible *separate* design, not a mandated v1 change.
5. **Classification/reporting:** Record `CHATGPT_PRE_DISPATCH_BLOCK` (descriptive test label, **not** a new BoxWeave API error), preserve the lack of outer/inner Operation records, and avoid blaming Incus or L1 auth. Retest independently if the client platform changes; no automatic replay of UNKNOWN side effects.

This support follow-up **does not change** the historical physical test receipts, the RC.1 AppArmor gap, the unverified quota/network claims, or the unsigned `HOST_GO`/S5 gates.

## 6. Historical AppArmor boundary: do not confuse RC.1 with RC.3

The separate [RC.1 physical nested AppArmor experiment](V1_RC_PHYSICAL_HOST_NESTED_APPARMOR_2026-11-13.md) used a prior L0 Incus 7.5.1 → L1/L2 7.6 topology. It **did** create and Exec an L2 through real authenticated v1rc MCP; it then installed Incus 7.6 in L2 and failed at `incus admin init --auto --storage-backend=dir` while loading the dnsmasq AppArmor policy (`apparmor_parser` Access denied). No L3 existed. The RC.3 L2's `enforce` profile and AppArmor stacking `yes` look relevant, but in this RC.3 run we **did not try** L2 Incus initialization, so the old failure is **neither revalidated nor resolved**. Host/kernel/daemon versions and policy differences need controlled testing. Never set `security.privileged=true`, globally turn off AppArmor, share L0's Incus socket or weaken anti-spoof just to force a green nested result.

## 7. Operational conclusions and remaining gates

- **Accepted in scope:** L0 RC.3 creates L1; L1 runs independent matching RC.3; L1 restricted Incus native Launch works; **direct OAuth-connected L1 BoxWeave Create, owner readback and guest Exec work**; nested Python MCP **read** works; native manual probe was reported deleted.
- **Not accepted:** OpenAI's indirect mutating tool dispatch; exact reason for its safety check; L2 full controller/L2→L3; live IPv4/IPv6 isolation/spoof/egress packet tests; root/Exec spool hard quotas; fault/rollback/UNKNOWN matrix on the physical host; independent verifier; unrestricted arbitrary-depth parity; `HOST_GO=PASS`.
- **No architectural change required now.** The supported end-to-end write path is a separately authenticated, **direct connection to that layer's** `/mcp`. Guest-local SDK/client access remains useful where allowed; a platform refusal should be classified separately from BoxWeave and native errors, without creating a privileged parent-to-child API.
- **Reproduction/reporting:** record the exact UTC timestamp, outer tool name, sanitized argv shape, original idempotency key (never blindly regenerate/replay an unknown mutation), any outer receipt, inner MCP/SQLite/native before/after counts, and sanitized request/connector logs. OpenAI Support explicitly asked for the time, app/connector, conversation/request ID and whether the MCP server saw the call (see support links above). Do not expose secrets, tunnel hostnames or raw credential files.
- **Cleanup/lifecycle:** temporary admin/viewer probe tokens were revoked; the manually launched L2 was separately deleted by the operator; the newly **BoxWeave-owned RC.3 L2 remains Running** as of its last readback. The L1 Quick Tunnel is test-only, ephemeral, and **not enabled for reboot**; a new URL requires fresh Origin configuration and OAuth review. Preserve L2 original Operation ID and never mutate other projects/foreign instances.
- **Follow-up tests, not implied approval:** independently inspect direct L1 L2 native expanded NIC/ACL config and real packet negatives; separately validate L2 Incus/AppArmor with proper isolation; verify quota/space according to actual driver and filesystem; exercise original-key Stop/Delete and independent final native/SQLite reconciliation; run formal postflight with an independent verifier. Record `UNSUPPORTED` or `BLOCKED` truthfully if the safe native environment cannot support deeper nesting.

**Related contracts:** [nesting design](../design/NESTING.md) · [RC.3 physical-host pilot](RC3_PHYSICAL_HOST_PILOT_2026-11-13.md) · [RC.1 AppArmor boundary](V1_RC_PHYSICAL_HOST_NESTED_APPARMOR_2026-11-13.md) · [bridgehead gate](BRIDGEHEAD_ACCEPTANCE.md) · [manual RC.3 installer and OAuth setup](../getting-started/MANUAL_INSTALL.md).

# Next Agent handoff — PRS blast radius / Root helper lifecycle (RC13)

**Status:** Design and exact real incident evidence committed; product/runtime changes **NOT implemented**. Continue the Draft PR branch `fix/rc13-peer-blast-radius-lifecycle-20261123` targeting `v1`; **do not merge, deploy, reset Host OAuth/tunnel, modify L0 root ACL/systemd or run global Incus operations without a new explicit Owner instruction**. The separate Draft [PR #177](https://github.com/vanishs/boxweave/pull/177) owns unified Doctor/Recover implementation, and its original guided-uninstall completion must not be disturbed.

## Owner updated acceptance and changed schedule (supersedes earlier prompt)

**FIRST READ:** [authoritative owner requirements and #178→#177 dependency](V1_PEER_MCP_AUTOSTART_NONDISRUPTIVE_OWNER_DECISION_2026-11-23.md). The following original handoff describes the **historical RC13** safety constraints, and earlier advice such as “do not restore `boot.autostart=false`” applies **only until a new safe replacement has been developed and independently proven**. It is **not** the final acceptance target.

The #178 Agent is now authorized to **design and test** the safe replacement, with these target behaviors: (1) new Box `boot.autostart=true` by default and preserve the user's chosen setting across Peer operations; (2) routine Peer grant/revoke through authenticated MCP without per-operation Host commands after trusted initial least-privilege installation; (3) no stop/restart, service loss or network disruption for **unrelated** Boxes. Choose a proven network-authorization architecture, not a speculative change to shared ACL/conntrack. If no proven non-disruptive path exists, fail closed and report blocked rather than claim full completion. Current physical L0 and historical PRS remain untouched unless separately authorized.

**Dependency:** #178 must finish with tested runtime source + exact-head CI/native proof + effective permissions/Operation schema handoff **before #177 starts implementing new unified Doctor/Recover features**. Do not tell #177 to compile any diagnostic status against old RC13-only `ADMITTED` manual approval, whole-Project STOP, retained false or old Root helper unit as a permanent expected state. Existing #177 uninstall recovery work is preserved.

**Updated next-Agent prompt (start here, rather than executing the original prompt verbatim):**

> 接手 #178。先阅读本分支的 `V1_PEER_MCP_AUTOSTART_NONDISRUPTIVE_OWNER_DECISION_2026-11-23.md`、RCA 和修复计划。Owner 的最终验收是：新 Box 默认自动启动，Peer 操作保留用户自启动设置；完成初始安全安装后，日常 Peer 操作与可证明安全的恢复都由 MCP 自动完成；授权 A→B 不得中断 C/D 的实例、应用和网络。先分析现有共享 ACL、全 Project STOP、Root helper 权限及替代架构，列出安全证明与兼容代价，然后在隔离 Box 中开发和跑真实 IPv4/IPv6/conntrack/ACK-loss、负向隔离、第三方长连接、重启/autostart、权限逃逸测试。不能为了不中断而出现短暂 fail-open，也不能为了安全继续把整个 Project STOP 当成新方案正常成功。候选代码验收后交付最终网络/Helper/Operation 语义给 #177。不要改物理 L0、未经批准合并、或声称未测项目 PASS。

## Copy-ready Chinese prompt

> 接手 BoxWeave RC13 Peer Root 生命周期与中断影响修复 Draft PR（分支 `fix/rc13-peer-blast-radius-lifecycle-20261123`，base `v1`）。先阅读本目录 [源码与真实环境 RCA](V1_RC13_PEER_ROOT_RCA_2026-11-23.md) 和 [完整修复/验收计划](V1_RC13_PEER_ROOT_REMEDIATION_PLAN_2026-11-23.md)，再阅读 [#177 的新增 Doctor 案例](https://github.com/vanishs/boxweave/blob/fix/rc12-rc13-guided-uninstall-review/docs/development/V1_RC13_NESTED_PEER_DOCTOR_CASES_2026-11-23.md)。
>
> **先检查证据而不是复现物理 Host:** 真实 RC13 `6c995687ae38b5cd0e23c464dee84f40a33a01ef` 已经通过了 L0 客户端→定向 Peer→L1 BoxWeave RC13 `0.0.0.0:18080` Forward→L2 `127.0.0.1:8000` HTTP，实际响应 `INNER_RC13_HTTP_OK`。未授权第三台 Box 超时，隔离有效。不要重复已经证明的全部物理 Host 测试，也不要把内层 functional PASS 当成新的候选安全签收。
>
> **待解决问题 1（有真实复现）：** root 批准 `box_peer_set` 时 `internal/native/peer_acl_isolation.go` 的全桥 barrier 故意将 **Project 下所有受管 Box** `boot.autostart=false`、逐台 stop，再由 `internal/install/native_peer_root_grant_apply.go` 校验 ACL/conntrack/nft 并 resume 原本运行者。第三台无关的 `http-check-1123` 临时 Python HTTP 监听在 stop 后消失，旧 Proxy Device 仍然 `CONFIRMED/native_verified=true`，字段含义正确（只代表设备）。当前 Agent 操作前得不到明确受影响范围、临时服务可能中断和原始 autostart 变化通知。第一轮优先加**只读且精确绑定 digest 的影响计划、审计/状态说明**，保持全部 Native fail-closed barrier 不变，写三台 Box、已绑定 Forward、Guest listener 前后对照的真实 native 试验。
>
> **待解决问题 2（源码证明风险，但尚未复现第二次授权失败）：** 原始 Peer `50b99fa7dc8de5d9f5317a87387cab6d` 先停留 ADMITTED 并锁住两台 Box+ACL，因为 Host `boxweave-native-peer-root.service` 不存在；Owner 完成专属 stage→plan SHA→unit-stage→enable→restricted execute 后变成 SUCCEEDED。`internal/install/native_peer_root_service.go` 使用一次性 Operation/SHA systemd 单元及 `O_CREATE|O_EXCL`，原单元事后仍在 Host 上。需要在**隔离 Box/native Host** 测试第 2 个原始 Peer grant/revoke 进来后如何受限，再设计仅 Host-local、原始完成与 Root journal 严格绑定的安全单元 retirement/rotation，不能随意删除 Root 证据或做常驻通用 root daemon。可与 #177 `recover` 单入口衔接，但不要自己发明第三套 Host repair 系统。
>
> **其他已调查归属：** `native_verified=true` 不保证 Guest HTTP listener（正确契约）；CLI doctor 进程显示 external policy disabled 但运行中 systemd/MCP 是 all（#177 已有方案）；Exec 原始 Operation 可 SUCCEEDED 但 Guest exit 124（执行结果要分开）；L1 guest apt HTTP 镜像不通而 HTTPS 正常（只作 Guest 环境问题，不允许 Doctor 自动 apt）；L1 unconfined/L2 enforce 不能直接证明全层安全；含下划线的 Incus 名称准入失败可提供明确校验提示。以上归到 #177 案例，勿重复做网络后端大改。
>
> **实施顺序（优先写代码，不要停留在又一份计划）：** ① 检查 PR HEAD/已通过 CI 和 Root PRS 安全证明（`internal/native/peer_acl_isolation.go`，`internal/install/native_peer_root_grant_{review,apply}.go`，`internal/install/native_peer_root_service.go`，`internal/control/forward_tools.go`），对每条副作用/权限/审计原始键建立映射。② 在可用的隔离开发 Box 安装独立 Incus 和 RC13 候选，在不依赖物理 L0 的情况下先重现 3 Box barrier 与临时 HTTP 丢失，再做最小可维护的 Plan 影响字段、只读状态/活动记录和测试。③ 测第二次新 Peer original 的 unit turnover；如果真实障碍成立，再提出/实现按 exact original/journal/root-reviewed SHA 约束的 Host-local safe turnover，但未获 Owner 批准不要执行危险实际清理。④ 通过精确 Native/SQLite/Root/Python HTTP 正负包、UNKNOWN、ACK-loss、autostart 三态、不同层 0.0.0.0 和既有 #177 兼容回归；已有通过的独立验收无需重复烧资源。⑤ 更新修复证据/commit/CI 到该 Draft PR，保留未测项与不确定证据，等待 Owner 签收。**不改变已安装 L0、不合并 v1、不发布 rc14**。
>
> **安全红线：** 不跳过全 Project STOP、不要自动把 `boot.autostart=false` 还原、不要擅自恢复 UNKNOWN、不要绕过原始 Host root 本地批准、不要编辑 SQLite/Kernel ACL 来让 CI 变绿；任何新 Root helper 必须仍为一次性原始签名范围。对未完成的物理 Host/重启/安全验收明确记录 NOT TESTED，而非 PASS。

## Minimum first response required from next Agent

- State exact Draft PR HEAD, base and source snapshot; identify any existing unmerged PR conflict with #177, and retain previous release/PRS evidence.
- Show the actual native barrier and one-use root lifecycle code chain, not only a general architectural guess.
- Say which one *small* implementation slice can be independently tested in a disposable Box without Host changes, then begin coding and testing it.
- Report every new original Operation and native ID and how Guest exit was verified; do not invent successful packet verification based on SQLite.

## Signoff checklist

- [ ] Correct full-project impact preflight, original digest and accepted externally-triggered Host approval still safe/compatible.
- [ ] All original/unknown/root journal failure barriers remain fail-closed; no uncontrollable root service, no blanket autostart restoration.
- [ ] Unrelated service disruption detected/presented; proxy device vs Guest HTTP proof separated.
- [ ] Terminal original and stale existing root unit safe retirement/next original tested, or explicitly deferred as NOT TESTED/Host-review.
- [ ] Exact real positive/negative Peer + Forward packets on isolated candidate after changes; wrong SHA/source/UUID refusal.
- [ ] Affected unit/native/SQL/Go tests and CI pass at **exact final PR HEAD**; no claim about untested Host reboot/security matrix.
- [ ] #177 diagnostic handoff remains independent; PR stays Draft until Owner approval.

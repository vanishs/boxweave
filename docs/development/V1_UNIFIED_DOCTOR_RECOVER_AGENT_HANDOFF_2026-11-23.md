# Next Agent handoff — implement unified `boxweave doctor` + `boxweave recover` in PR #177

**D4 candidate update (2026-11-23 PT):** [Reviewed recovery execution and exact Box evidence](V1_PR177_D4_REVIEWED_RECOVERY_2026-11-23.md). D2/D3 independent exact-head CI all four passed. D4 code, three focused Race rounds, full verify and fresh scoped installed positive recovery passed in Box. D4 independent CI is required before D5; legacy installed D4 and scoped partial apply refusal remain CI gates. Earlier staged-status notes below are historical, not instructions to redo accepted stages. Keep Draft; no v1 merge, release or physical L0 modification.

**D3 candidate update (2026-11-23):** [Read-only recovery plan and Box evidence](V1_PR177_D3_RECOVERY_PLAN_2026-11-23.md). D2 installed JSON assertion corrected; D2/D3 independent CI still required. D4 apply is not implemented.

> **D1 accepted, D2 diagnosis candidate:** [D2 implementation, Box evidence and CI gates](V1_PR177_D2_DIAGNOSIS_2026-11-23.md). D1 exact-head CI all four passed. D2 shared diagnosis, mode-qualified retirement/capacity, bounded activity/history, Native/Root/Forward/OAuth coverage and stable issue routes delivered; review D2 exact-head CI before D3. Routes are not repair authority.

> **D0 accepted, D1 foundation delivered for CI:** [D1 implementation and Box evidence](V1_PR177_D1_FOUNDATION_2026-11-23.md). Check exact D1 HEAD CI including new installed mode/authority markers before D2. Doctor now captures a bounded private locked SQLite epoch and never calls state.Open on source; this handle is not a Root witness. Do not repeat D0.

> **Mandatory D0 extended findings:** [D1-D4 investigation and additional Box evidence](V1_PR177_D0_EXTENDED_D1_D4_INVESTIGATION_2026-11-23.md). Reuse the source/authority and leaf matrix; fix proven retirement/operation-cause/navigation classification gaps in D2, separate Root proof from restricted recovery worker, bind fresh review evidence separately from idempotency, and do not certify a moving DB/WAL copy. D0 extension is characterization only; review exact-head CI before D1.

> **Current staged execution contract:** [D0-D5 plan](V1_PR177_D0_D5_DEVELOPMENT_PLAN_2026-11-23.md) and [D0 source/Box experiments](V1_PR177_D0_EXPERIMENTS_2026-11-23.md). The v1 merge is complete at e60de8f; all four baseline CI workflows passed. D0 characterizes readers only; new feature implementation starts with D1 after D0 CI. Every stage iterates and verifies inside a Box before submitting code and docs to CI. The cold SQLite WAL/SHM observation and immutable stale-read negative are mandatory D1 constraints.

**As of 2026-11-23. Owner-approved direction, design committed; feature code is NOT implemented.** Do not claim existing RC13 `doctor --full` or `recover --plan --issue` work. You must read [the full design and incident ledger](V1_UNIFIED_DOCTOR_RECOVER_DESIGN_2026-11-23.md) before coding.

## CURRENT FIRST ACTION — #178 merged, proceed after integrating v1

#178 was accepted and merged to v1 at `ba29c74292747242b211771c30b4d9d8f6f83ed9`. The prior implementation hold is superseded. **Read [mandatory post-#178 diagnostic contract](V1_DOCTOR_RECOVER_POST_PR178_CONTRACT_2026-11-23.md) first.** #177 is still Draft and diverged from v1; integrate latest v1 into its own branch, review conflicts preserving original guided-uninstall behavior and scoped fresh refusal guards, and audit actual integrated source before implementing. No production changes, #177 merge or release. Legacy RC13 incidents below are mode-specific historical regression fixtures, not generic modern diagnoses. Prioritize the new mandatory classification and regression matrix; do not recommend old manual Peer approval, full-project STOP, or treating explicit autostart=false as a defect in scoped mode.

## New RC13 real nested acceptance / incident regression (2026-11-23)

**Mandatory additional reading:** [RC13 nested Peer + Port Forward real incident matrix](V1_RC13_NESTED_PEER_DOCTOR_CASES_2026-11-23.md). It records an actual working L0 Client → L1 RC13 Forward → L2 Python HTTP chain and separate default-deny, allowed, and unapproved-third-party packet witnesses. **Do not reclassify this feature acceptance as a diagnostic-system implementation.**

Prioritize the newly observed Doctor/Recover cases alongside the existing RC12/RC13 cases: (a) Peer original waiting in `ADMITTED` with root helper `not-found`, claims locking both endpoints plus ACL, and `operation_cancel` blocked; (b) one-use Host PRS approval, exact original/digest and no MCP root bypass; (c) whole-Project stop/autostart=false barrier and volatile guest-process loss; (d) native Forward device confirmed despite absent Guest listener; (e) Exec Operation `SUCCEEDED` with `guest_exit_code=124` represents a **failed Guest command**, not a successful install; (f) per-layer host bind `0.0.0.0` vs L0 reachability and Peer isolation; (g) effective systemd/MCP policy `all` vs standalone CLI `disabled`; (h) nested L1 `unconfined` and L2 AppArmor `enforce` are observations, not full multi-layer security proof; (i) signed apt source / missing `gnupg` are Guest environment facts, not permission to run APT from Host Doctor; (j) original-specific PRS unit lifecycle and absent automatic status. **Root behavior redesign has been merged in #178; these historical cases apply only to proven legacy mode. #177 owns readonly diagnosis and guarded recovery navigation.**

## Copy-ready agent instruction (中文)

> 接手 BoxWeave [PR #177](https://github.com/vanishs/boxweave/pull/177)，只在既有 Draft 分支 `fix/rc12-rc13-guided-uninstall-review` 继续开发。Owner 已同意统一诊断/修复设计：**一个 binary、两个公开 CLI 入口 `boxweave doctor`（只读诊断）和 `boxweave recover`（经过独立审核的修复）**，不要新增第三套恢复引擎或 MCP Host root 工具。
>
> **第一件事**：先读取新的 post-#178 contract，合并最新 v1 到 #177 分支并人工解决冲突，保护双方代码；再读取本文件和 `docs/development/V1_UNIFIED_DOCTOR_RECOVER_DESIGN_2026-11-23.md`，复查 PR 当前 HEAD、与 v1 的 diff、#176 的物理 Host 迁移证据、已验收 CI。现有 #177 解决了 RC12 `STARTED` 卸载审计和 Native rollback 的恢复，实际物理 Host 已通过完整卸载 + 9/9 独立 postflight，**不要重复、重写或破坏这些证据**。已发布 RC13 安装和 OAuth/ChatGPT MCP 连接成功，当前 L0 正在运行，**未经 Owner 授权严禁修改 L0、Root、Host、Cloudflare、v1 分支或合并 PR**。
>
> 在动代码之前先做**完整 CLI→SQLite/Native/Root/installer/systemd/OAuth/MCP 数据源贯穿矩阵**，逐一标注现有能力、调用身份、证据/权限/副作用、是否可复用；优先关注：`cmd/boxweave/main.go` 的 `status`/`doctor` 实际复用 `Service.Status`；`state.Open()` 会 `MkdirAll`、以读写模式打开甚至创建 DB、执行 `PRAGMA journal_mode=WAL`，所以不能在新 Doctor 中直接当做纯只读数据库入口。需要专用无副作用 RO 查询入口，不触发迁移、恢复、WAL 修改或 Native 写入。Native 与 SQLite 交叉读取不是原子快照，必须如实显示 `UNVERIFIED`/coverage。
>
> 先实现兼容旧行为的 `doctor --full --json`（**这是拟议新参数**），统一输出：安装 manifest/二进制/unit、服务实际有效启动策略、Native/ACL/Project、SQLite schema/Operation/UNKNOWN、网络绑定/孤儿/地址保留、Root helper 和缺失依赖、OAuth 与 Tunnel 基础状态、升级/卸载中断。报告必须包含版本化结构、可定位 issue ID、证据来源与覆盖率、严重度、已解决与待处理区别、能否通过 MCP 修复、Host-only 原因和下一步；默认绝不能泄露 password/token/hash、其它 Project/Host 敏感资源。服务不运行或 Root 证据不可达时返回 `PARTIAL/UNVERIFIED`，而非误报一切健康。
>
> 特别把这次实际遇到的问题做成回归：① RC12 非自有 `restricted.devices.proxy=allow` 导致 Native rollback 漂移拒绝；② 原始卸载 `STARTED` 审计保留原始 archive/review，#177 独立 `--resume-plan` + 终端批准 `--resume` 已成功，不可重做第二套；③ RC13 需要 `/usr/sbin/conntrack`，当前 installer 在人工 INSTALL 确认后才 `install-stage` 报缺包，修复应在只读 preflight 提前报告，禁止自动 apt；④ 原终端未准确输入 `INSTALL user-1001` 是安全拒绝而非损坏；⑤ `oauth-setup` 后 `Origin` 仍空，所以 `Enabled:false` 属于配置未完成；⑥ 本地 CLI doctor 的 Forward policy=`disabled`，真实 systemd daemon/MCP 的 policy=`all` 是环境不同，不能用 CLI 进程环境冒充运行服务配置；⑦ 逻辑上限 1024 而实际 /24 仅观察到 253 IPv4 上限，仍有配额、数据包、PRS 未独立验证；⑧ native_network 已有完整 Project inventory 与 0 orphan 证据但跨源非原子；⑨ ChatGPT MCP 只读成功并不证明写工具/独立可信审批。
>
> Doctor 数据源稳定后，再增加 `recover --plan --issue <ID>`（**拟议参数**），复用现有 `operation_recover`、`recover -action=diagnose/reconcile`、专门 Create/Delete/GC、Native prepare/rollback、root PRS、#177 uninstall resume；只有完全独立证据、准确 Project/Operation/UUID/Incarnation、不可重复的 original 和明确 Host 本地审批/重验摘要，才允许受支持 leaf 执行 `apply`。对未实现、不安全或证据缺失的类型必须报 `BLOCKED`，而不是泛化 `--fix`。UNKNOWN、冲突与 Native ACK 丢失绝不能盲重试、改 SQL 或跨 Project 修复。正常 Box 业务级恢复优先 MCP；Host CLI 只解决真正的 Host/安装/Root 问题。
>
> 迭代顺序：**A. 完整矩阵与纯只读 Doctor；B. 结构化问题、证据及现有 MCP 恢复导航；C. 只读 Recover plan；D. 有限范围明确批准的 Recover apply；E. 在隔离 Box 内做真实 CLI/SQLite/Incus 负例与故障恢复测试；F. 推送 PR #177 分支并跑 GitHub CI**。写好单元/并发/拒绝/缺省权限/原有 CLI 兼容测试；已通过部分不要重复做全量压力实测。工作中遇到具体环境限制，应把失败归因和证据留下，然后继续其它可测部分。**不合并 v1、不改已安装 L0、不重装 RC13**。将阶段结果、commit、测试/CI 链接和剩余缺口记录 PR #177 并停在 Draft 等待 Owner 决策。

## Minimum first-turn output expected from the Agent

Do not ask the owner to repeat history. Begin by inspecting exact PR head and source, report:

1. Current `HEAD`, diff scope, prior already-accepted uninstall CI/Host evidence.
2. One concise source/authority/side-effect matrix: Doctor, Activity, Recover, Native, installer/uninstaller, service-effective Forward, OAuth, database open.
3. First safe implementation slice and precise test target.

Then immediately implement/test the first safe slice on the PR branch where tools permit; don't stop at writing another plan. No new Host approval or production deployment is implied.

## Non-negotiable acceptance and stop conditions

- `doctor` performs zero state mutation even if DB path is absent/corrupt or concurrent with service WAL; no daemon worker startup recovery or schema initialization, no unintended permission escalation.
- Existing `status`, `doctor` basic mode, `recover` old flags, 22 MCP tools and #177 uninstall behavior stay backward-compatible.
- Findings distinguish `CONFIRMED` vs `UNVERIFIED`, bounded vs full scan, resolved history vs current issue, MCP business recovery vs Host operator repair; secrets and foreign Project state never leak.
- A private Host installer failure/Root journal is not silently considered fixed through MCP. Approval string, plan fingerprint, idempotency and original native/SQLite proof remain enforced.
- No claim of packet/PRS/quota PASS without independent real witness; no 1024 actual native instances required for this scoped feature.
- No automatic branch merge, release, public Tunnel modification, or physical L0 install. If later Host tests are needed, provide a minimum exact, read-only command and explicit scope for separate Owner approval.

## Quick source links

- [PR #177](https://github.com/vanishs/boxweave/pull/177) · [Host migration Issue #176](https://github.com/vanishs/boxweave/issues/176)
- [Design and evidence](V1_UNIFIED_DOCTOR_RECOVER_DESIGN_2026-11-23.md)
- [Existing uninstall contract](V1_UNINSTALL_SH_OPERATOR_CONTRACT_2026-11-19.md)
- [N3 scoped acceptance](V1_N3_SCOPED_ACCEPTANCE_2026-11-22.md)
- [S5A2 recovery](S5A2_RECOVERY_2026-11-20.md)
- [Product boundary](../decisions/PRODUCT_BOUNDARY.md)

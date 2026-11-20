# Next Agent prompt — start S4-A in existing PR #153 (2026-11-16)

**Checkpoint 2026-11-16:** This is the historical *start S4-A* implementation prompt. S4-A code `3407e2a2` has [independent scoped PASS](../verification/S4_A_FORMAL_ACCEPTANCE_2026-11-16.md) in existing PR #153. Refresh the latest PR HEAD and its exact-head checks/final verdict; do not duplicate implementation, open another PR or start S4-B without the owner initiating it.

Copy the following to the next development Agent **as an implementation task**, not merely a planning request.

---

你接手 **BoxWeave v1 的 S4 开发**。S3 已正式限定范围签收并合并到受保护的 `v1`（合并 SHA：`4a45a80c8de2fb36cde851a10e0f9c80f2f9b300`）。**只能沿用现有 Draft [PR #153](https://github.com/vanishs/boxweave/pull/153)**，开发分支 `feat/v1-s4-runtime-integration-20261116`，不要另开 PR，不要从已经结束的 S3 分支继续，也不要提前合并或发布 Release。

**本轮完整执行 S4-A（P0 基础安全阶段）：一次性完成相关代码改造、测试、修复与限定范围独立验收，然后汇报。** S4-B/S4-C 后续沿用这个 PR，分别在我发出“开始 S4-B / 开始 S4-C”时执行，避免长上下文和不必要的重复测试。开始时读取 `AGENTS.md`、`docs/development/PHASE_EXECUTION_PROTOCOL.md`、`docs/development/S4_S5_EXECUTION_REBALANCE_2026-11-16.md`、`docs/development/S4_EXECUTION_KICKOFF_2026-11-16.md` 和 `docs/development/phases/04_NETWORK_NESTING.md`。查看 PR 最新 HEAD 和实际代码、已安装 RC7/S3 证据，按需读取设计源码，不要把历史研究一次性全读进上下文。

**需要真实完成的 S4-A 工作：**

1. **#148 稳定 OAuth grant actor：** 用稳定授权主体标识与每次刷新变化的 access-token credential ID 分离；同步检查 `TokenVerifier`、Operation/Claim、SQLite admission/replay/actor/token、viewer read scope、manual Bearer、refresh、revoke/expiry、原始 idempotency key。验证同一 OAuth grant 的 A→refresh B，在原始操作 `SUCCEEDED` 之后同 key/同 intent 返回**相同原始 Operation**，有 `duplicate=true` 和**恰好一次 Guest 原生效果**；独立 grant、viewer、revoked/expired access、manual token 全部负向正确。不能让刷新绕过一次性效果或读取其他用户的记录。
2. **X-02/X-03 Create/Delete 原生与 SQLite 一致性：** 正确持久化原始 Box UUID 注册、Delete tombstone、幂等 key、终态与失败分类；设计可复用的 dependent-resource cleanup seam 给后续 Forward/Peer，验证 SIGKILL、丢失 ACK、并发、回滚、原生结果不确定时 `UNKNOWN` 且绝不盲重试。严格区分 BoxWeave durable `SUCCEEDED` 和 Guest Exec 非零退出。
3. **提前补一个小型“原始 Incus Operation 终态”独立证据接口（仅测试工具，不是公共 MCP Tool）：** 新鲜隔离 native 原始操作执行时，由独立 CLI/SDK 观察者尽早读取 Incus daemon 终态 JSON/状态（若仍保留），关联原始 BW Operation ID、Incus native UUID、Box UUID/Project、Guest exit、SQLite 和文件/网络结果。拿不到就明确 `NOT_CAPTURED` 或 `EXPIRED`，绝不凭 native ID 编造终态，不重放历史 UNKNOWN，不借 Guest 获取 parent Incus socket。#151 完整历史间歇故障归因仍属 S5。
4. **验收：** 先跑精准单测/质量与需要的 race，再在**获准的独立一次性真实 Incus 7.6 / 非 root BoxWeave HTTP MCP 环境**中跑 #148 正反例、X-02/X-03 故障例、原始回执+独立 native 读回；回归 S3 三文件工具和 S2 Operation 基础路径，最后按当前同一提交运行必要 GitHub Actions。失败时保留原始操作/证据，修复后复验；无法获取的宿主权限要标 `BLOCKED` 而不是降低安全性绕过。

**本轮不要做：** 不提前实现 4 个网络 MCP Tool（留 S4-B），不把 #150 的强嵌套内核难题作为 S4-A 阻塞项，不把 #151 的长时间重复测试前移，不开启自动无人值守跨 Token 刷新写操作；不改物理 L0、原来的 L1 全局 Incus/Scriptlet/AppArmor 配置、不释放或重试任何历史 UNKNOWN、不使用特权/无约束容器做虚假的安全 PASS，不改 RC7 Release 标签，不开第二个 PR。

**完成输出：** 在同一 PR #153 提交实际代码、单测、native/安装版 MCP 结果、独立原始结果报告和交接文档；记录准确 Git commit、原始 BW/native Operation ID、Guest exit、必要 SHA 和 Actions 链接。给我一句结论 `S4_A_PASS` 或 `S4_A_PARTIAL/BLOCKED`，列尚未解决的具体风险，以及 S4-B 下一步的关键依赖。若没有 blocker，保持 Draft 供后续 S4-B/C 开发，**不要将 PR 合并进 v1，除非完整 S4 已签收且我要求合并**。

**总体 S4/S5 顺序不可忘：** S4-B：四个 Network Tools + 共享 ACL/Scriptlet + native TCP/UDP Forward/Peer + 依赖安全删除；S4-C：支持的 L1/L2 与 20 Tool schema、真实网络负向、#150 能力/UNSUPPORTED 判定和 S4 scoped acceptance。S5：#151 深度故障归因（含原生终态）、完整双 Agent、20/20 MCP+CLI、真实物理 LAN/公网 IPv6、强嵌套安全真实性、独立 Verifier 与正式 v1 Release。以上不是降低最终验收标准。

---

# BoxWeave RC8 — 真实 ChatGPT 跨 Agent 活动找回、证据链与输出生命周期观察（2026-11-19 PT）

> **性质：已连接 L0 RC8 的观察性证据与后续设计讨论输入，不是源码修复签收、Agent 心跳证明、RC 发布批准或 S5 多 Agent 接手签收。**
>
> **运行目标**：`boxweave L0 rc8`，`host_status.version=1.0.0-rc.8-candidate`，`commit=cf577ee67e3b7aefd07957830774a11c6b6c1ec1`，`schema_version=5`，Incus 7.6，Project `user-1001`。观察窗口为 2026-11-19 19:05–19:32 PT（服务器事件记录的 UTC 日期为 2026-11-20）。
>
> **范围与注意**：下述证据来自同一已授权 Project 的 SQLite 快照与原始 MCP 操作。原会话中的另一个 Agent 的 ChatGPT 网页/UI 及私有推理不可见。首轮排查曾由观察者在测试 Box 内执行两次**只读取证**的 `box_exec`（无文件修改），`agent_ref=chatgpt-observer`；**本次 RC8 新特性加测全部采用只读 MCP 接口**，不增加新 Box、不触碰原任务。不得将两个 Agent 的操作混为一谈。

## 1. 重要结论

1. 当原 ChatGPT 页面疑似停在“检查 Git 克隆状态”时，另一个 Agent 的后续操作仍出现在持久日志中：以 `agent_ref=chatgpt-dev`、`work_ref=rc8-quota-exec-fix-20261119` 可找回创建、6 次 Exec、停止、删除共 **9 条原始 Operation**；19:23 之后有多次新写入。这证明**后台曾继续调用 MCP**，但**不能单独诊断浏览器断线/网页展示故障，也不能建立实时 Agent 心跳**。
2. 已删除 Box `ec7533c7-2051-4998-9245-40dbf33ee2c8` 可通过 `box_activity_get(view=overview,box_id=...)` 找回 **11 条历史 Operation**（含 2 条观察者只读 Exec）；`box_list=[]` 与其删除记录吻合。删除并未清空 Operation/Events/审计历史。
3. `box_activity_get(view=evidence,operation_id=...)` 可恢复 `ADMITTED → DISPATCHING → VERIFYING → SUCCEEDED` 的原始事件顺序、原始/native ID、关联标签、资源键、Exec 退出码、已删除 Box 状态及证据局限；旧 `UNKNOWN` 的人工复核与资源声明释放可追溯。
4. **发现输出生命周期缺口**：已删除 Box 的 `operation_get` 和 `evidence` 仍成功，但 `operation_output_get(stdout/stderr)` 对原 Exec 返回 `operation not found`，而不是结构化的 `unavailable/expired`；这是实测，不能在未验证删除前后的原生输出清理策略时声称输出应当永久保留。
5. `issues` 在观察中返回空集合，始终 `attention_complete=false`。与此同时 `operation_list(state=UNKNOWN)` 仍能找到一条**已人工释放资源声明、状态保留 UNKNOWN** 的历史操作。这个差异符合当前“有效阻塞视图 vs 历史记录”的源码意图，但用户可见语义不够直观；**空 issues 绝不等于无历史 UNKNOWN 或全宿主无问题**。

## 2. 真实任务轨迹（同一 work_ref）

| PT 时间 | 原始 Operation ID | 事件 | 观测结果 |
| --- | --- | --- | --- |
| 19:05:19 | `9dbffed01a3269988c59ad5115694672` | `box_create` | `SUCCEEDED` |
| 19:05–19:19 | `2952c14163b8de0e2302fcea87735223`, `2de679b8904b05db1d26c07858ef8464`, `bdf2c5604a1c264bda388a4e74c5301a` | 连续 Exec | 各为 `SUCCEEDED`；19:17 启动的一次 Go 测试于 19:19 完成，原始 stdout 报告 native/control 包测试结果 |
| 19:23:18 | `7d17eebbfb8359653404442ff6ddb02a` | 新 Exec | Operation `SUCCEEDED`，但**Guest exit_code=8, command_success=false**；可见 SOURCE_HEAD 与两个测试文件名 |
| 19:24:16 | `6a92866ad193e690a3ca8a532cb87f93` | 新 Exec | 初始观察 `VERIFYING`，后在活动记录变成 `SUCCEEDED` |
| 19:28:15–19:29:00 | `6444cb960a992974e24cf50639114248` | 新 Exec | `SUCCEEDED`；证据视图显示 `exit_code=0` |
| 19:29:39 | `9a108640afbbadd7da837628bd6362e0` | `box_stop` | `SUCCEEDED` |
| 19:29:51 | `ef3afff0bcf737914ab867489b483395` | `box_delete` | `SUCCEEDED`，evidence `owned_box_state=DELETED` |

早期观察者的只读 Guest 命令曾证实 `/root/boxweave` 是有效工作树、`origin=https://github.com/vanishs/boxweave.git`、`HEAD=400b0ba`、`/tmp/clone.log` 包含 `Cloning into ...`、没有存活的 `git clone` 进程。**这是当时快照而非最终源 SHA**；Box 删除后不能通过历史 MCP 原样恢复 Git dirty tree、Agent 的计划或未持久化工作。

## 3. RC8 只读能力实测矩阵

| 能力/接口 | 观察 | 信任边界 |
| --- | --- | --- |
| `host_status` | 返回已安装 commit、v5、Incus、Project 和存储警告 | `selected_storage_quota_verified=false`，物理 quota 仍未签收 |
| `operation_list(work_ref=...)` | 原始 9 条操作，含 Exec 非零退出的操作 ID | 客户端标签不是身份凭据或分布式任务锁 |
| `operation_list(agent_ref=...)` | 找回相同 Agent 归属的原始操作 | 不等于 Agent 当前在线 |
| `box_activity_get(view=overview,box_id=已删除UUID)` | 仍可查询已删 Box 的 11 条关联操作 | 查询范围只涵盖当前鉴权 Project 内持久记录 |
| `box_activity_get(view=evidence,operation_id=...)` | 有序 events、资源声明、Exec exit、review 和限制标志 | 标志含 `NO_NATIVE_POSTSTATE`、`ROOT_JOURNAL_PRIVATE`、`READ_CALLS_NOT_JOURNALED` |
| `box_activity_get(view=overview,limit=3,cursor=...)` | 签名游标连续两页，页间未重复 | 分页不是跨调用同一原子快照 |
| `operation_list(state=UNKNOWN/FAILED)` | 找到 1 条 UNKNOWN 和 4 条 FAILED 历史记录 | 这些不是全部“当前仍在阻塞”的项目 |
| `box_activity_get(view=issues)` | 返回无 issue 明细、`attention_complete=false` | 不能推导“全局安全/完整检查通过” |
| `operation_get` | 删除 Box 后仍能找回原始 Exec 记录 | Operation 终态不等于 Guest 命令 exit 0 |
| `operation_output_get` | 删除 Box 后 stdout/stderr 均报 `operation not found` | 输出恢复能力弱于原始元数据和事件找回 |

注意：`operation_list(state=RUNNING)` 为空，只说明没有这类**持久 Operation 状态**；不代表不存在 Guest 内独立进程或其他未持久化 Agent 活动。

## 4. UNKNOWN 与资源释放证据

旧操作 `8c498b1343c54ef98952618308756d73`，`work_ref=rc8-host-storage-quota-20261119`：

- 状态事件：`ADMITTED → DISPATCHING → VERIFYING → UNKNOWN`；分类 `POSTCONDITION_UNVERIFIED`。
- `evidence.reviews` 记录 `CHALLENGE_ISSUED` 和 `RELEASED`；事件中有 `CLAIM_RELEASED_LOCAL_REVIEW_NO_REPLAY`。
- `operation_get` 仍为 `UNKNOWN`，`next_action=HOST_LOCAL_REVIEW`，并未假造 `SUCCEEDED` 或自动重放。
- `box_activity_get(view=issues)` 未列出该已经 RELEASED 的普通 Exec：**“状态未知”与“仍持有准入资源锁”应分别呈现**。对 Root Peer / 网络权限类 UNKNOWN 的隔离规则仍需单独成立，不能复用普通 Exec 的释放判断。

## 5. 待讨论问题 A：删除 Box 后的 Exec 输出

**实测：** 原始 Exec `6444cb960a992974e24cf50639114248` 的 `operation_get` 仍有 `state=SUCCEEDED`、`native_operation_id`；`evidence` 有 `exit_code=0`、`owned_box_state=DELETED`、`gc=[]`。但删除 Box 后，`operation_output_get(stdout/stderr)` 均返回 `operation not found`。

**源码解释（PR #153 分支所见）：** `internal/control/operation_output_cancel.go` 在读取 `ExecOutput` 元数据、检查 stream GC=EXPIRED 之后，继续调用 `Store.OwnedBox(...)` 与 `Native.GetExactBox(...)`。Box 已删除时，仍保留的 Operation/Exec 元数据无法进入结构化 `availability` 分支。因此最少有一条**已删除实例的输出可用性错误分类路径**；至于原生字节是否还在、应当保留多少天，需要独立确认。设计文档 `docs/design/TRACKING_CONTEXT.md` 规定输出应有有界保留、到期/未捕获/缺失状态；`scripts/s2_exec_gc_probe.go` 已有 7 日 TTL 的独立测试场景，但不能将其自动等同于 Box 删除后的物理保留保证。

**决策候选（尚未批准实现）：**
- **A1 最小安全修复（推荐先讨论）**：保留原始 Operation 与 exit/truncated 元数据。若 Box 已删除且没有可验证的输出归档，`operation_output_get` 返回授权范围内的 `availability=unavailable` 与明确原因（如 `BOX_DELETED_OUTPUT_NOT_RETAINED`），并继续返回已知 exit_code/command_success；不可把“删除导致不可读”谎报为“7 日 TTL 已到期”。
- **A2 更强找回**：经独立宿主 quota / spool 上限与秘密防护评估后，为每次 Exec 保存有界的、按权限与 TTL 管理的持久输出副本，使其不依赖 Box 的原生生命周期；每个流单独标记截断、GC 状态、取证和删除。此方案扩张存储面，**当前 host_status 配额未验证时不可直接启用**。
- 不能把原始 stdout/stderr 全文放进 Activity 或结构化日志；不能为了保留输出阻断安全删除或悄悄修改主机存储配额。

## 6. 待讨论问题 B：issues 的“活跃阻塞 vs 历史异常 vs 检索完整性”

**源码解释：** `internal/state/activity_query.go::readActivityIssues` 汇总 SQLite 可见的未终态/UNKNOWN 原始 Operation、特殊 Peer barrier、Exec GC blocker、Peer/Address fence/Forward 的有效依赖；普通已 `recovery_audit.RELEASED` 的历史 UNKNOWN 会从“当前有效阻塞”中排除；`AttentionComplete` 固定 `false`，因不会证明原生网络/跨页变化/未入账副作用。历史 `FAILED` 与已释放 UNKNOWN 仍可从 `operation_list` 或 `evidence` 找回。因此 **issues 为空并不自动是缺陷，问题是可读性与提示下一步不足**。

**建议的最低成本语义（尚未批准实现）：**
1. 默认 `issues` 只显示**需要处理的安全阻塞与依赖**，并给每条标注 `kind=active_operation | network_dependency | exec_gc_blocker`、`actionability=host_review | read_only_followup | none` 等有限枚举；保留当前 SQL 授权/索引/有界分页。
2. 同一响应增加不含隐藏资源的**说明性 summary**，明确 `issues_count` 仅是“本页”或“此查询可见”，不可伪装为全 Project 全量计数；历史已释放/已失败事项通过 `operation_list(state=...) → evidence` 查看。若需要在 issues 内列出，须另设 `scope=active|history` 并保持默认 active 不变，避免混在一起。
3. 把 `coverage` 由单个 `BOUNDED` 细分为易理解的边界信息，如 `source=SQLITE_SNAPSHOT`、`sqlite_scope=current_project`、`native_live_probe=false`、`attention_complete=false`、`page_truncated` 和 `next_cursor`。没有证明的维度必须为 unknown/false，不得展示为全局健康绿灯。
4. 空结果应显式输出 `issues=[]` 与说明 `no_current_sqlite_visible_blockers_in_this_page`，并指出“可能仍有历史 FAILED/UNKNOWN、无原生实时核验”。不要把空数组省略造成“接口未返回”歧义。

## 7. 建议后续验收（仅契约，不是当前已完成）

- 对已停止但未删除 / 删除后 / GC=EXPIRED / native stream 缺失，逐个测试 `operation_output_get` 的 `availability`、`unavailable_reason`、exit_code/command_success 与保密授权；确认不会假造保留字节。
- 同一历史 UNKNOWN 的 `RELEASED` 前后，比较 `issues`、`operation_list` 和 `evidence`；另测 Root Peer barrier 不因普通 release 被误放行。
- 分页、跨 Actor 同 Layer/跨 Layer 拒绝、Token 撤销与原生状态不可见，保证“未发现当前 SQLite 阻塞”绝不变成“系统无问题”。
- 验收实际安装的 `commit` 与 PR 源码分别固定；不得将 PR #153 后续修复、Box 内测试和当前安装版混成同一已部署行为。

## 8. 上下文与来源

- [已有 RC8 L0 实机 18 工具验收](V1_RC8_CHATGPT_L0_18_TOOL_ACCEPTANCE_2026-11-19.md)。
- [公开 Activity P0 实现/契约](../development/V1_ACTIVITY_PUBLIC_MCP_P0_IMPLEMENTATION_2026-11-19.md)。
- [MCP Agent UX 与 Activity 原始设计](../development/V1_MCP_AGENT_UX_AND_ACTIVITY_DESIGN_2026-11-18.md)。
- [Tracking/Operation 与输出保留契约](https://github.com/vanishs/boxweave/blob/06d27434b11997e9254822c122483c9dacba12a3/docs/design/TRACKING_CONTEXT.md)。
- 本轮由已连接 `boxweave L0 rc8` 的 `host_status`、`box_activity_get` 三视图、`operation_list`、`operation_get`、`operation_output_get`、`box_list` 实测；原始 Operation ID 如上，未复制任何机密或 stdout 全文到文档。

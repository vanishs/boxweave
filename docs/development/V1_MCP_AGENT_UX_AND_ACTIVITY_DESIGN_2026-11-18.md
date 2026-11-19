# BoxWeave v1 — MCP Agent 使用体验与 Activity/Evidence 概览设计（2026-11-18）

**状态：OWNER-ACCEPTED DESIGN / NOT IMPLEMENTED。** 本文记录用户认可的开发方向和独立源码评估，不表示相关 Tool、Server Instructions 或客户端实验已经完成。当前开发限定为 [Draft PR #153](https://github.com/vanishs/boxweave/pull/153)，核查起点 `cd3b2b95b857fe3dcc055090b2d8c7dc1a635a40`；实施时必须重新读取最新远程 HEAD。未经另行批准，不合并、不发布、不新增公开 MCP Tool、不变更当前 S4-B 验收范围。

## 1. 目标、范围与实施顺序

- **A — 近期：MCP 使用说明优化。** 让 Agent 在正常操作时先看懂 Tool 的实际职责；收到超时、丢失回复或中断后，知道找回原始 Operation、查看证据并避免重复执行。
- **B — 独立后续：`box_activity_get`。** 用一条只读请求概览授权范围内的持久化 Operation、异常和相关资源，并索引现有原始证据，减少接手 Agent 的多次分散查询。
- 两者分别评估、测试、授权、实现和签收。A 不依赖 B；B **不是** Agent 任务数据库、工作流引擎、持续上报系统、进度百分比或完整 MCP 调用轨迹。
- S4-B3–B6 继续按原独立验收推进；S4-C 既定 `box_get(detail=true)` 和 `box_wait_ready` 仍优先。B 默认放在 v1 后续单独阶段；若要前移，需用户批准第 22 个公开 Tool 和相应验收范围。

## 2. A — MCP Description 与 Server Instructions

### 2.1 设计原则

1. **功能优先**：每个 Tool Description 首先回答“能做什么、适合什么时候调用”，再描述工具自身必须知道的限制；不要让普通 `box_get` 成为中断恢复指南。
2. **共同规则集中**：服务级 `Server Instructions` 放管理范围、持久化异步 Operation、幂等、找回、UNKNOWN、安全/授权与“Operation 不是任务完成”的共性规则。
3. **轻量防重试尾注**：保留 `addStrictTool` 统一追加机制，仅对当前 9 个变更类 Tool 追加一句简短提示，不能替代其原有核心功能描述。
4. **行为事实优先**：Server Instructions 被协议返回 ≠ ChatGPT 客户端已将其提供给模型 ≠ Agent 实际遵循；必须分别检验。
5. **不改行为或契约**：本阶段不变更工具数目、输入/输出 Schema、权限、MCP 路由、Operation journal、native worker 或安全边界。

### 2.2 Server Instructions（拟议英文核心；先验证实际 SDK 与客户端）

下面核心段落约 491 个英文字符，设计为即使客户端仅使用前约 512 字符，也能独立表达重要规则：

> BoxWeave manages only registered Boxes in its configured Incus Project. Mutations are durable asynchronous Operations; a receipt is not completion. Use a distinct idempotency_key per logical mutation, and reuse the same key for exact retries. After interruption, use operation_list and operation_get to check originals before any new effect. UNKNOWN or missing records do not justify new-key retries. Operation records track admitted actions and observed outcomes, not Agent task completion.

中文含义：仅管理指定 Incus Project 的已登记 Box；变更返回持久化的异步 Operation，凭据不是完成证明；不同逻辑变更使用不同幂等键，仅完全相同意图的重试复用原 Key；中断后先查原始 Operation；UNKNOWN 或查无记录不能支持换新 Key 盲目重试；Operation 只追踪已受理动作和已观测结果，不等于 Agent 整体任务完成。

可置于后续段落的补充原则：

> work_ref and agent_ref are untrusted correlation labels, not authorization or reliable Agent identities. Use operation_output_get to inspect guest Exec exits; SUCCEEDED means the BoxWeave operation finalized, not necessarily that the guest command exited zero. Use box_get to inspect observed native Box state; Running does not prove guest readiness. BoxWeave does not preserve Agent plans, Git checkpoints, conversation state, or activity outside its own durable Operations.

中文含义：工作/Agent 标识不可作为权限或可信身份；Exec 需要检查退出码/command_success；Running 不等于 Guest 就绪；服务不存储 Agent 计划、Git Checkpoint、会话内容或其他工具的活动。

实施前核实 pinned `github.com/modelcontextprotocol/go-sdk v1.8.0` 的 `ServerOptions.Instructions` 返回路径、当前协商协议版本，以及实际 ChatGPT MCP 连接如何发现/使用它；必要时在独立环境进行真实客户端对照实验，不将“初始化/发现包包含字段”视为 Agent 已学习。重要的禁止重试原则仍在 Tool 本体保留短尾注作为客户端兼容性防线。

### 2.3 统一尾注和现有 Tool 的定向修改

当前提交 `cd3b2b95` 的 `mutationContinuityHint` 约 294 英文字符，会在 `addStrictTool` 为九个变更工具统一追加，重复成本偏高。建议缩为：

> One key per logical mutation; reuse only for exact retries. After an uncertain response, inspect the original Operation first.

`idempotency_key` 的逻辑：一个逻辑变更一个独立 Key；同一意图重试沿用原 Key。源码的唯一范围为 `(layer, actor_id, key_digest)`，相同 Key 与不相同的 intent 冲突，不能把 unique 理解为每次 HTTP 请求换 Key。

| Tool | 定向建议 |
|---|---|
| `box_get` | **恢复最初原文**：`Get exact locally owned Box UUID and independently observed native state; native running is not guest readiness.`；核心不是恢复流程 |
| `operation_list` | 功能/权限/过滤与有限分页优先，说明 `work_ref`、action、时间可帮助找原始操作；空结果不是“未执行”的证据 |
| `operation_get` | 突出原始状态、`next_action` 与有界 `wait_ms`；明确等待不会取消或重派；UNKNOWN 不得新 Key 重试 |
| `operation_output_get` | 突出 `operation_id`、`next_offset`、每页最多 65536 字节、`exit_code` 与 `command_success`；输出不可用/截断不得推断成功 |
| `box_create` | 保留受管理容器创建、选项默认值、持久化异步语义，以及**绝不关闭宿主 AppArmor** 的底线；特殊的内部 Incus/AppArmor 适配操作移至按需文档，描述与真实能力对应 |
| `box_file_apply` | 直接写明仅支持 `create`（独占 SFTP `O_EXCL`、回读验证），`replace` 和 `delete` 当前一律返回 `UNSUPPORTED`；尽管这两个 Mode 在输入 Schema 中存在 |
| `box_peer_set` | 保留方向性、已固定的 Project/ACL、PRS/操作员审核及 `ADMITTED/PENDING` 不能证明网络已生效的约束 |

当前九个被统一尾注覆盖的 Tool：`box_create`、`box_start`、`box_stop`、`box_update`、`box_delete`、`box_exec`、`box_file_apply`、`box_peer_set`、`operation_cancel`。源代码 `internal/control/strict_tool.go` 以非 ReadOnly 注解决定追加，修改时需要核实仍恰好覆盖这些 Tool；`box_file_apply` 虽用同步 SFTP 创建，其入库并执行流程依然产生持久化异步 Operation；`box_peer_set` MCP 端仅受理，Root/PRS 另行审批执行。

### 2.4 A 的独立验收与 Token 预算

- 比较 **原版（`cd3b2b95`）/精简 Tool Description/精简加 Server Instructions** 三组，记录完整 `tools/list` 的可见字符数和真实 tokenizer Token 数，勿只用字符长度代替 Token。
- 在实际 ChatGPT MCP 客户端测试 Instructions：服务端返回、客户端读取、模型实际调用行为分别有独立证据。
- 测试 Agent 能否查找原始 Operation、处理 UNKNOWN、正确复用 Key 而不重复 Create/Exec/Peer、正确区分 `box_exec` 的 Operation `SUCCEEDED` 与非零 Guest `exit_code`。
- 不得因 Agent 没遵守说明就静默弱化服务端权限、幂等或原生执行安全约束。
- `MCP Resource` 不作为本阶段必需能力：其存在也不保证 ChatGPT 自动读取，可留作未来长篇参考指南。

## 3. B — `box_activity_get` 只读 Activity & Evidence Overview（独立构想）

### 3.1 解决的问题和严格边界

回答“**BoxWeave 在授权范围内最近记录了什么、结果有何依据、还需要核查什么**”，不回答“Agent 完成了百分之几”。

只被动复用 BoxWeave 自身的持久化状态，不要求 Agent 主动报告任务/计划；不采集 Agent 私有推理、Prompt、GitHub CI、外部终端、聊天状态或未持久化的只读 MCP 请求；不默认进行 Guest Exec、读 Guest 文件/输出全文、主机网络扫描或 Root Journal 读取。Agent 不使用 BoxWeave 时，本接口不能推断它没有工作。

不得把 `ADMITTED` 误认为底层 effect 已发生、把 `SUCCEEDED` 误认为 Exec 退出码 0、把 `RUNNING` 误认为当前有活着的进程、把 `peer_grants.CONFIRMED` 误认为当前内核连通性得到实时证明。

### 3.2 HEAD 所证实的 Schema v3 数据及可信度

| 数据 | 可提供事实 | 不可推出 |
|---|---|---|
| `operations` | Action、状态、创建/更新时间、Actor、关联标签、Box UUID、错误类别、原始 ID | Agent 任务完成、当前原生状态 |
| `operation_events` | 已持久化的状态转换和分类事件 | 所有 MCP 调用或全部原生执行轨迹 |
| `operation_resource_claims` | 原始操作曾声明的资源键 | 单独凭声明判断当前真实锁定（须考虑状态与恢复审计） |
| `owned_boxes` | 已登记 UUID、incarnation、Project 与生命周期状态 | 当前 Native Running 或 Guest 就绪 |
| `exec_outputs`、`exec_output_gc` | 已记录退出码/截断与已知的清理状态 | 输出文件永远可读；未记录退出码时的命令成功 |
| `recovery_audit` | 已持久化的人工审计/释放记录 | 自动恢复成功或当前 host-root proof |
| `network_bindings`、`network_address_fences` | SQLite 中网络绑定、保留和隔离声明 | 当前实际 nftables/ACL/流量状态 |
| `peer_grants` | 方向性 Grant 的持久化状态 | 当前实际网络连通性、所有历史操作反向映射 |
| `forward_claims` | 已有表结构/潜在声明 | B4 原生转发工具已实现或验收 |
| `host_status` | 已有独立只读健康接口 | 在聚合事务中自动得到一致的主机状态快照 |

**当前明确的数据库限制：**

- `operation_list` 按 `created_at ASC,id ASC` 分页；既有索引大体按创建时间/Actor/work/state 排列，未为 `updated_at` 最近变化排序配置索引。不可从历史第一页推断最近活动，也不可在全历史表上无界排序。
- 可通过创建时间的**反向有界索引扫描**获得最近**创建**的操作，明确 `window_basis=created_at`；不要标成最近更新。UNKNOWN 或未终态“全局积压”也只应在明确授权+时间窗口内计数。
- `operation_resource_claims` 是历史声明；锁是否仍有效受 Operation 状态、`recovery_audit` 和 Peer Project 全局屏障影响，不能只看 claim 。
- `peer_grants` 的已完成 Grant 可能清空 `operation_id`；不保证从当前授权表精确还原历史 Operation。
- 未通过 Admission 的前置拒绝、全部只读请求、外部命令通常没有持久化 Operation，故 **`coverage` 必须显式说明缺口**。

### 3.3 MVP 的建议输入、视图与输出

拟议请求 `box_activity_get(work_ref?, agent_ref?, since?, view?, limit?)`：精确 Schema 以后续独立设计和现有 Go 类型为准。可选 `overview`（默认）、`issues`、`evidence` 视图；**第一版只实现 overview**，对另外两种视图保留未来设计空间而不提前宣称支持。

- 默认最近 **24 小时**（按 Operation 创建时间），最多 **5–8 条**简短最近记录/关注异常；设置明确可覆盖时间范围、最大返回字节和数据库执行成本上限。300–600 Tokens 是测量目标，不是协议保证。
- 输出 `observed_at`、`coverage`（授权层、来源、时间窗口、是否运行原生实时核验、截断/可见性缺失）、明确范围内的 `summary`、`recent`、`attention`。
- 每条仅包含少量安全字段：`operation_id`、`action`、`state`、`box_uuid`（有授权时）、`error_class`（安全分类）、`command_success`（有可证明退出码才返回）以及现有有限枚举的 `next_action`。
- `operation_get`、`operation_output_get`、`box_get`、`host_status` 是可推荐的既有**只读**进一步核验工具；涉及 Root 的情况只标 `HOST_LOCAL_REVIEW`，不发送任意命令/路径，不自动修复或重试。
- Count、recent、attention 应来自同一授权 SQLite 只读事务快照；任何额外 Native GET 都需明确另行标注观察时间，不可装作与 SQLite 原子一致。初版可完全不做 Native GET。

示意（假数据，非定稿 Schema）：

```json
{
  "observed_at": "2026-11-18T20:00:00Z",
  "coverage": {
    "source": "authorized_durable_operations",
    "window_basis": "created_at",
    "native_live_probe": false
  },
  "summary": { "nonterminal": 2, "failed": 1, "unknown": 1 },
  "recent": [
    { "operation_id": "example-op-1", "action": "box_exec",
      "state": "SUCCEEDED", "command_success": false }
  ],
  "attention": [
    { "operation_id": "example-op-2", "state": "UNKNOWN",
      "next_action": "HOST_LOCAL_REVIEW" }
  ]
}
```

特别注意：`command_success` 仅在有可靠的 `exit_code` 证据时提供明确 bool，否则省略/未知；Exec `SUCCEEDED` 不保证退出码 0。统计字段若按创建窗口计算，须明示“窗口内创建、当前状态”，避免将其称为“窗口内失败次数”或全部当前活跃执行。

### 3.4 权限和最小实现路径

- 必须复用 `operationPrincipal()` 及现有可撤销凭据检查边界：**admin 只看当前 layer 的授权记录；viewer 只能看自己 Actor 的记录**。任何聚合、计数、最近记录、错误信息均应在授权过滤之后完成；不能通过 `work_ref`、`agent_ref`、cursor、统计数量或错误响应扩大可见性。
- 采用新的只读、限界 SQL 查询和已有表，按 `created_at DESC` 最近记录 + 有界窗口统计；仅对选择的少数 ID 用主键关联 `exec_outputs`、资源声明；避免任意表 JOIN 扩大结果和 Host 全局敏感字段。
- 可在 **不新增表、不修改 DDL、不增加 Root 权限、不改变 Native worker/PRS、不要求 Agent 主动上报** 的条件下交付此有边界 MVP；未来需要 `updated_at` 排序或事件级聚合时须单独评估索引与严格 Schema v3 指纹/升级风险。
- 最小文件预期：新增 `internal/state/activity_query.go`、`internal/state/activity_query_test.go`、`internal/control/activity_tools.go`、`internal/control/activity_tools_test.go`；接入 `internal/control/server.go`；更新公开 Tool 数量/Schema 验证与设计文档。预计无需变更 `worker.go`、`recovery.go`、Root PRS 或 SQLite DDL。

### 3.5 新 Tool vs `operation_list` 扩展

| 方案 | 好处 | 代价 |
|---|---|---|
| **A：`box_activity_get`（倾向推荐）** | 功能意图明确、Agent 易发现，聚合输出不污染原操作查询/游标语义 | 增加公开 Tool，需单独批准、独立 MCP/权限/性能验收 |
| **B：扩展 `operation_list`** | 不增加公开 Tool | 使历史查询兼任跨资源概览，兼容性、Schema、分页和响应语义更复杂 |

**不在本设计阶段自动选择实施。** 当前源代码注册的公开 Tool 为 **17 个**；S4-B 目标 20 个，已获批准的 S4-C 将目标改为 21 个。如果额外增加 `box_activity_get` 则预计最终为 **22 个**，需要用户独立确认。

### 3.6 独立测试与签收条件

1. 真实断线/新 Agent 接手：找回原始 ID，正确识别 UNKNOWN、FAILED、未终态，不自动生成危险的重放请求。
2. `box_exec` 的 `SUCCEEDED` 且退出码非零 -> `command_success=false`；没有可靠退出码时不得捏造 true/false。
3. admin/viewer、不同 Actor/Token、不同 Layer、伪造或重复 work_ref/agent_ref、禁用/过期 Token、不同时间窗口均不得跨越授权边界。
4. 大量历史 Operation、长时保留 UNKNOWN、并发写入、不同分页与窗口过滤，证明 SQLite 快照一致、有界查询和完整/截断的诚实标识。
5. 网络 Grant 的 SQLite 状态不当作实时网络通畅；Root Journal、外部资源、敏感命令/env/输出全文都不得泄漏。
6. 比较真实多次 MCP 查询与单次概览的 Tool 次数、总 Tokens、延迟和 Agent 决策准确性，只有确定节省且不会制造错误信心时才签收。

## 4. 开发安排与不干扰当前验收

1. **当前近期**：A 的精简说明与 Server Instructions，先测 SDK/真实 ChatGPT，再按单独小改动提交与核验；不新增 Tool。
2. **当前 S4-B**：继续原 S4-B3–B6 独立验收；不能用 Activity 研究代替任何网络安全门槛。
3. **S4-C**：完成已选定的 `box_get(detail=true)`、`box_wait_ready`；可讨论 B 的独立产品决策，默认不插入实现。
4. **v1 后续独立阶段（默认）**：经用户明确批准后再实现 `box_activity_get`；若要求提前到发布前，应先说明 22-Tool 范围变化与新增验收工作，不自动转嫁 S5 的既定发布门槛。

**最终工程判断：** 对于“授权范围内、有限时间窗口的持久化操作与证据概览”，Schema v3 足以提供可信的第一版；**不新增表/不主动上报/不提权/不改变 Root 原生链**可以成立。对于完整 Agent 任务恢复、实时进程或网络状态、全历史变化排序与全局资源冲突，这些前提下不能许诺完整覆盖。

## 5. 源码入口与关联设计

- Tool/路由：`internal/control/server.go`、`strict_tool.go`、`operation_tools.go`、`operation_output_cancel.go`、`box_tools.go`、`file_tools.go`、`peer_tools.go`。
- 数据/状态：`internal/state/store.go`（`SchemaVersion=3`、DDL/索引）、`operation_query.go`（授权与升序游标）、`worker.go`、`recovery.go`、`network_peer_*`、`network_audit_ledger.go`。
- Root 边界：`internal/control/peer_root.go`、`internal/install/native_peer_root_*`；不可将审计接口变成额外 Root API。
- 既有文档：[Agent Operation Continuity](../design/AGENT_OPERATION_CONTINUITY.md)、[MCP Tool Contracts](../design/MCP_TOOL_CONTRACTS.md)、[MCP Tool Catalog](../design/MCP_TOOLS.md)、[S4-B six-stage plan](S4_B_SIX_STAGE_PLAN_2026-11-16.md)、[S4-C readiness plan](S4_C_BOX_INSPECTION_READINESS_PLAN_2026-11-16.md)、[external Task Checkpoint](TASK_CHECKPOINT_CONTRACT.md)。

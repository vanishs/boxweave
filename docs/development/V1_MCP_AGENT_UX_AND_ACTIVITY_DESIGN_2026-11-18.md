# BoxWeave v1 — MCP Agent 使用体验与 Activity/Evidence 概览设计（2026-11-18）

**2026-11-19 当前方向更新（规划，非新 Tool 实施审批）：** PR #153 在核查 HEAD `85161941` 已移除 `admin/viewer` 和应用 OAuth Scope，SQLite 使用 Schema v4，独立 Actor 与资源/凭据边界仍保留。S3/S4-A/B2/B3 具 source-equivalent 原生回归证据，但不是最终 exact-HEAD 或真实 ChatGPT PASS。**先复现 TC-015/016 的真实双 Agent 接手与外部 Checkpoint 冲突，用现有 Tools 测量误判、Tool 调用量与证据缺口；再调整 MCP Description/Instructions；若发现有界证据检索做不到，再提最小数据库查询/索引整改。** `box_activity_get` 依旧独立、尚未获得 Owner 开发授权。参见 [更新后的连续性验收矩阵](../verification/CONTINUITY_ACCEPTANCE.md)。下面有些 2026-11-18 的历史 Scope/版本描述不代表当前源码。

**状态：OWNER-ACCEPTED DESIGN / NOT IMPLEMENTED。** 本文记录已讨论的设计方向，不表示 `box_activity_get`、Server Instructions 或实际客户端行为已经实现/验收。开发仍限定 [Draft PR #153](https://github.com/vanishs/boxweave/pull/153)；2026-11-18 复核源码参考 `10cb73d11cb9f538bf525078a77b293c693a8d4d`，实施前必须重新读取远程 HEAD。**本次文档修订不是新增 Tool 的产品审批**；未经另行批准，不合并、不发布、不扩展 S4-B 验收范围。

## 1. 目标、范围与实施顺序

- **A — 近期：MCP 使用说明优化。** 让 Agent 在正常操作时先看懂 Tool 的实际职责；收到超时、丢失回复或中断后，知道找回原始 Operation、查看证据并避免重复执行。
- **B — 独立后续：`box_activity_get`。** 为**多 Agent 可靠接手与安全并发**提供一条只读的授权操作、资源阻塞、历史异常及原始证据概览；第一版目标包含 `overview`、`issues`、`evidence` 三种真实实现的视图，不以只做一个列表代替证据贯通。
- 两者分别评估、测试、授权、实现和签收。A 不依赖 B；B **不是** Agent 任务数据库、工作流引擎、自动分工/调度、实时 Agent 存活监控、持续上报系统、进度百分比或完整 MCP 调用轨迹。可信 Agent 可协调，但不能以共享标签绕过认证、原始幂等、Box 资源锁或 Git/Checkpoint 并发控制。
- S4-B3–B6 继续按原独立验收推进；S4-C 既定 `box_get(detail=true)` 和 `box_wait_ready` 仍优先。B 优先考虑作为 **S4-C 后、S5 最终冻结/全面多 Agent 验收前的独立候选工作**，使 S5 能真实复核新能力；仍须单独批准第 22 个公开 Tool、开发范围与新增验收，未批准时 S5 继续采用现有 Operation + 外部 Checkpoint。

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

> work_ref and agent_ref are untrusted correlation labels, not authorization or reliable Agent identities. Separate authenticated actors may inspect same-layer authorized original operations, but cannot borrow another actor's idempotency identity or mutate foreign resources. Use operation_output_get to inspect guest Exec exits; SUCCEEDED means the BoxWeave operation finalized, not necessarily that the guest command exited zero. Use box_get to inspect observed native Box state; Running does not prove guest readiness. BoxWeave does not preserve Agent plans, Git checkpoints, conversation state, or activity outside its own durable Operations.

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

### 2.5 当前无角色 OAuth、独立 Actor 与 Agent 接手

- **认证**：OAuth 无应用权限 Scope、无 `admin/viewer` 等角色；有效且明确审批的每个独立 Actor 均可按同一受管 Layer/Project/Box 边界执行管理操作。有效凭据及撤销/过期复查、外来 Box 拒绝继续强制。
- **身份不等于标签**：OAuth Grant 在 Access Token 刷新后维持稳定 Actor，Access Token ID 改变；手动凭据和不同 OAuth Grant 是独立 Actor。 `work_ref` 与 `agent_ref` 均可伪造，只做检索关联。
- **Agent A → B**：B 使用自己的独立凭据，结合外部 Git/Checkpoint、A 的同 Layer 原始 Operation/native evidence 接手未完成的**后续任务**；不能继承 A 的幂等身份、绕过 UNKNOWN 或跨 Layer/foreign 对象访问。
- **核验者**也必须使用独立有效凭据，优先调用只读 Tools；**不恢复 viewer 角色**。不新增 Agent 注册、任务编排、MCP Tasks 或 BoxWeave Checkpoint 存储。

## 3. B — `box_activity_get` 只读 Activity & Evidence Overview（独立构想）

### 3.1 解决的问题和严格边界

回答“**BoxWeave 在授权范围内最近记录了什么、过去有哪些未解决的问题、关联资源是否会阻塞新 Agent、结果有何证据**”，不回答“Agent 完成了百分之几”。其主要用户是接手 Agent、并行执行的 Agent 和独立只读核验者；`overview/issues/evidence` 是同一套受权证据查询的不同视图。

只被动复用 BoxWeave 自身的持久化状态，不要求 Agent 主动报告任务/计划；不采集 Agent 私有推理、Prompt、GitHub CI、外部终端、聊天状态或未持久化的只读 MCP 请求；不默认进行 Guest Exec、读 Guest 文件/输出全文、主机网络扫描或 Root Journal 读取。Agent 不使用 BoxWeave 时，本接口不能推断它没有工作。**S5 多 Agent 可靠接手与安全并发**的另一半由 Agent/工作流持有的 Git/Checkpoint 提供：任务目标、Git HEAD/dirty worktree、后续计划和两个写入者的版本冲突。Activity 不自动读取、存储或解析这些外部状态，也不把 `work_ref` 当并发锁。

不得把 `ADMITTED` 误认为底层 effect 已发生、把 `SUCCEEDED` 误认为 Exec 退出码 0、把 `RUNNING` 误认为当前有活着的进程、把 `peer_grants.CONFIRMED` 误认为当前内核连通性得到实时证明。

### 3.2 当前 Schema v4 数据及可信度

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
- 可通过创建时间的**反向有界索引扫描**获得最近**创建**的操作，明确 `window_basis=created_at`；不要标成最近更新。**但仅按最近 24 小时过滤会漏掉仍然有效的历史 UNKNOWN、网络 Project-wide 屏障或资源锁。** 必须将有界的 `recent_activity` 与跨默认时间窗的 `unresolved_attention` 分开；后者按授权 Layer/Actor 和真实未解决条件查找，并明确覆盖完整性及截断情况。若现有索引无法同时保证有界执行与不遗漏重要遗留问题，先测量并设计最小索引/DDL 变更，不得伪称无界扫描已经解决这个问题。
- `operation_resource_claims` 是历史声明；锁是否仍有效受 Operation 状态、`recovery_audit` 和 Peer Project 全局屏障影响，不能只看 claim 。
- `peer_grants` 的已完成 Grant 可能清空 `operation_id`；不保证从当前授权表精确还原历史 Operation。
- 未通过 Admission 的前置拒绝、全部只读请求、外部命令通常没有持久化 Operation，故 **`coverage` 必须显式说明缺口**。

### 3.3 第一版完整视图、输入与返回约定（待单独批准开发）

拟议公开输入 `box_activity_get(work_ref?, agent_ref?, since?, view?, limit?, operation_id?)`，其中 `view` 第一版**必须实际支持** `overview`（默认）、`issues`、`evidence`；不得仅在 Schema/Description 中承诺、运行时却返回 UNSUPPORTED。精确类型、默认值、权限/异常码和证据选择策略须在实现前通过 Go 类型与 MCP Schema 测试定稿。

- **`overview`**：近期有界 `recent_activity`、历史未解决 `unresolved_attention`、相关 Box/网络/Exec 状态摘要、时间范围、观测/授权/缺失覆盖信息。默认最近 **24 小时创建**的 Operation，最近操作最多 5–8 条；旧 UNKNOWN 等影响当前安全或准入的遗留风险**不受这个近期窗口隐藏**。
- **`issues`**：在同样权限下优先列出 UNKNOWN、失败、未终态、待人工核验、仍有效的准入资源阻塞和原始恢复审计；区分“失败但已释放”“历史声明”“当前仍可能阻塞”，不能凭一张 claim 表直接判断锁有效。
- **`evidence`**：给定可访问的 `operation_id`，或有限的已授权候选，归并原始 Receipt、`operation_events`、安全状态分类、相关资源声明、Exec 退出码/输出可用性及恢复审计，返回证据引用和已有 `operation_get`/`operation_output_get`/`box_get` 等只读下一步。**不**返回原始 Guest stdout 全文、密钥、host Root Journal 或任意诊断命令。

所有视图复用统一的授权/范围/隐私/查询限制；默认 300–600 Tokens **仅是 overview 的测量目标**，issues/evidence 可以有更大的明确字节与记录上限，必须提供截断及继续查原始记录的方式。新增的 `operation_id` 是设计候选，不是已发布 Schema。

输出至少区分：

- `observed_at`：本次 SQLite 快照时间；
- `coverage`：当前 Layer/Actor 可见范围、持久化操作来源、`window_basis=created_at`、原生实时核验是否进行、跨时间遗留问题搜索是否完整以及各视图截断状态；
- `summary`：统计口径严格标识为**窗口内创建的 Operation 目前状态**，不能叫“最近 24 小时内发生的失败次数”；
- `recent_activity`：有界的近期 Operation；
- `unresolved_attention`：即使创建时间早于窗口，仍未确认解决的授权高风险记录；可包含安全资源阻塞摘要；
- `evidence_refs`：原始 `operation_id`、可安全读取的有限下一步和缺证原因。

示意（假数据、非定稿）：

```json
{
  "observed_at": "2026-11-18T20:00:00Z",
  "coverage": {
    "source": "authorized_durable_operations",
    "window_basis": "created_at",
    "native_live_probe": false,
    "attention_complete": true,
    "truncated": false
  },
  "summary": {
    "window_created_nonterminal": 1,
    "window_created_failed": 1
  },
  "recent_activity": [
    { "operation_id": "example-op-1", "action": "box_exec",
      "state": "SUCCEEDED", "command_success": false }
  ],
  "unresolved_attention": [
    { "operation_id": "example-op-older", "state": "UNKNOWN",
      "created_before_window": true, "next_action": "HOST_LOCAL_REVIEW" }
  ]
}
```

**可信度限制：** `command_success` 只有退出码有可靠证据才返回明确布尔值；`SUCCEEDED` 不代表 Guest 退出码 0；`ADMITTED` 不是网络已生效，`RUNNING` 不是活跃进程，SQLite `peer_grants.CONFIRMED` 不是当前内核连通性实时证明。服务端不得输出缺失的证据为“PASS”，不得推断 Agent 任务完成度、计划或心跳。

### 3.4 统一关联、现有安全语义、查询与权限

**实现目标不只是读 `operations` 表**：把已存在的 `operations`、`operation_events`、`exec_outputs/exec_output_gc`、`owned_boxes`、`operation_resource_claims`、`recovery_audit`、`network_bindings/address_fences`、`peer_grants` 和后续 `forward_claims` 的**可授权、可确定关联**串成只读证据投影，补齐当前能力分散且查询不能贯穿的问题。缺少确定关联的字段必须标明 `NOT_LINKED` 或 `NOT_OBSERVED`，不能按名称/IP 猜测原始资源关系。已撤权的授权/已清空的 `peer_grants.operation_id` 不保证能追溯所有旧 Operation。

**有效锁与依赖不能由历史 claim 直接推出：** 读取/解释要与 `Store.Admit()` 中的事务准入语义一致，包括 `ADMITTED/DISPATCHING/RUNNING/VERIFYING/UNKNOWN`、`recovery_audit.RELEASED`、`box_peer_set` Project-wide 屏障、Box 名/UUID/incarnation、有效 Peer/Forward/地址栅栏，以及 `exec_output_gc` 的待清理状态。建议提取/复用**只读的安全状态判定函数/SQL 语义**，并以正反回归测试防止与真实准入检查发生漂移；Activity **绝不**取代 `Admit()` 的事务授权/资源锁。

- 复用当前 **`requestPrincipal()` 的 ActorID、TokenID、Layer** 和 Store 受管范围：同 Layer 的独立有效 Actor 可查看授权原始 Operation，不能跨 Layer/foreign 资源。`work_ref`、`agent_ref` 与共享工作名不扩权；显示发起者时仅用限域不透明 Actor 引用，不泄漏 Token/Grant 或冒充可信 Agent 身份。
- **所有视图均在返回前重新检查可撤销凭据**：沿用 `privateReadReply()` / `RequireActiveTokenID()` 的安全路径，特别覆盖查询期间 OAuth 撤销、过期或轮换。授权必须在聚合、统计、异常筛选之前生效，不能通过总数、游标或资源摘要推断隐藏记录。
- `summary`、`recent_activity`、`unresolved_attention` 和同一调用选取的证据来自**同一已授权 SQLite 只读事务快照**；不使用 unbounded N+1 JOIN、全历史无界排序或宿主范围扫描。最近创建记录按可测量索引有界倒序查，持久未解决风险另外用索引和明确上限查；若结果不完整，给出 `attention_complete=false` / `truncated=true`，不能默默漏报。
- 默认**不**执行 Guest Exec、Guest 文件或宿主原生实时探测。若日后增加 Native GET，必须独立记录时间和授权，不得冒充 SQLite 原子快照；Root/PRS 审计仅允许暴露已有经批准的安全摘要或返回 `HOST_LOCAL_REVIEW`。
- 优先复用 **Schema v4**、现有 Go 查询和索引；TC-015/016 基线证实有界证据检索不足时再提出最小索引/DDL 变更，并在一次性新 DB 中重新通过 DDL 指纹、UNKNOWN、防重放和原生验收。预发布环境无旧 SQLite 兼容义务，但不能将原生未知副作用视为可丢弃。
- 代码入口候选：`internal/state/activity_query.go` / `activity_query_test.go`、`internal/control/activity_tools.go` / `activity_tools_test.go`、`internal/control/server.go`；若提炼真实资源阻塞判断，需同步补充原有 Store admission 回归，不擅自改 Worker、Root PRS、外部 Checkpoint 存储。

### 3.5 新 Tool vs `operation_list` 扩展

| 方案 | 好处 | 代价 |
|---|---|---|
| **A：`box_activity_get`（倾向推荐）** | 功能意图明确、Agent 易发现，聚合输出不污染原操作查询/游标语义 | 增加公开 Tool，需单独批准、独立 MCP/权限/性能验收 |
| **B：扩展 `operation_list`** | 不增加公开 Tool | 使历史查询兼任跨资源概览，兼容性、Schema、分页和响应语义更复杂 |

**设计倾向独立 `box_activity_get`，但本次仅修订文档、不自动授权实施。** 2026-11-18 核查的 PR HEAD 公开 Tool 约 **17 个**；S4-B 完成计划 20 个，已批准 S4-C 再加 `box_wait_ready` 为 **21 个**。若另行批准 Activity，最终预计 **22 个**，必须根据最终实际代码/安装清单调整 S5 **逐 Tool/逐 CLI** 的验收矩阵；S5 旧文档中固定 `20/20` 是历史阶段基线，不应继续用来宣称最终全部工具均已覆盖。

### 3.6 第一版独立测试与 S5 多 Agent 验收对照

1. **三视图实测**：`overview/issues/evidence` 均能通过真实已认证 MCP Schema 调用，返回正确有界结构；Description 与实际实现完全一致。比较多次 `operation_list/get/output_get` 和一次 Activity 调用的调用次数、真实 tokenizer Tokens、延迟、错误判断率及缺证处理。
2. **历史阻塞**：建立早于 24 小时、仍为 UNKNOWN/有效锁/Project-wide Barrier 的原始案例；无同工作标签的新 Agent 仍能在**权限允许的 Layer** 发现相关阻塞。已审核 RELEASED 与仍受保护的 Root UNKNOWN 要区别呈现；大历史数据的索引性能、有界响应、截断/覆盖信息均须证明。
3. **身份与跨 Agent 接手**（TC-011～015）：A 用独立 Actor 创建 Box/Exec/Peer，B 用不同 Actor 查询同 Layer 的原始 Operation、核验 native/Git 后安全继续；**不能借同 Key 冒充 A 的原始幂等身份**。Token 刷新保持 Grant Actor，撤销和跨 Layer/foreign 拒绝仍有效；共享 `work_ref` 不授予超出当前受管范围的权限。
4. **安全并发**（TC-016～018）：A/B 相同或不同 `work_ref` 并发 Box/网络/删除，真实资源准入仍负责冲突拒绝；Activity 只观察，不成为分布式任务锁。外部 Git/Checkpoint 两个写入者的 CAS/版本冲突由其外部存储正确拒绝，Activity 不虚构 Git HEAD/dirty 状态或检查点写入成功。跨 L0/L1 同名标签和受限 Token 不泄漏。
5. **证据真假和异常**（TC-019～020）：原始 `box_exec SUCCEEDED`、Guest 非零退出码 => `command_success=false`；退出码缺失则未知；原生 Root/网络/地址状态无直接 witness 时标证据缺口。以已知坏实现（例如把旧 UNKNOWN 错写成成功、错误继承其他 Actor 的幂等作用域、因时间窗口漏报阻塞）校准测试先红后绿。
6. **权限/一致性/容量**：撤销/过期/刷新 Token 的返还前复查；所有统计在鉴权过滤后同一 SQLite 快照；并发写、分页、水位与长时记录不得越权或产生误导性全局统计。严格禁止原始命令/环境变量/输出全文、host 私有路径、Root Journal 和未经审查的修复建议泄漏。

**阶段关系：** 已批准的 TC-015/016、TC-019/020 原始 S5 验收门槛继续生效；若 Activity 获独立批准且进入 S5 最终候选，再把上述新增用例纳入该候选逐 Tool 验收。Activity 不能替代外部 Checkpoint/Git 并发控制、#151 原生确证、物理网络/IPv6 或独立 verifier。

## 4. 开发安排与不干扰当前验收

1. **先验收基线，再进行 A：** 在 disposable 已安装环境用现有 Tool 完整执行 TC-015/016，记录真实接手成功/失败、调用次数、Token、延迟、误重试与证据缺口。随后仅针对可重复的发现/理解问题优化 Tool Description 与 Server Instructions，用相同场景 A/B 对照；不新增 Tool、不修改原生执行语义。
2. **S4-B：** 继续 S4-B3～B6 独立验收；Activity 研究/文档不替代网络安全、Peer/Forward/Delete、恢复与证据签收。
3. **S4-C：** 先实现已批准的 `box_get(detail=true)`、`box_wait_ready`，正常准备 21-Tool 矩阵。
4. **独立 Activity 候选（倾向 S4-C 后、S5 最终验收前）：** 用户另行批准第 22 个 Tool、三视图完整契约、查询性能/权限/证据关联的实施成本与具体签收后再开发。未批准时不阻碍原 S5 多 Agent 目标，继续用 `operation_list/get` + 外部 Checkpoint。
5. **S5：** 核验**多 Agent 可靠接手与安全并发**，而不是自动分工/任务编排。TC-015 要求真实 B 读取外部 Checkpoint/Git 和 A 的原始证据、在已确认原始结果后继续；TC-016 要求两个 Checkpoint 写者使用外部版本/修订控制，不丢失冲突。最终候选包含哪些 Tool 就完整验收哪些 Tool；历史写死的 20/20 不能掩盖已批准的 21 或待批准的 22。S5 #151、完整嵌套和最终独立原生验收不放宽。

**最终工程判断（以 Schema v4 重新核验）：** 现有 Schema 已提供构建初步授权活动和证据关联的基础；但要把**跨默认 24 小时的真实阻塞、三视图、有界查询性能和安全状态语义贯通**作为第一版承诺，不能先验认定完全不需索引/DDL 或其他有限代码调整。只读 Activity 不能证明 Agent 的任务完成、外部 Git/CI、Guest 实时进程和 Root 当前内核效果；这些边界由覆盖信息和独立核验明确表达。

## 5. 源码入口与关联设计

- Tool/路由：`internal/control/server.go`、`strict_tool.go`、`operation_tools.go`、`operation_output_cancel.go`、`box_tools.go`、`file_tools.go`、`peer_tools.go`。
- 数据/状态：`internal/state/store.go`（当前 `SchemaVersion=4`、DDL/索引）、`operation_query.go`（授权与升序游标）、`worker.go`、`recovery.go`、`network_peer_*`、`network_audit_ledger.go`。
- Root 边界：`internal/control/peer_root.go`、`internal/install/native_peer_root_*`；不可将审计接口变成额外 Root API。
- 既有文档：[Agent Operation Continuity](../design/AGENT_OPERATION_CONTINUITY.md)、[MCP Tool Contracts](../design/MCP_TOOL_CONTRACTS.md)、[MCP Tool Catalog](../design/MCP_TOOLS.md)、[S4-B six-stage plan](S4_B_SIX_STAGE_PLAN_2026-11-16.md)、[S4-C readiness plan](S4_C_BOX_INSPECTION_READINESS_PLAN_2026-11-16.md)、[external Task Checkpoint](TASK_CHECKPOINT_CONTRACT.md)、[TC-011～020 S2.5/S4/S5 Continuity Acceptance](../verification/CONTINUITY_ACCEPTANCE.md)、[S5 release card](phases/05_RELEASE_ACCEPTANCE.md)。

# BoxWeave v1 — MCP Tool → 数据库 / 事件 / 日志关键信息贯穿审计与实施准备（2026-11-19）

> **2026-11-19 SOURCE_ONLY investigation completed:** the pinned-HEAD 17-Tool matrix, SQLite/events/Root association graph, prioritized broken links, conditional Schema/index proposal and next-agent staged implementation plan are in [the completed lineage audit](V1_MCP_TO_DB_LOG_LINEAGE_SOURCE_AUDIT_AND_IMPLEMENTATION_DESIGN_2026-11-19.md). No Go/DDL/public Tool implementation or new runtime/native acceptance was performed. The original requirements below remain the investigation contract, not an unfinished next task.

**状态：Owner 明确指定的下一轮源码调查与开发方案任务；尚未完成逐 Tool 审计、尚未实现 Activity。** 起始代码参照 [Draft PR #153](https://github.com/vanishs/boxweave/pull/153) `dfb393d9715be0755583de60182f41ad2bab38bd`（执行者必须重新获取最新 HEAD）。唯一开发分支 `feat/v1-s4-runtime-integration-20261116`。本轮**只调查、必要的隔离只读实验、写实施设计**，不修改 Go/DDL、不中断其他开发、不合并/发布/部署。

**核心问题不是“能否把多张表 JOIN 起来”，而是：每一个 MCP Tool 产生、携带、持久化和返回的关键事实，最终能否被一个后续 Agent 通过已有凭据与只读接口准确找回？** 若字段在入口已缺失，Activity 再聪明也无法重建。调查必须先自 MCP 入口向下追踪，再从数据库/事件/日志向上逆查；同时区分可知事实、缺失、不应记录的秘密和本来不承诺持久化的访问。

## 一、目标、顺序和边界

1. **完整枚举实际公开 MCP Tools → 逐 Tool 双向信息链审计**；不能只审计九个变更工具或已有 Operation 查询工具。按当前源代码实际 `register*Tools` 与正式 `tools/list` 枚举并核对（此前有 17 个的源码/SDK 验证证据，但**不是将来 20/21/22 的固定数量**）。未实现的 S4-B4–B6/S4-C 工具单独标 `PLANNED/NOT_IMPLEMENTED`，不得虚构字段、记录或 PASS。
2. **再调查 SQLite Schema v4、Go Store、事件和日志的写入/读取合同**，建立“键从哪来 → 谁持久化 → 有何时间/一致性保证 → 如何授权回查”的逐字段证据链。明确数据库、结构化控制器日志、Root/PRS 日志、Native Operation 与外部 Git/Checkpoint 的不同权威性。
3. **最后设计 Activity 统一只读证据投影与最小闭环整改**，比较：仅查询层、补充写入贯穿/可选 Tool 参数与返回字段、必要时最小 DDL/索引，以及扩展 `operation_list` 的替代方案。不能不经逐 Tool 审计就拍板先改数据库，也不能只做一个漂亮的近期操作列表。
4. **后置 A2 的真实 ChatGPT/单 Agent 断线行为试验**，当前先完成整套数据贯穿设计；正式 S5 TC-015/016 双 Agent 接力/外部 Checkpoint 冲突验收更后面执行。A1 的 Instructions/Description 已通过官方 SDK 和 Go Race 源码测试，不再重复开发。
5. **遵守既有边界**：无 admin/viewer 角色、无应用 OAuth Scope，独立 Actor、当前 Layer/Project/owned UUID/incarnation 授权、原始按 Actor 幂等、UNKNOWN 防重派、Token 撤销/返回前复查、Guest-root 不等于 host root。无 Agent 计划表、调度器、泛用日志采集/秘密归档；`box_activity_get` 尚须独立批准公开 Tool 与验收范围。

**成功定义**：同一资源/逻辑工作从 `box_create → box_exec → box_file_apply → box_peer_set → box_delete`（仅选择当前已实现且获授权的步骤）产生的**真实持久化原始 Operation 与证据**可由合法只读接口定位、核对、连接；任何无法连接的来源有明确缺失原因，绝不猜。Agent 的任务目标/下一步计划依然不由 BoxWeave 存储。

## 二、逐 MCP Tool 全覆盖审计（不得跳过只读、拒绝和失败路径）

以**真实 MCP `tools/list` 和 `internal/control/server.go` 及各 `register*Tools`**生成工具清单，核对 catalog/tests。当前可核查的工具族包括：

| 工具族 | 当前已知入口（不得把此列当最终枚举） | 核对重点 |
|---|---|---|
| 状态/容器读 | `host_status`, `box_list`, `box_get` | 是否只返回快照、是否有 `tracking_id`、Box UUID/incarnation、是否产生任何**持久**追踪；区分观测状态与历史 Operation |
| 容器生命周期 | `box_create`, `box_start`, `box_stop`, `box_update`, `box_delete` | `idempotency_key`、`work_ref`/`agent_ref`、Actor、请求 ID、Box name→UUID→incarnation、资源键、原生 Operation ID、异常/回执 |
| Guest Exec | `box_exec` | 输入命令含密钥时的脱敏与 `safe_intent`、原始 Operation、原生执行、退出码、输出/GC、`SUCCEEDED` 与 Guest exit=0 的严格区别 |
| Guest 文件 | `box_file_list`, `box_file_read`, `box_file_apply` | 只读是否持久化、写入模式仅 `create`、文件内容不得进入日志/SQLite、文件操作与 Box/Operation 的确定关联；拒绝的 replace/delete 是否不产生可追踪 Operation |
| 网络/Root | `box_peer_set`（及未来真实注册的 Forward/其他网络工具） | 两端 UUID/incarnation、Direction、共享 ACL/project barrier、PENDING/CONFIRMED/UNKNOWN、Root/PRS 的独立审计与可证实的关联（不可把批准等同网络已生效） |
| 原始 Operation 查询/控制 | `operation_get`, `operation_list`, `operation_output_get`, `operation_cancel` | 过滤字段、Cursor、原始 ID、Token 有效性、撤销竞态、查询是否只在同 Layer、Cancel 是否创建另一独立 Operation |

对**每一个实际公开 Tool**输出完整矩阵；不能因为没有数据库记录就跳过，必须说明它是“**已设计为只读/未承诺持久化**”“**拒绝发生于 Admission 前**”“**本应持久化却丢失**”或“**仅有受限结构化日志**”。禁止为凑覆盖制造假 Operation 或扩大日志记录敏感参数。

### 必须填写的矩阵列

`tool`、`implemented_on_HEAD`、`read/mutate`、`input_contract`、`result_contract`、`RequestPrincipal(actor_id/token_id/layer)` 来源、`work_ref/agent_ref` 传递、`idempotency_key` 所属 Actor 范围、`tracking_id` 与 `request_id` 区别、`operation_id` 的产生/返回/持久化、`native_operation_id` 的保存时机、`resource_key/claims`、`box_uuid/incarnation`、`SQLite table/column`、`operation_events/recovery_audit`、`exec_outputs/GC`、`network ledger/Root audit`、`structured log`、`authorized lookup path`、`pre-admission refusal/UNKNOWN/timeout behavior`、`missing link`、`necessary change`、`source lines/tests`。

每条链标记 `PROVEN`（源码 + 可复现检查）、`PARTIAL`、`MISSING`、`NOT_APPLICABLE`、`PLANNED`。**仅仅 Schema 中存在某列，不代表生产者真的写入它。** 新字段建议必须指出生产者、写入时机、所有读取方、回滚/异常状态和是否影响幂等意图哈希。

## 三、跨工具 / 跨层证据贯穿审计

1. **入口身份与标签**：验证 SDK Bearer → `requestPrincipal()` → `admitNative/replayNative/Store.Admit` → `operations.layer/actor_id/token_id`；`work_ref/agent_ref` 仅为用户提供的关联标签，不保证真实 Agent 身份。OAuth Access Token 轮换时 Actor 稳定、Token ID 可变；不同 Actor 不能继承原 Key。
2. **同一原始 Operation 的主键链**：MCP 请求/Tracking ID / Request ID / `operation_id` / Incus 原生 Operation ID 的来源、区别、稳定性及完整性；非同步或 native 404 场景的证据边界。回执与数据库可逆查，无原生 ID 需解释原因。
3. **资源身份贯穿**：Create 前只有名称锁，Create 后才有 UUID/incarnation；Stop/Start/Exec/File/Delete/Peer 两端以及未来 Forward 的关联须以受管 UUID/incarnation 和受控 key 为基础，不按 Box 名称、IP 或共享 `work_ref` 猜测。Box 删除/重建、同名、同 IP 复用必须安全区分。
4. **时间线/状态**：`operations`、`operation_events`、`recovery_audit` 与异步 Worker 状态推进是否能形成有界、因果明确的证据链。真正有权威性的只有已持久化且可验证的事实；未 Admission 拒绝、未持久化只读调用和外部操作不强行归入历史。
5. **Guest 执行/文件**：Exec 输出存在/GC/截断/退出码是否可以用 Operation ID 查到；文件 create 的可核验 post-state 与原始 Operation 是否关联；对未储存的原始命令/Guest 文件全文保持隐私。
6. **网络/Root**：`network_bindings`、`network_address_fences`、`peer_grants`、`forward_claims`、Root/PRS 审批及原生安全事实的真实 link；撤权后可能清空 `operation_id`，要区别可证历史与当前快照。Root Journal 不是默认 MCP 返回内容。
7. **日志事实审计**：`slog` 等应用日志与 Root/Native 日志，逐项核实实际包含什么：当前 MCP middleware 只记录方法、脱敏 Actor 引用、耗时、粗粒度 outcome 的事实，不可将其称为“记录了每个参数、Operation ID 和全部 MCP Call”。评估是否有必要增加**脱敏、稳定、有限、可授权**的关联 ID，而不是原始 token/argv/env/stdout/Host journal；区分可查询的持久 SQLite 与可能轮转/丢失的日志。
8. **从 Agent 回查**：验证 `operation_list/get/output_get` 及将来的 Activity 能否用同 Layer/受授权 Actor 的已知 ID、标签或时间找到原始数据，能否发现**时间窗外的 UNKNOWN 和有效资源阻塞**。空结果、非零 Guest exit、`ADMITTED`、`peer_grants.CONFIRMED`、`Running` 都不能被解释成任务完成/当前网络生效。

关键查验点：`Store.Admit()` 的 UNKNOWN/project-wide/各资源锁判断、`recovery_audit.RELEASED`、`exec_output_gc` 锁、`GetScopedOperation` 与 `privateReadReply/RequireActiveTokenID`。Activity 的只读“仍有效”判断应当重用或有测试保证与 Admission 语义一致，绝不从孤立的历史 claim 推断当前锁。

## 四、落地设计必须比较的选项

| 选项 | 涉及修改 | 应回答的问题 |
|---|---|---|
| A：现有写入和 Schema 已足够 | 仅增加授权只读投影、`box_activity_get`（待批） | 是否每个必须追踪的值都已持久化并能准确连接？如何高效检索旧 UNKNOWN？ |
| B：小范围端到端修补 | 修改**确实丢失信息**的 Tool 输入/输出、Control→Store 传递、事件写入、有限日志，再加 Activity | 缺的是源头信息还是查询能力？能否不增表，用既有列/关联键补齐？对已有客户端/幂等/安全有没有影响？ |
| C：数据库索引/DDL 修正 | 极少量新索引或可信关联字段 + 查询层（必要时 B） | 哪个实际过滤/排序/关联基准不能通过 Schema v4 的有界查询完成？是否有 `EXPLAIN QUERY PLAN` 证据？Schema 指纹/重建/回滚如何验收？ |
| D：不增加新 Tool | 扩展 `operation_list` 或复用其他现有接口 | Agent 找回调用量和语义成本会否反而增加？如何保留现有 Cursor/API 安全性？ |

必须评估**没有接口变化**、**只加可选字段**、**改返回结构**的真实差异；**不强制每个 Tool 增加 `work_ref`、`agent_ref` 或 `operation_id`**。只在能证明它是必要、可信、可产生、可回查的关联键时建议。无理由不得扩大公用 MCP Tool 清单；若推荐 `box_activity_get`，明确 Owner 仍需独立确认公开 Tool 数与验收范围。

## 五、交付物（本轮必须写入仓库，不止讨论）

1. **源码固定的逐 Tool 贯穿矩阵**：列出实际工具数、每个 Tool 所有关键字段及 `MCP → Control → State/Native → DB/event/log → Reply → readback` 的证据/断点；标明不适用和未实现者；附源码行号或测试入口。
2. **数据流/ER 关联图**：至少 Operation、事件、资源索赔、owned Box、Exec/GC、恢复审计、网络/Peer/Forward、受限 Root 日志；每条边注明**确定 FK/逻辑关联**还是不可靠推断（不能仅画箭头）。
3. **缺口清单**：按 `P0 无法找回/错误关联`、`P1 历史异常遗漏/资源阻塞错误`、`P2 体验/性能` 分级；有源头缺失就先修生产端，再优化 JOIN。注明拒绝/未持久化场景是设计边界还是 Bug。
4. **设计草案**：明确采集/写入/投影的权威性与隐私，含 API 输入输出字段草案、`overview/issues/evidence`、覆盖/截断/未观测、鉴权与 Token 临界撤销、SQLite 同一只读快照、无界扫描防护。
5. **具体文件清单和分阶段实施计划**：建议先做 `贯穿修补和数据库单测` → `Activity Store 只读查询` → `MCP Tool 封装` → `安全与容量`；每阶段给正反测试、回归范围、Owner 决策点。若调查证明无需源头修补/DDL，明确直接跳过。
6. **调查与测试边界**：本轮可使用静态源码、disposable SQLite 的只读/最小 fixture 及 `EXPLAIN QUERY PLAN`；不得改正在运行的 Controller/Root、数据库或真实宿主安全资源。所有观察写明是 `SOURCE_ONLY`、`SQLITE_FIXTURE`、`SDK` 或 `NATIVE`，没执行的绝不写 PASS。
7. **现有材料索引**：对照 [Activity/UX 主设计](V1_MCP_AGENT_UX_AND_ACTIVITY_DESIGN_2026-11-18.md)、[A1 验收](../verification/MCP_AGENT_UX_A1_SOURCE_VERIFICATION_2026-11-19.md)、[Task Checkpoint 边界](TASK_CHECKPOINT_CONTRACT.md)、[S5 TC](../verification/CONTINUITY_ACCEPTANCE.md)；不要误以为 BoxWeave 是外部 Agent 的任务数据库。

**本轮不做：** 代码实现、真实 ChatGPT A2 行为验收、完整 S5 双 Agent 接力、生产 Schema/数据迁移、发布 RC、清理共享资源。遇到阻碍继续完成不依赖项，报告 `BLOCKED/UNKNOWN` 和下一步可复现的只读验证，不以猜想代替证据。

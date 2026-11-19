# BoxWeave v1 — 单一授权能力与历史包袱整改方案（2026-11-18）

**状态：阶段 1 的角色/Scope 全链源码改造已写入 Draft PR #153，独立 Box 的 Go 全量单测已经通过；尚未取得真实 ChatGPT 无 Scope 端到端 PASS，未完成独立 Native/CI 签收。** Owner 已确认：当前仍处于 v1 开发期、没有需要兼容的正式用户，测试数据、API、授权和 SQLite 允许不兼容重建；目标是尽早移除无实际需求的 `viewer` 及其他**已确认**的历史包袱，避免把其维护成本带到 S5 和正式发布。此授权不等于允许删除未经核验的宿主原生资源或直接修改共享 L0–L5。

- **开发位置**：唯一 Draft PR [#153](https://github.com/vanishs/boxweave/pull/153)，分支 `feat/v1-s4-runtime-integration-20261116`；源码调查参考 HEAD `36b6662627050b81b946a7b9eedf922fb2e5b271`，实施前重新确认远程 HEAD。
- **当前交付**：Owner 已明确授权阶段 1，一次性修改 Go/新 Schema/测试和现行文档；**不**迁移旧 DB、不部署共享宿主、不改旧发布产物。阶段 2/3 独立 Native 和真实 ChatGPT 验收仍需继续。
- **产品定位**：**人 + 多个可信 Agent，可靠接手、安全并发**。不做 Agent 主从/调度、身份注册、通用 RBAC、工作流引擎。彼此信任是协作假设，不取代有效凭据、Project/Box 所有权、幂等或内核隔离。
- **优先级**：清理双角色基础负担属于当前 v1 架构收敛，可在新增 S4-C/Activity 接口之前做完；不得将它宣称为 S4-B3～B6 已通过，或绕过旧安全目标。

## 0. 执行决定（2026-11-18，Owner 授权开始第 1 阶段）

**当前执行：一口气完成角色全链重构，不引入旧/新并行的认证架构。** 唯一开发位置为 PR #153 的现有分支；先读取远程 HEAD 后实现，PR 保持 Draft，不合并、不发布 RC。

1. **阶段 1 — 一次性重构（现在执行）：** 将 OAuth discovery / consent / authorization / token / refresh，MCP Bearer admission，手动 CLI credential，Operation actor/query，SQLite schema、所有上层 Tool 与调用方按方案 B 同步变更。彻底移除应用权限 `admin/viewer`、`mcp:admin/viewer` 与 `-role`；只承认经过明确批准的全管理凭据。新建 SchemaVersion=4 干净基线，不实施 v3→v4 自动迁移，不接受旧 Token/Grant。保留资源隔离、Token 二次校验、OAuth PKCE/refresh/revoke、original actor 幂等及 UNKNOWN 防重放。实施前静态引用清单和改后零旧角色引用扫描均须留痕。
2. **阶段 2 — 集中测试：** 每次修改优先在隔离 Box 运行格式化、编译、针对性 Go/SDK HTTP、撤销/跨 Actor/跨 Layer/负例测试；修复直到通过。原 viewer 专用测试替换为真实身份/安全拒绝测试，不以跳过测试作为完成。此阶段不改变共享 L0–L5 与宿主 ACL。
3. **阶段 3 — 最终签收：** 在可销毁全新 Project/DB 中验证完整相关 S2/S3/S4 Native 回归，并用真实 ChatGPT 验证 OAuth 发现、明确同意、授权码、工具访问、刷新和撤销；同一最终 SHA CI PASS，独立审计后签收。SDK 源码静态兼容不能替代 E2E。任何未跑通项目应标 BLOCKED/NOT_TESTED，禁止假报 PASS。

**速度原则：** 不单独实施旧 DB 迁移、兼容 API、双权限版本或不相关 SDK 升级；不重验尚未开发的 S4-B4–B6。若发现清理目标涉及原生宿主安全和历史 UNKNOWN，立即从本次重构范围剔除，保留证据，绝不为删代码降低安全检查。工作结束更新 README/AGENTS/产品与 MCP 契约、执行记录并给出准确 source SHA 与测试结果。

## 1. 目标结构：一个授权级别，多个独立 Actor

建议删除**整个** `admin/viewer` 产品角色模型，而不是仅隐藏 `viewer`：

1. 任何**有效且已明确授权**的 MCP 凭据均有同一 BoxWeave 管理能力：读取、创建、Exec、变更、删除等仍各自遵循现有操作契约。无单独只读等级，也不需要 `requireAdmin` 类型的角色检查。
2. **不同 Agent 使用各自独立、可撤销的 Actor**。手动 Token 代表自己的 Actor；OAuth Access Token 隶属于稳定 OAuth Grant Actor，刷新后 Actor 不变。保留 `actor_id`、`token_id`、原始 `operation_id`、`work_ref`、`agent_ref`，但后两者仅用于关联，不是授权或锁。
3. 不同 Actor 可按**同一 Layer 的受管资源权限**读取原始 Operation 以接手；仍不能以 B 的凭据冒充 A 的原始幂等操作。幂等唯一范围 `(layer, actor_id, key_digest)` 保留，原始 UNKNOWN 不可因读取成功就重放。
4. **读取和写入同样必须认证**；匿名/伪造/过期/撤销、跨 Project/Layer、外国 Box、Root 宿主操作继续拒绝。慢查询返回前保留 `privateReadReply` / `RequireActiveTokenID` 复查；已开始的原生操作不会因为 Token 撤销而被错误视作自动回滚。
5. 不引入“所有 Agent 共享一个 Token”的默认做法。取消角色不是取消身份归属、审计或最小宿主权限。

## 2. OAuth 与手动 Token：已选定方案 B（无权限 Scope）

### 2.1 已选定的新公开契约：OAuth 有认证，不设 Scope

- **Owner 选择方案 B**：OAuth 只负责取得授权身份凭据，不定义任何产品权限 Scope；删除 `admin/viewer` 角色、`mcp:admin`、`mcp:viewer`，也**不**新增 `mcp:manage`。BoxWeave 仅存在一种经用户明确批准的完整管理能力。这个设计没有否认 OAuth 本身的授权码流程，且与没有用户角色的 Actor/凭据管理相容。
- **ChatGPT/MCP 官方文档支持此方向**：[OpenAI Plugin Authentication](https://developers.openai.com/plugins/build/auth) 明确将受保护资源元数据的 `scopes_supported` 列为 optional，且给出 OAuth tool `securitySchemes: [{"type":"oauth2","scopes":[]}]` 的示例；[MCP 2026-07-28 Scope Selection Strategy](https://github.com/modelcontextprotocol/modelcontextprotocol/blob/main/docs/specification/2026-07-28/basic/authorization/index.mdx) 允许在未声明 `scopes_supported` 时客户端省略 `scope`。**这是公开协议及文档证据，不是真实 ChatGPT 无 Scope 登录验收证据**。
- 目标协议：受保护资源/授权服务器 metadata **不发布应用权限 `scopes_supported`**，401 挑战**不携带必需权限 `scope`**；OAuth 授权请求允许省略 `scope`，Token 不需返回权限 `scope` 字段；Tool 仍要求有效 Bearer 凭据，**不得将空 Scope 误当匿名**。如 SDK Tool 级 `securitySchemes` 适用，明确 `oauth2` 且 `scopes: []`；对当前 Go SDK/ChatGPT 版本逐项核验而非假定支持。
- **所有 OAuth 登录必须展示完整管理风险并要求明确用户批准**：读取/新建/修改/执行 Guest-root 命令/删除本 Layer 受管 Box。没有 viewer 降级选项，**绝不**因为旧 `mcp:viewer` 或 `mcp:admin` 被请求而静默赋予管理权限；这些旧 Scope 输入应明确拒绝。若平台自动附加 `openid`、`offline_access` 等标准请求，先观察实际客户端行为并评估，无理由不得把它们偷偷解释成管理权限。
- 保留**现有**浏览器授权、注册/DCR、Authorization Code + PKCE S256、严格 redirect/resource/origin 检查、一次性 Code、Access/Refresh 轮换、撤销/重放检测与本地审批。OAuth 的 `resource` 参数、Issuer/PKCE 验证和真实凭据有效期**不是 Scope**，绝不能一并删除。
- 手动 CLI `token-create` 删除 `-role`；默认签发管理凭据，保留 TTL、撤销和凭据保护。旧 Token、旧 Grant/DB 无需迁移，重建新 DB 后必须验证旧 Token 拒绝。
- **特别需要测试刷新**：OpenAI 的[自定义 MCP 连接说明](https://help.openai.com/en/articles/12584461-connecting-custom-mcp-servers-in-chatgpt) 指出部分 OAuth/OIDC 集成可能需要在授权服务器声明 `offline_access` 或等价能力才能持续刷新。MCP 规范又提醒资源服务器不应把 `offline_access` 当成资源访问 Scope。因此 **ChatGPT 无 Scope 登录和刷新是否均成功仍是实测门槛**；不能因为 OAuth 规范允许空 Scope 就宣称完整 ChatGPT 兼容 PASS。BoxWeave 已自行发放 Refresh Token，需实际验证 ChatGPT 能否接收并使用。
- 若真实 ChatGPT/OAuth SDK 无 Scope 实测失败，**不得悄悄回退到多角色**：记录原始请求、metadata、返回值、SDK/ChatGPT 客户端和失败原因，再讨论最小的协议互操作替代（优先只改变外部协议适配、不恢复权限体系）。

### 2.1.1 最小验收样例（目标，而非现有实现）

```json
{
  "resource": "https://example.invalid/mcp",
  "authorization_servers": ["https://example.invalid"]
}
```

以上受保护资源示意**没有 `scopes_supported`**；Client 在授权请求中不提供 `scope`，人仍在浏览器确认完整管理范围。Token 响应返回 `access_token` 和适用的 `refresh_token`，`scope` 无须存在。示意 URL 非可用服务器地址。

### 2.2 实际已锁定 SDK 的无 Scope 能力核查（2026-11-18，静态源码 PASS）

这不是升级依赖或替换 OAuth 库的理由。PR #153 当前 `go.mod` 固定：

| SDK | 当前版本 | 源码核查结果 | 精确依据 |
|---|---|---|---|
| `github.com/go-oauth2/oauth2/v4` | `v4.6.0` | **支持空 Scope 的授权请求、Token 请求、响应生成** | [`server/server.go` 授权校验 L174–214](https://github.com/go-oauth2/oauth2/blob/v4.6.0/server/server.go#L174-L214)；[Token 校验 L326–393](https://github.com/go-oauth2/oauth2/blob/v4.6.0/server/server.go#L326-L393)；[响应 L548–563](https://github.com/go-oauth2/oauth2/blob/v4.6.0/server/server.go#L548-L563) |
| `github.com/modelcontextprotocol/go-sdk` | `v1.8.0` | **允许没有请求/声明的 Scope，仍要求有效 Bearer Token** | [`auth/auth.go` L97–158](https://github.com/modelcontextprotocol/go-sdk/blob/v1.8.0/auth/auth.go#L97-L158)；[`oauthex/resource_meta.go` L43–46](https://github.com/modelcontextprotocol/go-sdk/blob/v1.8.0/oauthex/resource_meta.go#L43-L46)；[`oauthex/auth_meta.go` L46–48](https://github.com/modelcontextprotocol/go-sdk/blob/v1.8.0/oauthex/auth_meta.go#L46-L48) |

**具体核查事实：**

1. `go-oauth2` 的 `ValidationAuthorizeRequest` 使用 `r.FormValue("scope")`（缺省即空字符串），并未将空值判为无效；`ValidationTokenRequest` 在授权码交换时不强制 `scope`，刷新流程可以传空值。
2. 上游 `GetTokenData` **只在 `ti.GetScope() != ""` 时添加 JSON `scope`**。BoxWeave 当前 Token JSON 是自己手工拼装的（`internal/control/oauth.go`），整改时**仍需主动删除它无条件返回 `scope` 的代码**，不能以“SDK 支持”代替应用整改。
3. `MCP Go SDK` 的 `ScopesSupported` 字段为 `json:"scopes_supported,omitempty"`；`RequireBearerToken` 仅在 `opts.Scopes` 非空时要求 Scope，并且缺少 Bearer Header 时仍返回 HTTP 401。**Scope-free ≠ authentication-free**。
4. BoxWeave 当前 `internal/control/server.go` `TokenVerifier` 仍用 `t.Role` 填入 `TokenInfo.Scopes`，`operationPrincipal` 也要求 `admin/viewer`，`parseOAuthScope("")` 仍返回 `viewer`。**这是整改前源码事实；整改后的 PR #153 不再保持双角色实现**。
5. **验收分层：** `SDK_STATIC_SOURCE_COMPATIBLE=YES`；`BOXWEAVE_SCOPE_FREE_SOURCE_CANDIDATE=YES`；`ISOLATED_GO_SDK_HTTP_TESTS=PASS`；`REAL_CHATGPT_CONNECT_REFRESH_PASS=NOT_TESTED`。当前不需要切换 OAuth/MCP SDK 版本；完成代码修改后增加无 Scope SDK HTTP 测试，随后再用真实 ChatGPT 验证发现、同意、授权码、Token、刷新、撤销。

### 2.3 主要源码锚点（已核实）

| 范围 | 当前历史包袱 | 整改方向 |
|---|---|---|
| `internal/control/oauth.go` | metadata 两个 Scope、浏览器同意页 role、缺省 viewer | 无 Scope metadata / 无 scope challenge；明确全管理同意页 |
| `internal/state/oauth.go` | `parseOAuthScope` 返回角色、Grant/Code/Access 角色传播 | 去掉 Role 与应用 Scope 的存储/响应；保留 Grant Actor 与刷新/撤销事务 |
| `cmd/boxweave/main.go` | `token-create -role viewer` 为默认 | 移除 `-role`，统一签发 |
| `internal/control/operation_tools.go` | `operationPrincipal` 解析两角色 | 仅解析已认证 Actor、活跃 Token ID、真实 Project |
| `internal/control/box_tools.go` / `server.go` | `requireAdmin`、read-only 时允许 viewer | 保留认证/所有权，删除角色分支 |
| `internal/state/operation_query.go` | `Role` 参数、viewer 强制 actor-only | 统一 Layer 授权查询；保留防伪 Cursor/Actor 过滤的原有用例 |
| `internal/state/store.go` | DB role 字段和 CHECK、`IssueToken(role)` | 全新 Schema，角色字段与相关分支一起删除 |

**注意**：删除 `requireAdmin` 的角色判断不能顺带删除其使用的有效凭据、Layer/Project 与受管对象校验。需要从入口到 Store 验证真正的信任边界，防止修改引起未认证请求越权。

## 3. SQLite：允许全新基线，不保留迁移负担

- 当前 `SchemaVersion=3` 的 `credentials`、`oauth_codes`、`oauth_grants` 含 `role` 字段及 CHECK，且数据库表/索引有**严格指纹校验**。故简单改 DDL 而保留版本号不可行。
- 推荐建立新的、**不含 Role 的干净 Schema 基线**（暂记 `SchemaVersion=4`），在新临时数据库验证；**不实现 v3→v4 迁移、不兼容旧表、不保留双写和后备 Role**。保留版本与 DDL/索引指纹防篡改检查，使旧 DB **明确拒绝启动**，不要自动迁移、覆盖、清空。
- 用户允许放弃的是**开发环境业务数据**，不代表可以悄悄删除原生 Incus 实例、Bridge/ACL/Proxy、脚本策略、宿主 Root Journal 或原来 UNKNOWN 依赖。每个要重建的环境必须先由可授权读取得到独立 inventory，确认是否可以废弃；共享 L0–L5/未解决 UNKNOWN 只保留/隔离原状，**不因清库而解锁或重放**。
- 若必须重建：使用**全新 disposable Box + 独立 Incus Project + 新 SQLite** 首先验证；仅对用户明确指名且证明可销毁的旧测试目标执行重建。安装器的升级/rollback 能力不能拿“无需 DB 兼容”作为借口删掉，它仍然负责二进制升级失败恢复。
- SQLite 准入、Operation Journal、资源 claim、网络 fences、Exec 输出/GC、`recovery_audit`、未知状态和防重放均保留。只删角色存储与已证实无用的旧兼容逻辑；不删故障恢复证据机制。

## 4. 历史包袱的清理边界与分类

本次**可直接作为删除目标**（需要代码引用扫描、测试替代和验证）：

- `viewer` 角色、手动 token `-role`、OAuth 双 Scope/角色、角色相关 DB 字段、`requireAdmin`/只读角色路径。
- 单纯为 viewer 编写的正反角色组合矩阵、viewer 专属授权 helper、脚本中 viewer 凭据的创建/传递；**将仍有效的防越权测试改成无效/撤销/过期/伪造 Token、跨 Actor、跨 Layer、外国 UUID、丢失 ACK 及并发负例**。
- 对当前主线**无调用者且已被现有契约替代**的兼容别名、重复代码、过期开关/文档说明。删除前逐项证明唯一引用和实际行为，不因为文件名含 legacy/archive 就直接删除。
- `AGENTS.md` 里带日期堆叠的开发进度迁移到单独阶段 checkpoint/签收引用，使 AGENTS 保留小而稳定的规则；现行 Scope/README/MCP Catalog/Setup/Verification 契约统一，避免“新方案 + 老方案”并存。

本次**明确不属于无用包袱**：OAuth 的 PKCE/刷新/撤销；服务器对 host/origin 的验证；非 root Incus Project 边界；Root/Scriptlet 独立特权边界；UUID/incarnation、幂等、UNKNOWN、资源锁和 Native postcondition；无原子 File CAS 时的 `UNSUPPORTED`；native 升级/备份/回滚；TC-015/016 两 Agent 接手和外部 Git/Checkpoint 版本冲突；#151 的原始未归因问题。这些机制即使代码复杂也属于产品承诺或安全保障。

**历史证据策略**：Git 历史与原始 verification/签收文件保留，作为准确的“旧候选曾测过什么”，可另建清晰索引标 `HISTORICAL / NOT CURRENT CONTRACT`。不改写已发生的 PASS/FAIL、原始 CI/source SHA，也不为美观抹除未解决事故。

## 5. 建议执行批次（后续实施，不在本次文档提交运行）

1. **冻结目标与全面引用清单**：核对最新 HEAD、当前 MCP Tool/CLI 数量，逐项扫描 `viewer`、`mcp:admin`、`mcp:viewer`、`IssueToken(role)`、`Role`、DB DDL、`requireAdmin`、`OperationQuery` 以及测试/脚本。按“现行能力 / 历史证据 / 可删”标记；以已选定的**无应用权限 Scope** 目标为准。
2. **一次性一致改造**：同一 PR 同一逻辑变更集改认证、OAuth、CLI、Store、新 Schema、Operation 查询/所有 Tool 的调用、公共 MCP 描述，不做两套运行时或 Role 兼容；保留安全事务。Go 文件先 `gofmt`，通过 `make verify` 等检查后再提交，不直接提交未经格式验证的代码。
3. **测试/CI/文档闭环**：修复 Go 单元、真实 OAuth SDK HTTP、协议拒绝、Scriptlet/PRS 与 native S2/S3/S4 探针中 viewer 依赖；原本 viewer 安全负例必须替代为“无有效认证无法调用 Tool”及“已撤销 Token 不得读取/准入/派发”。更新现行 README/AGENTS/安全契约/S5 矩阵；不要篡改历史原生签收。
4. **一次性 native 验证与修复**：仅在 owner 批准的全新 disposable Box/Project 中测试从空库初始化、真实 ChatGPT OAuth 全管理同意/刷新、两个独立 Actor 接手、跨 Actor 非重放、撤销/过期、权限拒绝、IPv4/IPv6/Peer/Forward 所有受影响入口、Crash/UNKNOWN/原始结果找回。区分模拟/SDK、原生与独立验证证据，原失败保留，重复失败修复后复验。
5. **收尾**：最终检查线上 Schema 没有 role 字段、MCP/CLI 没有 viewer 路径，旧 Scope 被明确拒绝、无 Scope 授权及刷新被真实客户端核验、旧数据库不能被静默采用、所有受影响原生 gates 通过；同 SHA CI 通过、PR #153 保持 Draft，未经单独批准不合并、不发布、不动已有宿主。

## 6. 验收基准（本次整改要额外通过）

- **功能简化**：只有一种经过明确授权的管理能力；公开 OAuth/CLI/Schema 没有 viewer 或无用双角色参数；OAuth Grant Actor 与 Credential ID 仍不同且刷新后不变；所有独立 Agent 均可接手同 Layer 原始 Operation。
- **负例健全**：旧 `mcp:viewer`/旧凭据不能获得管理权限；匿名、过期、撤销、伪造、错误资源/Origin/Redirect、跨 Layer/Project/foreign Box 全部拒绝；二次证据读取和慢查询仍检查 Token 是否存活。
- **幂等与恢复**：不同 Actor 同 Key 不错误继承原始幂等身份；UNKNOWN 未确认不重放；原始 ACK/Native ID/Guest exit、部分失败、网络锁和 Root 屏障能按已有契约核查。
- **部署**：全新 v4 SQLite 初始化/重启稳定；旧 v3 数据库拒绝而不是自动删、自动升权或假装迁移；如需回滚二进制，文档明确允许整套 disposable 环境重新初始化、不承诺跨 Schema 数据回退。
- **测试真实度**：所有直接受影响的已签收 S2/S3/S4 安全/功能切面，在新代码和独立 native 环境复验；不把历史旧 SHA 的绿灯直接算成新实现 PASS；S5 TC-015/016、TC-019/020 等不豁免。

## 7. 已做决策与剩余风险

**A. OAuth：Owner 已选 B。** 当前目标是**无应用权限 Scope** 的 OAuth，非 `mcp:manage` 单 Scope；最终实现不再有 `admin/viewer` 角色。已经核对锁定的 `go-oauth2 v4.6.0` 与 `MCP Go SDK v1.8.0` 上游源码，都允许无 Scope；当前 PR 候选代码已完成角色链整改，但真实 ChatGPT 端到端授权/刷新尚未测试。必须先用 disposable server + ChatGPT 客户端对 `discovery → consent → code/PKCE → token → Tool → refresh → revoke` 给出独立证据，才能标记完全通过。

**B. 清理边界不变。** 执行**角色整链清理**和现行文档瘦身，再按静态引用与安全证据删旧 Helper/脚本。其他旧 S4 网络、恢复和 Installer 代码不得凭文件名批量删掉；需要先证明未被当前安全契约和原生验收引用。

**本文件同时是整改方案和阶段 1 的源码进度记录。** PR #153 已实现无 Role/Scope 的 Go 候选并通过隔离 Box 单元测试（首个完整 Go 通过的格式化源码提交 `a46d168d`）；新增明确缺省 Scope/旧 Schema 负例后仍须最终同 SHA 复验、CI 与独立 Native/真实 ChatGPT 签收。所有此前静态源码表格与旧角色锚点记录的是整改**之前**的状态，不应再作为当前代码判定。

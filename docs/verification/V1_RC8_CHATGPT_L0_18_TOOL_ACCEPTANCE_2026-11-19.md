# BoxWeave RC8 — ChatGPT L0 实机 18 工具验收与 P1/P2 修复跟踪（2026-11-19 PT）

**结论（安装版，原始提交 `cf577ee67e3b7aefd07957830774a11c6b6c1ec1`）：17 个工具在本次选定范围 PASS，1 个工具 `box_peer_set` BLOCKED；不是 18/18 形式签收，不是 RC 发布批准。**

- 真实 ChatGPT MCP：`boxweave L0 rc8`，版本 `1.0.0-rc.8-candidate`，SQLite v5、Incus 7.6、Project `user-1001`。
- 工作关联：`work_ref=rc8-l0-audit-20261119-1754`，`agent_ref=chatgpt-rc8-l0-audit-20261119`。
- 检验起点和终点：`host_status.ready=true`、`mcp_ready=true`；最后 `box_list=[]`。
- 此文件把**已安装旧二进制的实机证据**与**PR #153 后续代码修复/Box 内复测**严格分开。后续的源码提交不等于 L0 已热更新或宿主完成存储额度证明。

## 1. 全部 18 个工具的覆盖矩阵（原始 L0 安装版）

| 工具 | 本轮分类 | 最有力实测范围及限制 |
|---|---|---|
| `host_status` | PASS | 安装 commit/schema/Incus/ready 与存储警告 |
| `box_list` | PASS | 初始空、双 Box 分页、最终空 |
| `box_get` | PASS | UUID 精确读取，Stopped/Running/资源配置；非法 UUID 拒绝文案不准 |
| `box_create` | PASS | 两个低配 Box；同键同参数回原 ID，不同参数冲突 |
| `box_update` | PASS | 内存 512→640 MiB；无效 cpu=-1 安全失败但已先 ADMITTED |
| `box_start` | PASS | 双 Box Running，非法 UUID 拒绝 |
| `box_exec` | PASS | stdout/stderr、exit=7、非 root UID/GID、有限网络探测、同键重放 |
| `box_file_apply` | PASS | create+读回、O_EXCL 重复创建失败、replace UNSUPPORTED |
| `box_file_list` | PASS | 三文件游标分页、识别符号链接、non-snapshot 标记 |
| `box_file_read` | PASS | offset 分页、SHA256、Base64、路径与 symlink 拒绝 |
| `box_activity_get` | PASS | overview/issues/evidence、原始事件及资源声明、signed cursor |
| `operation_get` | PASS | 原始异步状态与 native ID / next_action / 终态 |
| `operation_list` | PASS | work_ref/agent_ref/action/state 过滤、keyset 游标 |
| `operation_output_get` | PASS | 非零退出码不被误认成命令成功、stdout/stderr 分页 |
| `operation_cancel` | PASS（安全拒绝） | 已派发的取消动作单独记录 FAILED，未取消原操作；成功取消未测 |
| `box_stop` | PASS | 双 Box Stop SUCCEEDED |
| `box_delete` | PASS | Running 删除拒绝；Stopped 删除 SUCCEEDED |
| `box_peer_set` | BLOCKED | 非法 UUID 返回资源锁类错误；**未对有效端点执行 ACL/PRS 授权/撤权** |

**边界：** PASS 只代表上述场景，不代表所有权限主体、网络协议、宿主故障或压力条件的完整证明。

## 2. 原始 Operation/Box 证据

- Box A：`862c17d6-cede-4ae0-9914-ffc0cd91b2cf`（1 CPU，内存 512→640 MiB，4 GiB，internet=false）。
- Box B：`b2e85c6f-7788-4286-bfc1-9d0eb809d71a`（1 CPU，384 MiB，3 GiB，internet=false）。

| 操作 | 原始 Operation ID | 终态 |
|---|---|---|
| Create A | `de50fa5e3612cf58d3ab3ada0e969378` | SUCCEEDED |
| Create B | `a4e6801f85e0988a54ada24aeebf72af` | SUCCEEDED |
| Update A 内存 | `ea84ca4025513d389dc32f797e5101cd` | SUCCEEDED |
| Start A | `c2ff50c3e979100a161f90efc7c3d063` | SUCCEEDED |
| Start B | `551c8e90ee81232bacd6565ac5e7232a` | SUCCEEDED |
| Exec A（退出码 7） | `7ef061bc416bf27a1e3d2ef80512b4c0` | Operation SUCCEEDED，Guest 失败 |
| Exec B（创建三个文件） | `8453f480e0a8a7e9c1f6f7f281198873` | SUCCEEDED，Guest exit=0 |
| File Apply A | `9116b7dca1522d30f7d9805c2e11bcba` | SUCCEEDED |
| Exec B symlink | `8c5839db9fa22978986701abe13a51ae` | SUCCEEDED |
| Exec B UID/GID 65534 | `f243e64cb17c5c27ec31bcd194630df7` | SUCCEEDED，权限拒绝输出正确 |
| A→公网 ICMP | `007ca7500456d5daf823e9b4115bda44` | Guest exit=1，单次 100% loss |
| A→B ICMP | `4054505b1cfe7aa08c95694bee908b08` | Guest exit=1，单次 100% loss |
| B→A ICMP | `9b24f14cd8f363f00e6390098a1ad987` | Guest exit=1，单次 100% loss |
| Stop A / Stop B | `af1ff4b9bd3e306b5213b45ded651960` / `340aa2a867563adf14056e9f38832e9c` | 双 SUCCEEDED |
| Delete A / Delete B | `e54154ab68fc960652fed1445d488b90` / `65fedad73b399de98c472424b7672422` | 双 SUCCEEDED |

预期失败（**安全保护工作正确，不能计作代码功能 FAIL**）：

| 场景 | 原始 Operation ID | error_class |
|---|---|---|
| Cancel 已派发原操作 | `8a54c513fc37688c0d051b90b7989c5c` | `ORIGINAL_ALREADY_DISPATCHED_CANCEL_UNSUPPORTED` |
| Delete Running Box | `ce4432d0602604ed576795dbc2475322` | `SAFE_NATIVE_PREFLIGHT_DENIED` |
| Update cpu=-1 | `0fda902231501191d4a7cd53e224624d` | `SAFE_NATIVE_PREFLIGHT_DENIED` |
| create 已存在文件 | `805ef036b13fbd56b68ac65d7a991006` | `SFTP_EXCLUSIVE_FILE_ALREADY_EXISTS` |

其他被直接拒绝的请求无 Operation ID：`box_get`/Start 非法 UUID 回 `operation not found`；`box_peer_set` 带非法源/目标 UUID 回 `original unresolved resource claim is still locked`；非法相对路径、symlink 等分别返回边界拒绝。

## 3. Agent 工作找回实测

故意不使用原来保存的 Operation ID，而以 `operation_list(work_ref,agent_ref,action=box_exec)` 找回 `f243e64cb17c5c27ec31bcd194630df7`，再用 `operation_get` 验原始 native ID / 终态、`operation_output_get` 验原 stdout/exit_code、`box_activity_get(view=evidence)` 验 ADMITTED→DISPATCHING→VERIFYING→SUCCEEDED、resource_keys、work_ref 和观察证据。两类列表的 cursor 分页未发现重复条目。

这证明**新调用重新发现持久操作**，但没有人工断开 Tunnel、杀控制器、断电/ACK-loss 注入；`attention_complete=false` 和 `NO_NATIVE_POSTSTATE` 等标志意味着 SQLite Activity 不等于原生完成确认。

## 4. P1/P2 根因分析（源代码级，不假定实机已修复）

**P1-Peer（原 BLOCKED）：** `internal/state/network_peer_scope.go:ResolvePeerAuditScope` 对不存在、非 OWNED、未 PINNED 或不同 ACL 的目标统一返回 `ErrResourceLocked`；`ReadCompleteNetworkAuditLedger` 还将不完整/冲突的网卡、地址 fence、绑定拒绝。对故意非法 UUID，锁类错误是**预期 fail-closed**，没有证据证明 SQLite 锁泄漏。最小源代码改进是**不改变准入、安全门禁及 Root/PRS**，只在 public MCP 层把这个不区分存在性的拒绝标记为 `PEER_SCOPE_UNVERIFIED_OR_LOCKED`，并说明只读恢复路径。合法端点和真实 ACL 尚待另行专项签收。

**P1-Host 存储（仍待环境整改）：** `host_status` 真正报告 `STORAGE_QUOTA_UNVERIFIED`、`selected_storage_quota_verified=false`、`selected_storage_separate_mount=false`、`selected_storage_space_checked=false`；它涉及宿主机/共享存储池及 Exec spool 的物理边界。**不得通过软件静默设置 verified=true、为消警告而修改宿主全局 Incus，或以 Guest 的 df/磁盘大小冒充 Host 的 quota 证明。** 需要运维通过独立的只读 Host 存储池/配额/容量证据或独立隔离池整改后才能关闭此项。

**P2-schema/error：** 旧 MCP tool input schema 暴露 `box_file_apply.mode`、`operation_output_get.stream` 为自由字符串，`box_exec.argv` 接受 null/空数组；旧 `box_get` 不存在 UUID 时透传了存储层 `operation not found`。旧无效 CPU Update 会先入账后原生失败。PR #153 源码改进仅收紧官方 SDK JSON schema、修正错误的 Box/Peer 类型文字和 Move-early Update 数值拒绝；**旧幂等 key replay 必须仍先检查原始 Operation**。源代码和安装版必须分别验证。

## 5. 修复/测试签收清单（后续进展填入，不得抢先声明 PASS）

- [x] 旧版真实 L0 原始证据收集，并删除全部旧测试 Box（`box_list=[]`）
- [x] 新版 Peer 预检错误语义保留 fail-closed，不创建 ACL/Root 副作用
- [x] 新版 Schema 枚举、Exec argv 非空和 Box 类型化错误修改已提交 PR #153
- [x] 新版 Update 数值范围预校验，放在 replayNative 之后、admitNative 之前
- [x] 新源码在**独立 Box** 内 Go 单测/官方 MCP SDK 验收（精确源 `d32ae049`，下文第 7 节）
- [ ] 新源码的真实 Guest L0 已安装替换后重新验证（本轮**不**更改现有 L0）
- [ ] 宿主存储/Quota 原生物理证据（P1，**未关闭**）
- [ ] 真实同 Project 合法 Peer/PRS/ACL + TCP/UDP/IPv6，须专项批准和独立验收
- [ ] 真实断线/UNKNOWN/OAuth 多身份场景

## 6. 安全清理与阶段建议

原始两盒的 Stop/Delete Operation 均为 SUCCEEDED，最终 `box_list=[]`；原始 `box_activity_get(issues)` 查询空但 `attention_complete=false`。**旧版阶段判断：可有条件进入下一阶段专项验收，不可发布 RC8 或声称全部 18 Tool PASS。**

本修复只应通过未合并 [开发 PR #153](https://github.com/vanishs/boxweave/pull/153) 继续；未经明确授权不更新物理 L0 的二进制、不改 Root/PRS、ACL、宿主池或 OAuth 密钥。

## 7. P1/P2 源码修复后的独立 Box 实测签收（2026-11-19 PT）

**源码测试结论：`P2_SOURCE_AND_BOX_TEST_PASS / P1_PEER_SAFE_ERROR_PASS / HOST_QUOTA_BLOCKED / NO_HOST_DEPLOYMENT`。** 对此前明确的 Peer 非法 UUID 安全拒绝语义已修复并在 SDK 测试中验证；**没有**把缺 PINNED、无 ACL 或跨 Project 的请求改为允许。P1 物理存储容量/Quota 与真实 Root/PRS 网络改动仍未验证，不得在本轮签署完全 P1 PASS。

- **被测最新源码**：`d32ae049f87823e7475d2c22b63cc05157188d91`（Draft PR #153，未合并、未发布）。
- **独立真实 L0 MCP Box**：`rc8-fix-test-1119`，UUID `98e49de3-177b-4b74-b142-472af288d950`，Ubuntu 24.04.5 / 2 CPU / 2048 MiB / 10 GiB，具有一次性 internet=true；`work_ref=rc8-p1p2-fix-20261119`，`agent_ref=chatgpt-rc8-fix`。所用原始 `box_create` Operation `eacef60178938a727e5731be3f76d15b` 为 SUCCEEDED。宿主仍运行**旧安装** `cf577ee67`，没有升级/替换服务。
- **Box 内 Go 1.26.9 / CGO / gcc / libc6-dev 原生编译环境**；在 Box 内 `git pull --ff-only` 到精确 `d32ae049...`，随后 `gofmt -l` 关键修改文件无差异 **PASS**，`go test -p 2 -count=1 ./internal/control ./internal/state ./internal/native` **三包 PASS**（分别约 15.660s、38.545s、0.010s）。`go build -o /tmp/bw-rc8-p1p2-build ./cmd/boxweave` **PASS**；旧 S3 SDK 客户端 probe 已可单独编译。**原始 Exec Operation** `c6974161837c56127f989b84bdbddb2c` 为 SUCCEEDED 且 Guest `exit_code=0`，stdout 同时保留测试源码 HEAD、GOFMT_PASS、TARGETED_GO_TEST_PASS、CANDIDATE_BUILD_PASS。这个构建只是 Guest 内编译，不构成已安装物理 L0 实测新二进制。
- **错误/模式最小修复**：`box_file_apply.mode=create` 枚举；`operation_output_get.stream=stdout|stderr`；`box_exec.argv` 非空数组；无效 Box 拒绝标记 `BOX_NOT_FOUND_OR_NOT_OWNED`；`box_update` 数值范围在历史 replay 之后、新 admission 之前拒绝；`box_peer_set` 对未 PINNED、未知 UUID、账本不完整和锁冲突都返回不泄露身份信息的 `PEER_SCOPE_UNVERIFIED_OR_LOCKED`；SFTP symlink/nonregular 错误保留原安全类别并添加可操作类型提示。
- **CI 已完成**：`d32ae049` 的 [required quality gate #44661569100](https://github.com/vanishs/boxweave/actions/runs/44661569100) **SUCCESS**；上一个生产代码 SHA `845d465f` 的 [S3 installed MCP + Git 原生 CI #44661285183](https://github.com/vanishs/boxweave/actions/runs/44661285183) **SUCCESS**，以及 [S3 native SFTP negative #44661285216](https://github.com/vanishs/boxweave/actions/runs/44661285216) **SUCCESS**。修正的是过时的 17 Tool 断言，现在校验 18 Tool 含 Activity。**这两个 S3 CI 并非针对最终 d32 的完整 native acceptance**；本节记录时最新 d32 的 amd64/arm64 artifacts 和 S4-A CI 尚未确认完成，不得提前记 PASS。
- **环境警告持续**：最新已知 `selected_storage_quota_verified=false`、`selected_storage_separate_mount=false`、`selected_storage_space_checked=false`；任何 Guest 磁盘容量或编译成功不能代替 Host pool/Exec spool/Project quota 物理证据。应另行独立只读验收（必要时获得明确主机配置授权）。
- **测试 Box 清理：PASS。** 原始 Stop `b5baf6f9cb883d9af44d2f6ec1109cba`、Delete `ba7fc83c723ea9b254232a010e61b26a` **均 `SUCCEEDED`**；最终 `box_list=[]`。Activity `issues` 在本轮 work_ref 下未列出任何条目，但 `attention_complete=false`，不应扩大成全局不存在未知状态的证明。

**后续验收门槛：** 在可控 Host/独立原生 runner 上做新二进制已安装的相同 MCP 请求对照；真实授权端点及 PRS/ACL 只能在专门授权的原生测试场景完成；独立核对物理池容量/有效 Quota 后才能关闭剩余 P1。不得把已签的 Guest/源码测试写成物理 Host 已修复。

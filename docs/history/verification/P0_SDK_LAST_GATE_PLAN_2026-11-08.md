# v1 开发前最后一轮 P0 SDK 合约验证（2026-11-08）

**任务范围：只验证既定的 20 个 MCP 工具需要的官方 Incus Go SDK 和官方 MCP Go SDK 行为，不新增 BoxWeave 功能。** 验证执行前创建本清单；后续真实结果见 [对应执行报告](P0_SDK_LAST_GATE_RESULTS_2026-11-08.md)。

## 用户确认的边界

**Cowsql leader 在内层 Incus 重复丢失应视为待解释的测试环境可靠性事件**：可能由现有 Box 嵌套环境资源/内核/超时导致，尚无因果证明。保留原日志、UUID、超时与服务状态；**不开发 BoxWeave v1 的 Cowsql/数据库修补、恢复服务、额外存储引擎或 Incus 专用守护程序**。开发中遇到 native UNKNOWN 时，采用已有的「按原任务及 native UUID 回读、无确认不重复发操作」原则。不能将环境失败误判成 SDK 缺陷，也不能因此悄悄扩大 v1 产品边界。

## 测试前划定的 P0 项及通过门槛

| ID | 官方 SDK | 验证内容 | 判定标准 |
|---|---|---|---|
| I-01 | Incus | Guest 文件 create、条件 replace/delete 和 symlink/hardlink/race | 使用官方 API/SFTP 实际观察独占创建、冲突时旧字节不变；找不到原子条件替换/删除则**明确缩窄产品承诺或阻塞，不能虚构 CAS** |
| I-02 | Incus | async guest Exec 接收原生 Operation ID、断开 reader、回读退出码、stdout/stderr，限时取消 | 不将 `RecordOutput=true` + writer 的阻塞误判为异步；真实进程/SDK 观察并验证取消之后进程状态；日志/输出恢复不可证明则列 UNKNOWN |
| I-03 | Incus | create/other native mutation 结果超时、UUID 归属、原始 Operation / readback | 利用已有一次实际「退出码 124 但容器已创建」证据以及幂等只读回读；不重复提交 UNKNOWN 创建，不编造 operation ID |
| I-04 | Incus | 一张共享 Bridge NIC ACL 的 IP/MAC 过滤、现有连接撤销和非原子多 NIC 失败 | 所有已测普通方向性与 SDK ETag 保留 PASS；新增隔离/异常测试使用单独受控实例/规则和只读回读；无法安全注入则标 UNKNOWN 而非 PASS |
| M-01 | MCP | 官方 SDK HTTP Streamable：2026-07-28 `server/discover`、legacy `initialize`、typed tools/errors、真实 Socket/HTTP | 必须实际使用 SDK client 与 loopback server 通过 HTTP，读取握手版本/工具结果，不把纯源码当验收 |
| M-02 | MCP | 真实 HTTP Bearer 用户隔离、viewer/admin、过期/吊销、并发同一个 Server、Origin/Host/body 限制 | 正反例 + concurrent calls 必须真实请求，服务端每次从 `req.Extra.TokenInfo` 读取；不把 annotations 视作权限 |
| M-03 | MCP | 客户端主动断线/取消与已接纳的后台动作/原始收据 | 在真实 HTTP 请求场景证明 handler ctx 与 operation ctx 的边界；**没有 v1 操作日志引擎时不称真正断线恢复 PASS**，只是确定 SDK 的安全用法 |

## 操作约束

- 首选已存在的 `bw-debug-0460-lab` 的**内层** Incus 7.6，外层不升级/重启，不新建外层 Box，不访问外层 Incus 管理 Socket。
- 实机变更只准新建明确前缀、原生 UUID 确认的内层临时实验资源；优先少量实例，避免再触发 Cowsql；发生 UNKNOWN 禁止重复创建，先恢复可读状态，保存原始证据。
- 两个官方 SDK 按经验证的 tag 测试（Incus 7.6.0、MCP 1.8.0，Go 1.26.10）；不以升级 SDK 或自行实现额外服务作为修复。
- **PASS、PARTIAL、FAIL、UNKNOWN 分开记录**，特别区分「SDK 原生合约已确认」与「v1 运行时黑盒已验收」。
- 结束时验证所有新建对象的原始身份，移除授权/监听/测试 Box，回读 native 清单；保留无敏感凭证的证据与离线备份。

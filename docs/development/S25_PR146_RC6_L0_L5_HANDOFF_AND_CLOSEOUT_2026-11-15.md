# PR #146 交接与收尾门禁（2026-11-15，L0～L5 实际 MCP）

**2026-11-15 actual L0 owner-run result:** `S25_DEV_BASE_PASS=SCOPED_PASS_WITH_DECLARED_LIMITS` for one supervised trusted operator only. Real physical-host L0 RC6 binary+unit+manifest integrity PASS; Incus 7.6 restricted identity and Project checked; native 8/8 PASS; 4 CPUs/12.52 GiB available RAM/~1.82 TiB reported free. L1 remains a different local binary, L0/L1 retain original UNKNOWNs; no host rollback, physical packet negative or quota enforcement is claimed. Full details [host acceptance](../verification/S25_PHYSICAL_HOST_DEV_BASE_POSTFLIGHT_2026-11-15.md). S3 #148/Create/Delete/#151 and S4 AppArmor/BPF/UID+GID/network/quota proceed; S5 full BH/TC remains pending. Current PR stays Draft until ordinary approval and final Actions; no RC6 rewrite.


**2026-11-15 owner-directed acceleration / current exit contract:** [S2.5 scoped early development base D-01–D-08; S3/S4/S5 reassignment](S25_EARLY_DEV_BASE_PHASE_REALLOCATION_2026-11-15.md) **supersedes the old requirement to solve all BH-019 platform features within S2.5**. Existing L0–L5 functional and isolated native/installed/Verifier evidence is accepted only for its tested scope. **Remaining S2.5 work is targeted non-destructive host-specific postflight (source identities including L1 mismatch, owner-approved fallback, resource envelope and existing UNKNOWN unchanged), not another broad L0–L5 lifecycle or new AA kernel workaround.** Current verdict `S25_DEV_BASE_PASS=PENDING_HOST_POSTFLIGHT`; stronger original BH-019 and full release gates stay OPEN.


**2026-11-15 同类 PR 集中处理：** 已保留 [PR #145](https://github.com/vanishs/boxweave/pull/145) 的原始 L1 RC4 UNKNOWN/AppArmor 243 事故记录及 O-036 风险索引，并把 [PR #149](https://github.com/vanishs/boxweave/pull/149) 的 RC5 横向源码审计（OAuth refresh X-01、Create/Delete 双事务 X-02/X-03、原生 Operation 收据 X-04/X-05）作为 **PR #146 后续单独验证与开发任务**。详细映射：[S25_PR145_149_CONSOLIDATION_2026-11-15.md](../verification/S25_PR145_149_CONSOLIDATION_2026-11-15.md)。**仅整合历史知识与待办，并未宣称这些问题已解决，也不把 Exec SQLite 原子终态改写成 Create/Delete 的实现。**

## 2026-11-15 继续开发后的收尾补充（覆盖下面旧的 mergeable=false 描述）

**Git 合并冲突已解决，不等于安全验收通过。** 通过无强推双父提交 `cc03620f5642fc2b15bf138520e9bc65b0fcbf6b` 合并目标 v1 的 BH-018 pilot #147；18 个相关路径中 14 个字节一致，另外两个 Go 文件保留 PR #146 RC6 typed options，工作流保留本 PR 分支守卫，同时保存 RC5 pilot 验收文档。当前 PR #146 为 **Draft、mergeable=true、security gates BLOCKED**；未触及 `v1` 和 immutable RC6 Release/tag。

已启用本 PR 分支的 GitHub 一次性隔离独立 Verifier 工作流（不借用现有 L0～L5 故障环境）。[v1 independent nested MCP LAB run 43769548904](https://github.com/vanishs/boxweave/actions/runs/43769548904) 在上一候选 SHA `5a2e75a596959fee6ae3eaff7c095e5dfefae0bb` **SUCCESS**，覆盖真实已安装 L0→L1→L2、两 Agent、原生操作收据、进程 SIGKILL/ACK 丢失及 IPv4/IPv6 和外 Project 负例；它明确保留 `S25_ALL_BH_AND_TC_FORMAL_SIGNOFF=NOT_YET`。同 SHA quality/S1/双架构/native BH018 27 case **SUCCESS**，但 [安装版完整 BH018 HTTP MCP #43769548925](https://github.com/vanishs/boxweave/actions/runs/43769548925) 首次 **FAIL** 于重启后的 fresh 126 Exec 回读，不能误签收；已提交原 SHA 的隔离重跑，需确认最终结果。本文更新本身前移源码 HEAD，须重新以最新精确 SHA 验收。

当前必须解决的不是普通六层生命周期：**BH-019 的逐层 AppArmor namespace/policy admin 243、子层 BPF EPERM、持久化双 UID/GID 映射分配、深层兄弟和跨 Project 网络包级负例、真实配额/Exec spool；BH-018 的间歇性 installed-MCP fresh readback 与 OAuth refresh stable-idempotency issue #148 仍需独立复验**。BH-019 分项阻断在 [#150](https://github.com/vanishs/boxweave/issues/150)，精确提交、历史 UNKNOWN 和 Actions 证据在 [S25_PR146_FINAL_GATE_RECONCILIATION_2026-11-15.md](../verification/S25_PR146_FINAL_GATE_RECONCILIATION_2026-11-15.md)。禁止未经验证的 policy/privilege fallback；只能正式 BLOCKED/PARTIAL。


**PR：** https://github.com/vanishs/boxweave/pull/146
**当前开发分支：** `fix/v1-s25-bh018-bh019-20261114`
**本交接前最新代码+现场文档提交：** `db2a5452fc37b726b7cced876be8db38d54ce927`。提交本交接文档会再次前移 HEAD，以实际 GitHub 返回值为准。
**受保护目标分支：** `v1`。PR 当前 **Draft / mergeable=false**，应先精确核对分支差异、冲突、审批和 CI，不要假设可直接合并，也不要绕过受保护分支规则。
**正式发布的测试资产：** `v1.0.0-rc.6` prerelease，commit `370e419c43f06b7488c7330ba61913ffec0e9f78`，amd64 SHA `0932ad434d2a1b113e2254f6ca02a218c2dff473f70d7ed3b2f45e93d0610174`。已发布的 immutable tag **不修改、不重签、不覆盖**；若有代码修改使用可追踪的新候选版本与 SHA。L1 单独仍安装早期候选 `89b0f891` / `85788181…`，其他五层才是正式 RC6 二进制。

## 一、已经做完，不要重新开发/重复证明

1. L0、L1、L2、L3、L4、L5 的独立真实 HTTP MCP 连接全部成功；`host_status` 每层返回 Incus 7.6、`ready=true`、`mcp_ready=true`；各层的本地 `user-1001` Project 不是同一个 daemon。
2. 已完成 L0、L1 和 L2→L3、L3→L4、L4→L5 实际 BoxWeave/Incus 创建、Guest 安装和 Tunnel 现场测试；L5 本次额外完成一个真实 Box 的 Create→Update→Start→root/nonroot Exec→Guest 非零退出 23 和 stderr→Stop→Delete，全部完成，Box 已清理。L5 测试时暂时打开的低层 Project 权限已恢复为 unset。
3. L2～L5 现使用 `lxc.apparmor.profile=unchanged` 继承同一个 L2 祖先 `(enforce)` Profile。内层独立 incusd 的 `INCUS_SECURITY_APPARMOR=false` 是针对 **daemon 本身** 的兼容设置；不是把内核或外层 AppArmor 关掉，也**不是**每层独立的 Policy Namespace。
4. L2～L5 末尾的 UID/GID 映射区间每深一层缩减约 100 万；安全、持久化、原子防冲突的 UID/GID lease allocator **还没有实现/验收**。只通过了 `security.idmap.isolated` 被安全拒绝的负例。不要把自动嵌套映射误报为分配器 PASS。
5. 相同名字 `user-1001` 的跨层别人的 Box UUID 或 Operation ID 实测 5 次全部 `operation not found`；这仅是所测查询的控制器可见性边界，不是全量威胁模型验收。
6. L0 两个原始 UNKNOWN：`4a5130ee9025e7ee39c1866d16ad1b39`（当前 L1 Box `cd1ea3d0-a24f-477b-b01c-08fb5acac53e` 上的 box_exec）与 `718b99438cfa8f1349895ade0e8dfa9a`；L2 另有原始 Start UNKNOWN `c6938fe17342ad02567ae11cf5642590`（Stopped `rc6-l3-lab-1115`）。本轮 L0 父层对 L1 的新 Exec 被 `original unresolved resource claim is still locked` 正确阻止，即使 L1 自身 MCP 依然健康。**不重放、不删除、不自动 unlock、不改历史收据**。
7. `v1.0.0-rc.6` GitHub Actions 构建是**提前供递归现场验证的非正式安全验收 prerelease**，不能倒推完整 S2.5/HOST_GO/BH-019 通过。

**本轮最完整证据：** `docs/verification/S25_RC6_L0_L5_CROSS_LAYER_MCP_FIELD_MATRIX_2026-11-15.md`。前序独立证据：`docs/verification/S25_RC6_L2_L3_INSTALLED_MCP_RECURSION_2026-11-15.md`、`S25_RC6_L3_L4_INSTALLED_MCP_RECURSION_2026-11-15.md`、`S25_RC6_L4_L5_INSTALLED_MCP_RECURSION_2026-11-15.md`。原始 S2.5 Gate：`docs/verification/BRIDGEHEAD_ACCEPTANCE.md`；源代码/内核根因：`docs/research/S25_BH018_019_PINNED_ROOT_CAUSES_AND_HOST_GATE_2026-11-14.md`；原始执行日志：`docs/development/EXECUTION_LOG.md`。

## 二、PR #146 还需要真正完成的工作（按优先级）

### P0. 先恢复 PR 真实状态，独立区分代码与验收文档

- 读取 `v1`、PR #146 HEAD/base、变更文件/流程、最新 Actions 和上述验收材料。PR 已有 **173 个 commits、57 个 files changed、Draft、mergeable=false**（统计为读取时，需以最新为准）。先判断冲突/旧分支差异是否会影响安全语义；只做审查过的版本同步，不得盲目大规模重写已通过的原生实现。
- 过去读到的已完成 PR SHA `17a380d37fad5f4fcaa0fcb47a40aa39df3de224` 的两项 BH-018 独立 CI 均成功：真实 installed HTTP MCP https://github.com/vanishs/boxweave/actions/runs/43764554419 ，native isolated https://github.com/vanishs/boxweave/actions/runs/43764554491 。**这不是本交接文档提交后的新 HEAD 通过证明**，必须重新读取精确 HEAD 的 Actions 才能引用。
- 生产和真实宿主的旧 Box/SQLite、Cloudflare、OAuth、历史 UNKNOWN 不作试验场；功能代码与 CI 试验在全新隔离 Lab/Actions 作业上进行。

### P1. BH-018 原生 Exec 终态，按真实安装版收口

- 已有原子 `FinalizeVerifiedExec` SQLite 事务（native ID、Project+Box+incarnation、Guest exit、stdout/stderr 路径、最终状态、恰好一条审计事件在一个事务中），**不要重复实现**。现有 GitHub installed MCP 真实 126/127、丢 ACK、SQLite WAL 切断和 SIGKILL 场景有成功证据，但正式最终验收仍标记 PARTIAL：需要独立 Verifier、当前同一提交真实受保护门禁和日志分类。
- 独立 Verifier 在自己新建的 7.6 disposable Lab 中，用官方 MCP Go SDK + 真实安装版 HTTP MCP 原始 operation/native 收据验证正常 Exec（0、23、126、127）、幂等重放、不确定中断前/后 COMMIT、SQLite all-or-none、stdout/stderr、单一审计、重连恢复、原生副作用 exactly-once，及新事务准入。强制 SIGKILL 仅对自有 disposable daemon，不动 L0～L5 共享环境。
- 分别为真正的“已确认原生启动失败”与“原生确认尚不确定”定义证据充足的终态分类，特别是 L2 首次 Start 触发 `POSTCONDITION_UNVERIFIED` 且独立 Incus log 有 AppArmor unconfined denial + BPF EPERM 的现场；**不要只因为事后 Stopped 或已有子层就把旧 UNKNOWN 改为 FAILED 或 SUCCEEDED**。如果缺真实原生 Operation/COMMIT 回读，继续 UNKNOWN 锁定是正确的 fail-closed 行为。必要时加入新 disposable 回归测试，然后与已有成功场景做反向检查。

### P1. BH-019 安全委派真缺口，不能用递归启动成功替代

- 内核与 Incus 研究已发现下一层 `apparmor_parser -a` 可能因为 `not policy admin` / 243 而拒绝；cgroup-device BPF EPERM 是另一个独立信号。启用 `restricted.containers.lowlevel=allow` + `lxc.apparmor.profile=unchanged` 是限定的**功能兼容路径**，不是独立 Policy Namespace / Policy Admin 的安全证明。不要使用 privileged、`unconfined`、关闭宿主 AppArmor、根 daemon socket 映射来“修复”安全目标；SELinux 暂缓。
- 首先定义/实现或清晰限定真正的 per-depth AppArmor policy-admin、stacking、能力门禁、BPF/cgroup-device，以及在确切 kernel+Incus 组合上父层、子层、兄弟和跨 Project 的正向/负向证据。若物理内核根本不支持安全委派，应**明确 BLOCKED 与 capability gate**，提出上游/内核支持或产品目标缩减的决策材料；未经批准不得把继承模式自动重命名为独立安全隔离 PASS。
- 实现并真实测试 UID/GID 持久化 lease allocator：从已验证的父 UID/GID map 池中进行原子范围预留、兄弟不重叠、开始/失败/停止/删除/UNKNOWN/崩溃/重启的安全状态机，池耗尽时拒绝；不能仅根据 L2～L5 剩余范围推断无碰撞。
- 附加网络/磁盘安全证据：不同层、兄弟及跨 Project 的真实 IPv4、IPv6 ULA/link-local/gateway/防伪造包级负例；ACL 生命周期与路由；`dir` / ext4/XFS project quota 实际条件，native disk 和 Exec spool 容量/压力真实测试。测试标签注明真机、隔离 Lab、CI 模拟的证据级别。避免因为性能/安装困难就默认隐式跳过。

### P2. 可复现的跨层自动化验收与合并

- 将上述 `Stage A–E` 跨层矩阵沉淀成**非破坏的先读型运行手册**，在一用即弃测试宿主/CI 自动化部署 L0～L5 正例；需要 Cloudflare/OAuth 的人工配置单独为 gate，工具安全检查拦截即如实记录，不得绕过。保护真实 L0 两个 UNKNOWN、L2 一个 UNKNOWN。
- 同一个最终代码+验收文档 HEAD 跑完整质量/S1 race/amd64+arm64、真实 Incus7.6、官方安装版 HTTP MCP、专门 BH-018 crash/ACK 和 BH-019 kernel/AppArmor/BPF/ID allocator packet负例；逐个核验 Actions 和独立 Verifier 同 SHA。保留原始日志、对象身份、失败分类、清理证据。
- 只在全部**事先既定**验收门禁满足，PR 可合并、审查批准后，才从 Draft 转可合并并入 `v1`；不能为通过 Actions 删除测试/放宽条件。若 BH-019 内核安全委派确为不可支持，PR 必须保持 Draft/blocked 或征得用户明确同意重新界定范围并拆分 PR，不能擅自签收。
- 如果要发布下一个 RC，使用新的不可变 tag、Release 资产、manifest/SHA256，不能改写已发布 RC6。L1 当前二进制与其他层不同，不可视为“同 SHA 六层完整验收”。

## 三、交给下一 Agent 的执行提示词（可直接复制）

> 继续完成 BoxWeave v1 分支 PR #146 的 S2.5 BH-018/BH-019 正式收尾。我已经使用真实接入的六层 MCP 对 L0～L5 做过实测，并把本轮证据写在 docs/verification/S25_RC6_L0_L5_CROSS_LAYER_MCP_FIELD_MATRIX_2026-11-15.md，以及本交接文档。
>
> **首先**读取当前 PR #146 的最新 head/base、Draft/mergeable、工作流、完整代码/文档变更和所有既有验证报告。不要以旧 PR body 或聊天概括代替最新原始证据。版本现场：L0 与 L2～L5 为 GitHub 正式 RC6 commit 370e419c；L1 仍为早期手工 RC6 候选 commit 89b0f891。所有六层 MCP 和 Incus7.6 已可用，L5 disposable 创建、更新、启动、root/nonroot Exec、非零退出、停止、删除已成功，测试 Box 和 L5 暂时放开的低层权限已清理。
>
> **接着**审查原代码并优先解决真正验收缺口：BH-018 已有原子 SQLite Exec 终态，主要缺当前同 HEAD 的独立安装版 HTTP MCP Verifier、真实故障与丢 ACK 严格复验以及有证据的 UNKNOWN 分类；BH-019 的 L2～L5 目前仅继承同一祖先 AppArmor enforcing Profile，不具备每层独立 policy-admin 正向证据，历史 parser 243 / BPF EPERM 需独立定位，UID/GID 原子持久化分配器、兄弟/跨 Project 隔离、IPv4+IPv6 安全负例和真实磁盘/Exec spool quota 仍未验收。
>
> 按“源码与验收矩阵审查→在全新隔离 Lab 实施必要修复→安装版真实 MCP 和 Incus7.6 自动化/故障/安全复验→独立 Verifier→同一最终 SHA 的 Actions→文档更新与 PR 门禁结论”推进；可以一轮改完明确的代码缺口，但每个高风险场景必须在自己的 disposable Box 分步跑，避免一长串安装/故障操作无法定位失败。原始操作收据、native ID、重启/清理状态必须记录。
>
> **约束：**不得触碰、重放、取消或强制解锁 L0 的两个历史 UNKNOWN（4a5130ee…、718b9943…）与 L2 的原始 Start UNKNOWN（c6938fe1…），不得对 L0～L5 共享 daemon 做 SIGKILL，不得关闭宿主 AppArmor、使用 privileged/unconfined、映射高权限 Incus socket，不研究 SELinux，不借跨层 MCP 绕过平台安全拦截。L0→L1 父层 Exec 仍因原始 claim locked 被拒绝，但 L1 自身 MCP 正常——要保留并作为恢复隔离的回归证据。
>
> **最终**在 PR #146 写回经过证据支持的 BH-018、BH-019 case-by-case PASS/BLOCKED，不把“成功递归 5 层”偷换为“逐层独立安全策略 PASS”。核实 PR 当前 mergeable=false 的原因并安全处理差异；只有所有约定门禁与独立 Verifier 真正通过才解除 Draft、合并 v1，并汇报提交、Actions、原始验收结果。若底层内核不支持 BH-019 所需 policy-admin，给出事实阻断与可行范围决策，不伪称完成或修改 RC6 不可变 Release。

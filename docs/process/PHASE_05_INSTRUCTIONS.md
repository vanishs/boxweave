# 阶段 5 提示词｜一个切片一个切片地实现完整产品

你是迭代开发 Agent。**本提示词每次运行只做一个最小可验收的 `TASK-xxx` 或紧密耦合的一小组任务**，而不是一次性完成整个阶段。每个切片必须同时交付代码、测试、文档和追溯关系。

## 启动检查

1. 确认独立新 Worktree、新分支，检查是否有未提交修改，不覆盖其他人的工作。
2. 阅读 `AGENTS.md`、`docs/INDEX.md`、`docs/process/PHASE_STATUS.md`、`docs/process/IMPLEMENTATION_PLAN.md`、`docs/product/REQUIREMENTS.md`、`docs/product/ACCEPTANCE.md`、`docs/design/ARCHITECTURE.md`、有关 ADR、`docs/verification/TEST_STRATEGY.md`、上一轮 `HANDOFF.md`。
3. 确认阶段 4 的**开发者架构/切片确认**已记录；若没有，停止启动新功能。
4. 检查旧分支知识增量：记录 SHA，对重要新增规则、Agent 笔记、行为变更做评估，只更新被认可的需求/知识，不自动合并旧实现。

## 选择本轮工作

从 `IMPLEMENTATION_PLAN.md` 选择最高优先级、前置任务已满足、范围清晰且能独立验证的最小任务。如果任务太大，先拆分 `TASK` 和相应测试，再挑选一个执行。汇报本轮 `TASK/REQ/AC` ID，并据此开发。

## 实现流程

1. 在动手前补齐该任务的验收条件与测试设计；重要知识来源记录到文档。
2. 只实现满足该任务需求的最小正确方案；以清晰边界和可读性优先。
3. 同步新增/修改自动化测试及必要的集成/端到端测试。
4. 真实执行适用测试与静态检查；明确区分已通过、失败和未运行项目。
5. 更新 `docs/process/IMPLEMENTATION_PLAN.md` 中任务状态及阻碍，`docs/verification/FEATURE_MATRIX.md` 中 `REQ/AC/TASK/测试` 的关系。
6. 如发现产品或架构需要变化：小规模、已授权的工程细节可更新 ADR；重大需求/行为变化须写明方案并交开发者裁定，不能静默改变产品。
7. 更新 `docs/INDEX.md`、相关知识、`LEGACY_SYNC.md`、`OPEN_QUESTIONS.md`、`PHASE_STATUS.md`、`HANDOFF.md`；确保文档与本轮代码提交一致。
8. 只在新分支进行合理提交，不推送、部署、重置或修改旧项目。

## 每轮的完成定义（DoD）

- 该 `TASK` 的验收条件明确、实现完成、适用测试实际通过，或者如实标记未完成。
- 对应的 `REQ/AC` 有证据与状态，架构变更及重要发现有记录。
- 代码、测试和文档属于同一轮可审查提交；无隐藏的未完成项。
- 提供可复现的执行命令、提交 SHA 和下一轮建议任务。

## 阶段 5 何时整体结束

当阶段 2 确定的高优先级需求均已满足验收，剩余事项被明确分类为延期、低优先级或受阻，且 `FEATURE_MATRIX.md` 能说明覆盖情况时，方可建议进入阶段 6。**每次运行仍只处理一个切片**；不能自行跳过未解决的高优先级风险。

最终报告本轮 `TASK/REQ/AC`、具体改动、真实测试结果、文档更新、新分支提交 SHA、旧分支知识同步情况、剩余风险与下一任务。写入 `HANDOFF.md` 并停止本轮。

# 阶段 2 提示词｜基于知识重新定义产品与需求（不写代码）

你是产品与需求分析 Agent。你的目标是**从旧项目的知识证据出发重新定义产品**，而不是沿用旧代码所暗示的功能结构。**本阶段只产出和修订文档，不能写实现代码。**

## 进入检查

- 确认当前位于独立的新 Worktree、`rewrite/v2`（或项目已记录的对应分支）；严禁修改旧 Worktree。
- 阅读 `AGENTS.md`、`docs/INDEX.md`、`docs/process/MASTER_PLAN.md`（存在时）、`docs/process/PHASE_STATUS.md`、所有 `docs/knowledge/` 基线与上一阶段 `HANDOFF.md`。
- 核查 `docs/knowledge/LEGACY_SYNC.md`：如旧分支已变化，登记变动，分析是否影响本阶段的产品知识，不自动合并代码。
- 若知识基线不足，记录具体缺口并优先补证据；不能凭空补齐“用户真实需求”。

## 思考任务

1. 用用户目标和实际使用场景描述产品价值，不以现有页面、API 或模块作为需求清单的唯一依据。
2. 对旧功能逐个做 **保留 / 改善 / 移除 / 待验证** 判定，说明为什么；允许提出新增功能，但必须与核心目标关联。
3. 区分已确定需求、从代码推断的需求、未经验证的想法和被废弃的历史决策。
4. 对关键路径、异常场景、权限边界、数据一致性、可靠性与操作体验给出可检验的行为定义。
5. 建立稳定标识：`REQ-xxx` 对应需求，`AC-xxx` 对应验收准则；每个高优先级 REQ 必须关联至少一个 AC；尽可能引用 `SRC-xxx`。
6. 明确非目标与范围边界，防止后续 Agent 以“顺手优化”为名不断扩张产品。
7. 准备必须交开发者裁定的选择，提供简短的利弊比较与推荐，不替开发者捏造业务事实。

## 文档交付物

```text
docs/product/PRODUCT.md       # 用户、目标、成功标准与核心使用场景
docs/product/REQUIREMENTS.md  # REQ ID、优先级、状态、来源、依赖
docs/product/SCOPE.md         # 保留/改进/移除/新增/不做，附理由
docs/product/ACCEPTANCE.md    # AC ID、输入/前提、预期结果、边界
```

同时更新：`docs/INDEX.md`、`docs/knowledge/OPEN_QUESTIONS.md`、必要的 `SOURCE_REGISTER.md` 和 `LEGACY_SYNC.md`、`docs/process/PHASE_STATUS.md`、`docs/process/HANDOFF.md`。避免在多个文件维护互相矛盾的同一要求；以 `REQUIREMENTS.md` + `ACCEPTANCE.md` 作为权威定义。

## 验收标准

- 关键使用场景能映射到清楚的需求和可执行的验收条件；标明暂时无法验证的行为。
- 对原项目主要功能都有去留判定，且提供依据。
- 不确定性与产品取舍被诚实记录；重大决策有待裁定清单。
- 有一份**可以供开发者阅读并做出决策的简短产品评审摘要**。
- 文档可从 `docs/INDEX.md` 导航到；提交仅影响新 Worktree。

## 停止门禁

**必须在本阶段结束后等待开发者确认产品方向、功能范围和关键行为定义**。即使自认为结果正确，也不得进入阶段 3、决定新技术栈或写应用代码。

最终报告：文档路径、最重要的新增/删除/改变、尚需开发者裁定的 3—5 个问题、相关 `REQ/AC/SRC`、新分支提交 SHA、阶段门禁状态。将摘要写入 `HANDOFF.md`。

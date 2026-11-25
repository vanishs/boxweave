# BoxWeave task entry (portable)

Apply BDH to every BoxWeave task, not every message: usage, support,
installation, diagnosis, design/research, docs, development, release and
BDH itself. Use the smallest workflow that fits the goal and risk.

1. Read [AGENTS.md](../AGENTS.md) and [asset index](INDEX.md) at the actual
   repository ref; load only relevant contracts. Recover original task,
   target and operations before changing anything.
2. With a checkout, use bwdev work plan --kind KIND --query QUERY, then
   existing start/resume/task/verify/finish for development. Without one,
   read these files through authorized GitHub access; a separate control
   checkout is optional, never install tooling on a target Host just for BDH.
3. Continue safe ready work, validate outcomes, and at material failures,
   corrections, decisions, handoff and completion make a learning judgment.
   Reuse first; fix a bounded gap or flag/defer it with evidence.
4. Keep actual outcome separate from advice, task notes and CI/native gates.
   Ask for exact owner authorization where required. Preserve original
   UNKNOWN operations; never replay, delete or weaken safety to continue.
5. Persist sanitized material records in authorized Git/PR storage.
   If inaccessible, return an explicit pending portable record; do not claim
   persistence. Small read-only guidance needs no Box, PR or new asset.

For a supported Agent surface, configure once to load this entry before
BoxWeave work. Repo merge alone does not install such a trigger. If entry
loading cannot be guaranteed, use the explicit read fallback and report it.
See [remote entry configuration](examples/REMOTE_ENTRY.md), [workflow](README.md) and [delivery plan](../docs/development/BDH_DUAL_LOOP_DELIVERY_PLAN.md).

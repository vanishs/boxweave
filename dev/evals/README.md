# Harness evaluation

Run `python3 dev/bwdev --json eval`. The versioned [cases](cases.json)
are deterministic behavior regressions, also called by the offline
bwdev test suite. They cover installer context (H5 learning),
superseded/current and missing CI (BDH-L012), and dependency blocking.
They test output properties, not language model quality.

For comparative Agent evaluation, select historical tasks with a frozen
starting SHA, sanitized task text, exact environment recipe, held-out
solution and independent observable acceptance oracle. Keep model,
configuration, authorized tools, task time/call budget and starting
environment equal. Compare ordinary workflow against BDH with repeated,
separately isolated runs. Do not expose expected patches. Record success,
false completion, interventions, duplicate effects, handoff recovery,
test/rework count, tool calls, context/token cost and elapsed time where
available. Missing metrics remain unknown; report sample size and failure
cases. Preserve private evidence when needed and only sanitized summaries
in Git. Never loosen the oracle or product gate to improve a score.

A changed harness should add or update a relevant behavior case when
useful and run cases plus targeted tests before CI. Mandatory product CI
remains independent. No comparative Agent benchmark, external adapter
integration score or productivity gain is claimed yet.

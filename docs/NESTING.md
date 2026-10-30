# Container nesting

Enable nesting when a box must itself run containers, such as a BoxWeave server. Configure the outer instance from its parent management connection; a guest cannot configure itself.

| Entry | Behavior |
| --- | --- |
| Web create: Allow containers inside the new box | Enabled for a BoxWeave server, Disabled, or Provider default; independent capability checked at preview |
| MCP create_container.enable_nesting | Optional boolean on owned streamed endpoints; omission preserves Provider behavior; Podman unchanged |
| MCP boxweave_nesting inspect | action=inspect, logical=exact managed username; config read does not verify execution |
| MCP boxweave_nesting plan | action=plan, logical and nesting boolean; returns target and expected identity/config |
| MCP boxweave_nesting apply | Repeat logical/nesting and returned expected; confirm equals full instance name; reject stale plans; no restart |
| boxweave.sh doctor | Local container detection, parent/effective nesting unverified |
| boxweave.sh box diagnose NAME | Exact instance log read; proc mount denial guidance |

Older endpoints reject explicit unsupported nesting instead of silently enabling Podman. The privileged tool is installed only by the owned streamed root installer, not by a plain application update.

On the parent host, inspect its verified exact project/instance: sudo incus config get OUTER_INSTANCE security.nesting. After reviewing the security-boundary change, set sudo incus config set OUTER_INSTANCE security.nesting=true. Replace OUTER_INSTANCE with a verified identity; do not infer it from guest hostname.

Restart is a separate explicit action with workload interruption. Our same-instance tests required outer restart for both enable/disable to affect nested starts. Restart the configured outer box before relying on changed capability, then check actual child MCP identity/workload output. BoxWeave never silently restarts it.

Proc permission denial suggests nesting or other confinement, not a proven parent value. Preserve interrupted instances and inspect provisioning; starting an interrupted create does not necessarily finish account/SSH installation. A timeout never authorizes replacement creation.

[Evidence and limits](verification/NESTING_VERIFICATION.md), [root boundary](design/ADR/0008-independent-nesting.md).

---
name: box-development
description: Choose an authorized development environment and preserve native ownership.
---

# Box development

## Reuse-first iteration
Use the existing authorized isolated workspace if appropriate; local checkout, VM and CI are valid. Do not create or delete a Box just to follow a workflow. Verify installed versions and permissions.

## Bounded guest source transfer
Use supported guest file/exec mechanisms and preserve original receipts. Verify exact destination and content digest. Optional `python3 dev/transfer_plan.py --help` provides a local planner; it is not authorization.

## Direct MCP orchestration
Use actual tool discovery and exact resource identities. Resolve an uncertain effect through the original Operation before further dependent work. Bound observations, avoid blind replay, and report unavailable capabilities.

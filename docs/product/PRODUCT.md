# Approved product direction

Approved by the developer on 2026-10-28 (SRC-026), following the initial product proposal. [Requirements](REQUIREMENTS.md) and [acceptance](ACCEPTANCE.md) own behavior; [scope](SCOPE.md) owns selected features. Phase 4 now has a partial minimum-system implementation; release/live acceptance is not claimed. See ../verification/VERTICAL_SLICE.md.

BoxWeave serves one trusted operator using coding Agents for remote development, including explicit project/environment binding. It must make the selected environment, trust checks, actual execution result and uncertain outcomes understandable and recoverable.

First-release client matrix: Windows-hosted Codex and ZCode(GLM), each using Windows native SSH or WSL SSH. The developer wrote 'WSL SSL'; this is interpreted as WSL SSH from the preceding transport choices, not a request for a new SSL transport. WSL distro is selected explicitly. Running Agent CLIs inside WSL is a separate future profile, not silently included. Other platforms/Agents are later extensions.

Basic Web, Tenant network writes, port publication, Prompt Graph and model wizard are included in the first-release scope; recursive deployment is deferred. Scope describes basic operations and exclusions. ZCode model access is distinct from MCP infrastructure access; model credentials remain client-owned. No migration from the old implementation or universal support claim is implied.

## User scenarios

| Job | Requirements | Criteria |
| --- | --- | --- |
| Enroll a chosen Agent/transport without losing settings, then diagnose it independently | REQ-001, REQ-002, REQ-003, REQ-016, REQ-017 | AC-001, AC-002, AC-003, AC-016, AC-017 |
| Bind/select a project, verify target identity and execute authorized work | REQ-004, REQ-005, REQ-006, REQ-007, REQ-010 | AC-004, AC-005, AC-006, AC-007, AC-010 |
| Install/update and understand actual readiness and repair limits | REQ-008, REQ-009 | AC-008, AC-009 |
| Use basic Web to observe/manage selected resources without exposing credentials | REQ-011, REQ-018 | AC-011, AC-018 |
| Create a shared Tenant and grant/revoke basic Tenant-wide ingress | REQ-012 | AC-012 |
| Publish/remove a selected port with explicit authorization and honest connectivity evidence | REQ-013 | AC-013 |
| Save/copy/export prompt context without implicit execution | REQ-014 | AC-014 |
| Configure ZCode GLM access using supported client-local setup | REQ-015 | AC-015 |

Success means the selected profiles and basic features pass their actual acceptance paths, including failure/recovery cases. Fake-provider/renderer tests alone cannot certify real Agent integration, isolation, route reachability or provider capability. Model generation requires a separately authorized provider request; no paid inference is run during design.

One trusted operator is not an unprivileged multi-user platform. Broad upstream provider privileges remain visible; trust failure stops execution, unknown writes are not retried blindly, and recovery is bounded. Hosted service/billing, automatic workflow execution and recursive deployment are excluded. The phase 2 market-demand gaps remain historical; developer preference is now explicit, not inferred from old code.

# Nesting configuration and live-update verification

Source: SRC-103. Scope: REQ-003/008/009/011/018/019, AC-003/008/009/011/018/019; TASK-017; [ADR-008](../design/ADR/0008-independent-nesting.md).

## Actual baseline (T, source 0.15.1)

The streamed installer ran in bw-e2e-server-1029-container (10.206.30.13). Its peer bw-e2e-client-1029-container (10.206.30.14) shares native Network Tenant bw-e2e-1029. The peer used generated SSH config and real MCP initialize/initialized, tools/list58 and tools/call, never direct Incus execution as workload proof.

The original bw-e2e-hello-1029-container was retained after creation stopped at a mount failure, before account/SSH provisioning. A separately recorded complete creation after correcting the environment produced bw-e2e-hello-nesting-1029-container,10.203.0.3, createdAt1793333498. The peer verified exact username/hostname and Python stdout hello world, exit0; test script exit0.

Comparisons reused that successfully provisioned instance without reinstalling software, changing product code or recreating it. The operator changed/reported the parent setting; the driver did not independently read parent Incus configuration.

| Operator-reported setting and outer lifecycle | Actual observation |
| --- | --- |
| true, outer restarted | Same child Running; MCP identity and Python hello world exit0 |
| false, outer not restarted | Child execution continued; restarting only that child still yielded MCP hello world exit0 |
| false, outer restarted | Child start failed; kernel denied mounting proc |
| true restored, outer not restarted after false boot | MCP start failed; latest LXC log again denied mounting proc |
| true restored, outer restarted in earlier comparison | Same child recovered and MCP Python exit0 |

This supports an outer restart for the tested mount-capability transition, not a universal claim about every Incus live-updated field. Configuration and effective execution differ. A BPF permission error also appears in failing logs; it was not independently isolated.

Retained controller directory: C:/Users/vanis/Downloads/boxweave/.runtime/e2e-install-1029. Evidence: accepted-peer-result.json, nesting-false-peer.json, nesting-restored-peer.json, nesting-live-false-peer.json, nesting-live-false-after-inner-restart.json, nesting-false-reboot-before-live-enable.json, nesting-live-enable-peer.json, nesting-live-enable-server.json. Peer directory: /home/bw-e2e-client-1029/boxweave-peer-smoke. Last live-enable test leaves the child stopped; no automatic recovery is claimed.

## Implementation 0.16.0

The owned streamed endpoint adds optional create_container.enable_nesting and boxweave_nesting inspect/plan/apply. Older endpoints reject explicit unsupported selection before dispatch; omission preserves upstream behavior. Podman is unchanged. Web freezes the choice in the plan/receipt without claiming child execution.

Only the reviewed root streamed installer installs the helper and exact fixed-argument sudo rule. The isolated script reads bounded JSON stdin and changes only security.nesting on exact default-project managed Ubuntu instances. It rejects protected/unmanaged names, stale creation/UUID/config identity and wrong confirmation. No restart, deletion, package installation, profile/SSH/firewall/parent credential changes. Current credential-authorized get_container precedes helper access. Guest installers cannot configure their own parent.

Installer/doctor report parent policy and effective execution unverified. Exact box diagnosis identifies proc mount denial as possible parent nesting or other confinement, never proof of the parent's value.

## Validation and remaining checks

Assigned non-root development source: /home/boxweave-tenant-dev/e2e-default-1029/source. Targeted test_nesting.py:9 tests, exit0 (0.499s); existing test_online.py:8 tests, exit0. Contract/fixture tests are not privileged live acceptance. Full non-root .venv/bin/python scripts/validate.py --browser passed:209 tests in66.936s, frontend build and all seven Chromium fixture workflows, ten steps exit0. Report: /home/boxweave-tenant-dev/e2e-default-1029/source/.runtime/validation/76c3479a7c2846b5bf127808c8b64f34/result.json. No live MCP/deployment is performed by that runner.

U: published helper/sudo installation, new MCP plan/apply, independent-create integration and interactive Web selection. Next: install verified release in the dedicated server under existing test authorization; inspect/plan the exact child through peer MCP; apply only a specifically reviewed change and verify unchanged identity/no automatic restart. Do not change the original Ubuntu parent, unrelated/core instances or public routes. No bare-metal, universal confinement or paid-model acceptance follows.

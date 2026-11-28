# P3 R4 caller and witness disposition

Inventory baseline: 52f1ce9013208470f10b07c8cf26e277cd46ac8a.
This is a source classification and handoff, not independent acceptance.
See R0 inventory and [retirement inventory](RETIREMENT_INVENTORY.md).
Retain all H obligations pending owner instruction after Support.

| Caller or oracle | Current disposition |
| --- | --- |
| main.go / cli_contract.go | 45 generic names with typed flag subsets; six fixed internal dispatch entries; reviewed recover subparsers. Current serve uses explicit fresh assertion; no new CLI or MCP tool. |
| local.go generated unit, scoped guard/lifecycle units | Current fixed commands and explicit loopback scope; retain all three unit identities and dependencies. New unit directory mode follows intended metadata contract even with installer umask; existing directories and private files unchanged. |
| install.sh | Current fresh reviewed plan/apply/guided wrapper, exact installed status and Doctor postflight; five old selectors refuse before candidate access. R3 failed postflight typed evidence retained. |
| uninstall.sh / uninstall_core.py | Current full/Scoped/offline/purge review and original resume remain; old archive/manifest recognition is not authority to adopt or mutate an old install. P4 trace transitive historical readers and unreachable bodies. |
| native-plan/bootstrap/prereq/prepare/migrate (16 entries) | Retain reviewed Incus dependency lifecycle; not BoxWeave binary upgrade. |
| upgrade/rollback/install-commit | Explicit refusal. N2/Forward and S25 active/offline negatives are current requirements; successful old binary upgrade belongs to historical evidence/#170 scope, not a current release gate waiver. |
| native-peer-audit / native-peer-purge-plan / native-peer-root-unit-stage | No current dispatch; explicit unsupported-command/no-source-creation regression. Existing native_peer_audit_test.go missing-DB tests cannot establish a live reader witness. Preserve them for P4 disposition; no H test changes or replacement proof. |
| local Recover / MCP operation_recover | Local settle-only review, separate Scoped MCP lifecycle recovery; GC exact stream review remains local. Historical bind/pin syntax retained for receipt readback, not new local authority. |
| N2 script / matrix | Entry requires Scoped selector and refuses historical offline-retirement lane. 29 original current lanes retained; old branch bodies remain unreachable in this candidate. |
| S25 outer/inner scripts / agent | PR and controlled release callers set P1B_S25_SCOPED=1; inner agent forwards it. Both entrypoints now refuse omitted/old mode before setup. Keep original current marker checks; historical branch bodies deferred P4. |
| Old guided-upgrade/S4/S3/S5 workflows | Historical branch-specific or frozen source evidence, not current PR #191 producer. Historical S25 workflow can be manually dispatched but current script rejects its absent Scoped selector; P4 reconcile triggers and frozen-source consumers. |
| s5b1_installed_closeout / s5b_peer_production_native / s5b_rc10_residual_native | Historical witness helpers, no current Scoped S25 dispatch. P4 explicit old-oracle/caller disposition; no silent PASS. |
| MCP catalog / contract / network / user guide / dependency lifecycle / UNKNOWN guide | Updated to fresh-only schema 7 candidate; historical version-specific guidance and original evidence retained. |

Remaining P4 scope: transitive helper/body removal only after equivalent
witness review, historical workflow triggers, marker crosswalk and release
plumbing. H and C04 remain open. Do not use this inventory as proof that
all historical test requirements have been discharged.

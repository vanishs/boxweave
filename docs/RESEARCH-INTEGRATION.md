# Applying the nested LXC research

The development goal remains simple: install BoxWeave on Ubuntu, install it
inside an ordinary LXC, and exercise the nested workflow. Research artifacts
are useful evidence, but are not prerequisites that every user must reproduce.

## Incorporated into product code

| Observation | Product change |
| --- | --- |
| A configured default Incus remote can address another host | Bootstrap, host-key synchronization and CLI inspection explicitly select local Incus |
| Instances may exist outside the current project | Empty-host initialization checks instances across all projects before initialization |
| A service may be activating while its core containers initialize | Readiness accepts activating/reloading, bounds individual waits and continues observing the API |
| Clones can inherit volatile UUIDs | Host-key updates recheck exact identity, configuration, running state and IP before writing |
| Provider versions and BoxWeave versions are different | Dedicated VERSION file and management CLI; component versions carry explicit labels |

## Extracted into a reusable tool

`boxweave.sh` is a small source-checkout launcher for `server/cli.py`. The
installed command is `boxweave`. See [CLI reference](CLI.md). Reusable checks
include version discovery, service states, exact-name instance inspection and
namespace-aware nested diagnostics. They remain separate from the MCP stdio
transport so management output cannot corrupt MCP messages.

## Kept outside product logic

Machine addresses, experiment names, archive paths/hashes, fixed UID maps,
physical PCI masks, cold-backup procedures and package-upgrade rehearsals belong
to the experiment's records or an operator-specific runbook. They are not
hard-coded into the installer or required before ordinary nesting tests.

No generic rule requires `raw.lxc` to be empty: it can contain intentional
protective mounts. Likewise, an internal bridge address, a child `unconfined`
label, an active route, or a single successful API probe is not a complete
reachability or confinement verdict. Unknown capabilities stay explicitly
unverified instead of being turned into automatic repairs.

## Future work, not implemented

- Add a small nested integration scenario using ordinary Ubuntu images: outer
  installation, inner installation, one disposable workload, observable results
  and scoped cleanup. Reuse compatible installed Incus; upgrade only for a
  demonstrated requirement.
- Fix exact-name addressing in the upstream provider MCP. The local CLI offers
  literal inspection, but does not patch the provider binary.
- Add provider adapters behind the CLI's observation functions as another
  provider is actually implemented. Keep JSON field changes documented and
  versioned; avoid speculative lifecycle commands or broad plugin frameworks.
- Add opt-in, separately authorized backup/restore and parent-side policy
  diagnostics if needed. Keep mutation and privileged host capabilities separate
  from the read-only doctor.

The behavioral tests use fake commands and temporary files. They do not install
packages, restart real services, or constitute a clean-machine end-to-end test.

## Validation on 2026-10-22

The source was copied into `/tmp/boxweave-code-review-p2sdh488` in the existing
`boxweave-e2e-container` and tested as the non-root `boxweave-e2e` user through
Platform MCP. `python3 -B -m unittest discover -s tests -v` passed all 40 tests
(exit 0). Bash syntax checks for `install.sh`, `publish.sh` and `boxweave.sh`
passed (each exit 0). `bash boxweave.sh version --json` returned `0.1.0-dev`
(exit 0). The full test output is retained there as `test-results.txt`.

The test container has no Incus executable. Real daemon inspection, installation
of the revised CLI on the Ubuntu host, and the nested installation/workload
scenario were not performed by this validation. No host services or experiment
container lifecycle settings were changed.

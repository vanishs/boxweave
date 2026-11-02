# ARCH-0A — executable boundary and mutation-route inventory

**Source baseline:** dd9e78bef92d2f8b4c5aac7da45da9c317ce2404 (v0.29.0). **S only**: audited source; no privileged service or real Box action. [Stage owner](../design/ARCHITECTURE_CONSOLIDATION_GATE.md) · [Master](../design/FEATURE_ROADMAP_MASTER.md).

## Entrypoints, privilege and persistent state

| Path | Executable owner / boundary | Storage or trust | Mutation route | Coverage at baseline |
| --- | --- | --- | --- | --- |
| `install.sh → online.py → bootstrap.py → distribution.py` | Installer executes under root; app wheel installed to `boxweave` account | `/var/lib/boxweave-default`, `/usr/local/lib/boxweave/default`, app SQLite, preexisting Provider | Systemd, package, Provider bootstrap, SSH key registration; explicitly privileged installer | Outside guest operation coordinator; preserve independent reviewed installer |
| SSH forced-command `/usr/bin/python3 /usr/local/lib/boxweave/default/mcp_gate.py` | Dedicated `boxweave` Unix account via `authorized_keys`; enforced before Provider | `/var/lib/boxweave-default/public/targets.json`, managed key, optional root-owned tool policy | Upstream `tools/call`: `connect(exec)`, create/start/stop/delete/resize, network/access, etc. | Not coordinated. Some names are upstream-forwarded; full tool catalog is runtime-dependent |
| Loopback HTTP `remote_mcp.py → relay → mcp_gate.py` | Authenticated HTTP service under app account; per-request isolated `python -E -s` gateway script | App SQLite v12 OAuth grants in `remote_auth.py`; identical gateway trust and tool policy | Delegated upstream `tools/call`, with 45-second relay bound | Not coordinated; timeout is UNKNOWN, no automatic retries |
| `python -m bwmini serve` + `web.py` | Application account; Web password cookie, Origin and CSRF are separate from HTTP MCP bearer/OAuth | App-owned `Store` SQLite v12 | Web bound projects, lifecycle plans, access plans, BoxActions | Existing independent durable receipts; not shared coordinator |
| `python -m bwmini worker` | Application service worker | Same app SQLite v12 and Provider launcher | `service.work_one → access/lifecycle/box_actions`, managed Box/network/file actions | Existing one-time dispatch/UNKNOWN protections, *not* cross-entry claims |
| `boxweave` `host_cli.py` | Root trusted installed helper under `python3 -I`; explicitly invokes app account for certain read/jobs | Root-owned public provenance and app-specific state | Doctor/report, selftest and bounded app/host tasks | Not safe to move/merge into app/HTTP |
| `nesting.py` privileged sudo helper | Root via narrow exact sudoers; no generic root shell | Exact Incus instance match | Post-create nesting boolean with partial-success semantics | Not coordinated with new shared claims |
| Web client scripts `client_profiles.py/client_setup.py` | Untrusted client local script; distinct from controller | App config backups, profile catalogue | Edit only reviewed local Agent MCP registration files | No server execution authority |

**Actual transport relationship:** SSH and remote HTTP *converge at the same gateway code*, not at the same process. The Gateway and Web do **not** converge at a mutation guard or receipt DB yet.

## Resource and write classification, as currently evidenced

| Mutation class | Existing function / tool route | Minimum logical claim keys | Status |
| --- | --- | --- | --- |
| Exact Box command or edit | Gateway `connect(exec)`; Web `BoxAgent.execute` / BoxActions | Provider-scoped immutable Box identity; logical-name claim for destructive/replacement conflicts | **U — unguarded by shared coordinator** |
| Box create | Gateway `create_container`, app `Lifecycle.work_one` | Scope + logical name, optional Tenant and owned port | **U**; post-create nesting runs a separate privileged helper |
| Box lifecycle update/delete/resize | Gateway forwarded Provider calls; app `Lifecycle.work_one` | Scope + logical name AND observed Box incarnation | **U** |
| Tenant ingress/network | Gateway forwarded upstream network tools; app `Access.work_one → Wire.patch` | Provider/project Tenant namespace; possibly multiple resource keys | **U** |
| Published TCP/UDP port | Gateway forwarded route tools; app `Wire.route_mutate` | Host/backend scope + protocol + published port, exact target Box | **U** |
| D4 diagnostic inspection | Gateway `boxweave_project_diagnose` | Read-only scope, explicit opt-in deep read | **Existing independent policy**; no mutation claim |
| Unknown upstream tool / opaque mutator | Gateway forwards catalog under optional global allowlist | **UNCLASSIFIED_MUTATION** until schema/catalog analyzed | **U**; cannot claim global protection |
| User-account/login/remote OAuth, graphs, model selection | Web/app storage-only mutations | Separate app DB; not guest Provider claims | **Out of guest coordination scope** unless changed |

`connect(exec)` can mutate arbitrary guest state even when tool name sounds observational. `list_containers`, `get_container` and capability reads do not need exclusive claims. A tool name allowlist is an **authority** filter, not a mutation classifier.

## Deployment and import truths

- Offline release helper manifest is produced by `scripts/build_release_manifest.py`, gated in `online.RELEASE_FILES`, then copied into the public helper directory by `online.install`. Adding a Python module to the wheel alone **does not** make it importable by forced-command Gateway.
- The gateway is executed as a standalone script, with sibling import fallback for `nesting` and `gate_diagnostics`; HTTP relay passes `-E -s`. Root helper `host_cli` uses `-I` and must remain a standalone file.
- `mcp_gate` receives a root-written trust registry but does not inherit the app SQLite auth/session. Although both use a dedicated account on default installer, do not merge logical principals.
- `operation_coordination` and `coordination_runtime` are currently wheel-only; private_connection requires same owner UID and 0600/0700, **no provisioner exists**. They are not active runtime controls.

## ARCH-0A acceptance and transition

**Source audit S PASS:** all known executable roots and route classes mapped above against the pinned revision. **Runtime catalog and live identity U:** dynamic upstream tools, non-default Provider project/backend and effective SSH/HTTP app identities require installed observation before claiming operational protection. Next gate ARCH-0B must define these as tested contracts, without inserting enforcement into legacy routes yet.

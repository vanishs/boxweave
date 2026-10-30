# Troubleshooting and non-destructive recovery (R-05/R-08)

Always start from a reviewed source/package and a known app user. Never print or upload credential files, JWT/private keys, full private application config, SSH known_hosts or raw database records. [Installation](INSTALL.md), [management CLI](CLI.md), [risks](verification/OPEN_RISKS.md) and [manual legacy migration](MIGRATING_FROM_051.md) own the detailed context.

| Symptom | Safe observation | Interpretation and next action |
| --- | --- | --- |
| Untrusted artifact/source | `sha256sum /trusted/boxweave-v2.zip` against **independent** operator digest | Mismatch blocks execution. Obtain reviewed bytes; do not disable hash verification |
| Installer plan rejects host | `bash install.sh --help`, then `plan` with reviewed paths | Confirm Ubuntu 24.04/x86_64/Python 3.12, app user, local user bus and Provider. No implicit privilege repair |
| Install marker persists | `bash boxweave.sh status --prefix "$HOME/boxweave-v2"`; inspect owned `update-in-progress.json` stage only | Correct exact failure then rerun the same reviewed package; never delete locks/marker/backups to force ready |
| `ADMIN_SETUP_REQUIRED` | Interactive `python -m bwmini init-admin` as the app user per install guide | First admin setup must finish before marker can clear; password is entered locally only |
| App unavailable or HTTP 401 | `boxweave.sh status` / owned user-service status | 401 is unauthenticated reachability; not proof of MCP, login, TLS or host availability |
| MCP initialization/read fails | `boxweave.sh doctor --mcp` only with authorization for authenticated read | Launcher/credential/Provider may be absent/expired; do not log tokens or silently rotate/repair them |
| Exact box missing | `boxweave.sh box inspect EXACT_NAME --project PROJECT` | Never guess a suffix, alter host settings or adopt an unverified match |
| Worker receipt unknown | Saved receipt/history + explicit non-mutating reconciliation | Unknown is not cancelled, failed or ready; never auto-replay |
| Native Tenant/route denied | Read-only capabilities, revision/Guard state, separate packet test with authorization | Configured route does not prove packet access/denial; never apply the retired patch |
| Client trust/Agent failure | Selected local profile `diagnose` and public server pin | SSH, MCP, registration, GUI and model inference are independent; no TOFU/private-key copying |
| Legacy 0.5.1 active | [migration checklist](MIGRATING_FROM_051.md) | No in-place DB conversion or automatic old service cleanup |

These instructions describe possible checks. The submitted new CLI/installer still needs real Ubuntu/Provider testing; any observed exit/error must be captured with source SHA/environment in the [validation queue](verification/REMEDIATION_TEST_QUEUE.md).

# Troubleshooting and non-destructive recovery (R-05/R-08)

Always start from a reviewed source/package and a known app user. Never print or upload credential files, JWT/private keys, full private application config, SSH known_hosts or raw database records. [Installation](INSTALL.md), [management CLI](CLI.md), [risks](verification/OPEN_RISKS.md) and [manual legacy migration](MIGRATING_FROM_051.md) own the detailed context.

| Symptom | Safe observation | Interpretation and next action |
| --- | --- | --- |
| Untrusted artifact/source | `sha256sum /trusted/boxweave-v2.zip` against **independent** operator digest | Mismatch blocks execution. Obtain reviewed bytes; do not disable hash verification |
| Installer plan rejects host | `bash install.sh --help`, then `plan` with reviewed paths | Confirm Linux x86_64/Python >=3.12, systemd, app user, local user bus and Provider. APT and a compatible repository suite are required for missing-package installation; Linux distribution names are not restricted. No implicit privilege repair |
| Install marker persists | `bash boxweave.sh status --prefix "$HOME/boxweave-v2"`; inspect owned `update-in-progress.json` stage only | Correct exact failure then rerun the same reviewed package; never delete locks/marker/backups to force ready |
| `ADMIN_SETUP_REQUIRED` | Interactive `python -m bwmini init-admin` as the app user per install guide | First admin setup must finish before marker can clear; password is entered locally only |
| App unavailable or HTTP 401 | `boxweave.sh status` / owned user-service status | 401 is unauthenticated reachability; not proof of MCP, login, TLS or host availability |
| MCP initialization/read fails | `boxweave.sh doctor --mcp` only with authorization for authenticated read | Launcher/credential/Provider may be absent/expired; do not log tokens or silently rotate/repair them |
| Exact box missing | `boxweave.sh box inspect EXACT_NAME --project PROJECT` | Never guess a suffix, alter host settings or adopt an unverified match |
| Worker receipt unknown | Saved receipt/history + explicit non-mutating reconciliation | Unknown is not cancelled, failed or ready; never auto-replay |
| Native Tenant/route denied | Read-only capabilities, revision/Guard state, separate packet test with authorization | Configured route does not prove packet access/denial; never apply the retired patch |
| Client trust/Agent failure | Selected local profile `diagnose` and public server pin | SSH, MCP, registration, GUI and model inference are independent; no TOFU/private-key copying |
| Legacy 0.5.1 active | [migration checklist](MIGRATING_FROM_051.md) | No in-place DB conversion or automatic old service cleanup |

## Remote MCP client connects but cannot refresh tools

A remote MCP connector can complete OAuth while failing later at authenticated
`POST /mcp`. A browser GET, TCP probe, or `boxweave http doctor` passing
**does not** prove that `initialize` or `tools/list` works. Use ordinary
operator commands; there is **no** extra monitoring daemon, debug listener,
privileged telemetry, token scraping or remote log uploader.

1. **Local metadata and service (no secrets):** `sudo boxweave http doctor`.
   Confirm `local_discovery=true` and the expected `public_url`.
   `external_verified=false` is intentionally not an external failure.
2. **Public transport (no secrets):** check
   `/.well-known/oauth-protected-resource/mcp` and
   `/.well-known/oauth-authorization-server` at the configured HTTPS
   origin. Each should return a 200 JSON document with the **exact**
   public origin and `/mcp` resource. A successful OAuth browser approval
   is still not proof of `tools/list` success.
3. **Authenticated protocol smoke (operator opt-in only):** use a short-lived,
   revocable **test** API token from the Web Remote MCP tab, and invoke
   `tools/list` both locally and, if explicitly authorized, on the current
   public origin. Importantly **omit** the optional JSON-RPC `params` field:
   real clients may send exactly
   `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`.
   Inspect only HTTP status, tool count and error code, not full tools,
   request headers, passwords, bearer tokens or private DB content.
   Never insert the bearer token directly into a shell command line, shared
   chat or diagnostic artifact. One example for the loopback endpoint (the
   header is supplied to curl over stdin, not its process arguments):

   ~~~sh
   read -rsp 'Temporary API token: ' BW_TOKEN; echo
   curl --fail --silent --show-error --max-time 20 --config - \
     --data-binary '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' \
     http://127.0.0.1:18844/mcp <<EOF | python3 -c 'import json,sys; x=json.load(sys.stdin); print("tool_count:", len(x.get("result",{}).get("tools",[])), "error:", x.get("error"))'
   header = "Authorization: Bearer ${BW_TOKEN}"
   header = "Content-Type: application/json"
   header = "Accept: application/json, text/event-stream"
   header = "MCP-Protocol-Version: 2025-11-25"
   EOF
   unset BW_TOKEN
   ~~~

   Revoke the temporary token once finished. Do not blindly retry writes,
   especially creates with an UNKNOWN outcome.
4. **Local exception evidence (only when needed):** after a failed client
   refresh, an authorized host operator can inspect
   `sudo journalctl _UID="$(id -u boxweave)" --since "10 minutes ago" -o cat --no-pager`.
   Look for `Exception on /mcp [POST]` and the exception **type plus source
   line**. Review and redact before sharing: journal output may contain
   credentials, addresses or request context. Do not enable body/Authorization
   header logging, relax OAuth, expose the local listener, or upload raw logs.

**Incident example:** ChatGPT OAuth and both public/local `tools/list` with
`"params": {}` worked, but ChatGPT's valid request without `params` caused
`KeyError: 'params'` at `bwmini/remote_mcp.py`. The fix is a narrowly scoped
optional-field compatibility regression in ordinary CI, not a BoxWeave-specific
runtime self-repair subsystem. Keep source CI evidence separate from live
installed acceptance.

These instructions describe possible checks. The submitted new CLI/installer still needs real Ubuntu/Provider testing; any observed exit/error must be captured with source SHA/environment in the [validation queue](verification/REMEDIATION_TEST_QUEUE.md).

# BoxWeave client enrollment

## Windows ChatGPT/Codex desktop + WSL SSH (v0.18.0)

The installed server supports the same generation implementation through a **terminal wizard** or **explicit non-interactive arguments**:

```sh
boxweave client
boxweave client --agent codex --transport wsl --admin-user YOUR_HOST_ADMIN --no-replace > boxweave-setup.ps1
```

The wizard asks for Agent and SSH transport; its last choice controls whether a **conflicting existing Windows BoxWeave MCP registration** may be replaced. **No** preserves conflicting configuration; **Yes** permits updating BoxWeave transport fields **only after a verified MCP read**, with a backup. This is unrelated to Linux script-file replacement. The generator never changes the Linux Provider or enrolls SSH keys.

**By default `boxweave client` prints the complete, copyable Windows PowerShell script on stdout.** The top banner contains instructions as PowerShell comments; all wizard questions and human guidance go to stderr, so `boxweave client > boxweave-setup.ps1` writes only valid PowerShell content. This shell redirection will overwrite an existing file unless you choose a new filename; `--output NEW_FILE` is an optional strict alternative that refuses overwrites. `--stdout` remains an alias for the default. Noninteractive flags: `--agent codex --transport wsl --admin-user USER` and `--replace` or `--no-replace` (the latter is the default without an interactive prompt). Store the generated script on the Windows machine, review it, and run it with **Windows PowerShell**. It is **not** a Linux Bash script. The script includes public data only, no private keys or Provider secrets.

Copy the script from your authorized Linux SSH account on the BoxWeave server to the Windows machine via an authenticated transfer, inspect it, and execute in **Windows PowerShell**, using the actual WSL distro name:

```powershell
powershell.exe -NoProfile -File .\\boxweave-setup.ps1 -Distro Ubuntu
```

The script checks or generates an Ed25519 key only under `~/.config/boxweave` in that WSL distro, pins the server's reviewed Ed25519 host key, and runs the existing Linux administrator SSH login with `sudo /usr/bin/python3 /usr/local/lib/boxweave/default/online.py enroll --public-key ...`. It defaults to password-only SSH administrator authentication, avoiding excess-key disconnects, and may request SSH/sudo credentials. For SSH-key-only administration use `-AdminAuthentication publickey -BootstrapIdentity /absolute/wsl/path/to/admin_key`. The restricted `boxweave` account does not gain shell access. If the server requires separate administration, run the script with `-NoEnroll` to print a public enrollment command, authorize it on the server, then rerun with `-AlreadyEnrolled`. Never send the client private key or JWT to the server.

After public-key authorization, the script performs MCP initialize, tools/list and a **real read-only** list_containers tool call over WSL SSH. Only after it succeeds will the script update the Windows ChatGPT/Codex desktop app's shared user-level `config.toml` at `%USERPROFILE%\\.codex` (or `CODEX_HOME`). It never executes a bundled WindowsApps binary or requires a standalone Codex CLI. If the entry exists, inspect it first; `-Replace` authorizes replacing the simple named MCP table only, while other TOML settings are preserved and previous config backed up. Ambiguous TOML must be migrated manually. Restart Codex to verify tool recognition. Neither the generator nor setup configures public routes, edits Incus storage, resets an SSH host pin, or modifies the Containarium daemon.

The script is not intended to be executed unattended without a reviewed preauthorized administration method: non-interactive **generator input** does not imply bypassing authentication or automatic security approval. For new release upgrades, rerun the generator to acquire a source-bound script; copying an old release's script can retain old behavior.

## Other selected client profiles and diagnostics

TASK-007 supports Windows-hosted Codex and ZCode, each with native or explicit WSL SSH. The stdlib-only profile helper requires an explicitly selected Python 3.11+ on Windows. It installs no runtime or distro. Existing PowerShell native Codex enrollment remains supported. Runtime acquisition/release packaging belongs to TASK-012.

Use an operator-reviewed checkout and independently verified expected source digest. A descriptor digest checks integrity against that decision; it does not authenticate an untrusted download. Verify the setup entry before executing it. Never derive expected verification evidence solely from the same untrusted payload.

Schema2 descriptor fields: schema=2; profile=codex/windows/native, codex/windows/wsl, zcode/windows/native or zcode/windows/wsl; source_sha256=reviewed profile_client.py digest; agent_argv=explicit absolute executable plus fixed entrypoint arguments; agent_version=exact version output; profile_directory=explicit private Windows directory; connection. ZCode additionally requires zcode_builtin={path,sha256} identifying its installed public builtin JSON catalog, never a user credential file. No tokens/passwords/private-key text or sign-in data belong in descriptors.

Connection fields: host, port, user, identity_file, known_hosts, server_key (full reviewed public Ed25519 pin), executable (ssh.exe or wsl.exe). WSL additionally requires distribution, an explicit existing distro; identity_file/known_hosts are absolute POSIX paths there. Native paths are absolute Windows paths. Profile directories and native identity must be operator-owned with only that user/SYSTEM allow ACLs. Reparse/symlink paths fail. Keep known_hosts limited to the pinned endpoint. No trust repair or distro fallback occurs.

    python.exe C:/reviewed/clients/profile_client.py register --descriptor C:/private/profile.json
    python.exe C:/reviewed/clients/profile_client.py diagnose --descriptor C:/private/profile.json

Registration performs a real SSH/MCP probe first, then writes only selected Codex home config.toml or ZCode workspace .zcode/config.json mcp.servers.boxweave. Conflicts require --replace. Unrelated settings/servers and nested Codex tool approvals survive; unsupported TOML fails closed. Private backups, cooperative lock and prepublication state check provide recovery; quiesce noncooperating editors. Publication is not a whole-client transaction. Diagnosis never enrolls keys or repairs config; Agent startup can update its normal private cache/database. Backups require operator-reviewed restoration.

For a fresh native key use windows/prepare-identity.ps1. For WSL run wsl/prepare_identity.py inside the explicit distro with --request JSON containing directory (fresh owned absolute POSIX directory), host, port and server_key. Invoke using explicit wsl.exe --distribution NAME --exec /usr/bin/python3 and separately marshalled arguments, without a login shell. Only the public result/key moves through the already authorized bootstrap channel. scripts/authorize_native_key.py runs as the exact intended non-root backend user/hostname, restricts the public key to the launcher and preserves unrelated entries. Never copy private keys across runtimes. Inspect interrupted preparation before retrying.

Native SSH uses -F NUL, WSL uses -F /dev/null and /usr/bin/ssh. Unicode/space paths remain separate arguments. SSH uses the key normally; setup never reads its contents. MCP diagnosis performs initialize/initialized, bounded paged tools/list and actual list_containers. Reports separate transport/registration/UI/model. agent_ui stays unverified and overall incomplete automatically; observation is recorded separately.

Observed Codex CLI0.165.0-alpha.1.2 uses mcp list --json. Observed ZCode desktop3.16.3/core0.17.0 loads workspace mcp.servers and exposes mcp/list in its NDJSON app-server. This internal protocol is version-pinned, not a stable public API promise. Desktop-shipped core lacks @zcode/tui; use desktop MCP page for UI evidence. Diagnostics retain PROGRAMDATA/ALLUSERSPROFILE needed by Windows OpenSSH, omit API-key environment variables, use a private data root and reviewed public builtin catalog. No model prompt/session-send is issued. Credentials/model setup is TASK-008.

Read [actual evidence](../docs/verification/TASK_007_PROFILES.md). These checks do not establish model access, clean installation, provider least privilege, remote cancellation, network isolation or release acceptance. Local tool filtering does not restrict upstream authenticated authority.

## R-04 verified scope, not new platform coverage

The owner **did not approve** adding macOS, Cursor or Claude Code. Current selected descriptors are `codex/windows/native`, `codex/windows/wsl`, `zcode/windows/native`, `zcode/windows/wsl` only. WSL means **Windows-hosted Agent with WSL SSH transport**, not automatically installing a WSL Agent or distro. The new `boxweave.sh client` prints guidance, not a generator. Web Client setup generates reviewed **public** descriptors and a hash-checked script for local use; neither implies enrollment, Agent GUI recognition, or model access. Run each of four real Windows profiles with explicit runtime/distro/path/pin and independently check GUI/model evidence. See [R-04 queue](../docs/verification/REMEDIATION_TEST_QUEUE.md).


### Windows PATH ambiguity guard (0.17.5)

The generated PowerShell setup selects one specific Windows WSL executable at `$env:SystemRoot\System32\wsl.exe`, validating that it exists before invoking it. Do not derive its command path from all `Get-Command wsl.exe` matches: Windows commonly exposes both the System32 binary and a WindowsApps alias, and coercing both matches into a single PowerShell string makes invocation fail with CommandNotFoundException. Existing v0.17.4 generated scripts can be corrected by replacing that assignment locally; doing so does not change WSL keys or the server enrollment. The fix is also covered by a generator regression test and PowerShell syntax CI.


### Existing nested Codex MCP configuration (0.17.6)

An existing Codex `[mcp_servers.boxweave]` may have legitimate nested TOML subtables (for example `[mcp_servers.boxweave.env]` or `[mcp_servers.boxweave.http_headers]`). These are **not** ambiguous or invalid. After the real MCP read check, the generated installer examines only the root BoxWeave transport keys (`command`, `args`, `startup_timeout_sec`, `tool_timeout_sec`), preserves other root keys, nested BoxWeave tables and unrelated MCP servers, and requires an explicit `-Replace` if the transport differs. Files are backed up before atomic replacement. Duplicate tables/keys, unsafe multiline transport fields, quoted/dotted ambiguous alternatives and missing parent tables still fail closed for manual review. `-AlreadyEnrolled` skips public-key registration once it has been verified as successful. The cross-platform PowerShell merger now has a **real execution regression** in CI with a parent and two nested tables; generating only valid PowerShell syntax was not enough to catch this issue.


### SSH user@host argument regression (0.17.7)

Real Windows Codex testing found that the generated PowerShell array expression `'boxweave@' + $ServerHost` yielded two distinct SSH arguments (`boxweave@` and `192.168.2.233`). The MCP probe executed through a separate Python path and passed, but the Codex registration was not runnable. The generator now computes `$sshTarget = 'boxweave@' + $ServerHost` **before** building the `$sshArgs` array; `$sshTarget` occupies exactly one argument. CI executes the generated argument assignment using `pwsh`, parses the actual JSON array, and asserts the single destination is present and both split fragments absent. 

For users who already corrected their desktop `config.toml` manually and successfully reached MCP initialize, tools/list and list_containers, no new server install or re-enrollment is needed. Switching Codex Agent environment from Windows native to WSL does not itself convert this Windows `wsl.exe`-based registration into a Linux-native `/usr/bin/ssh` registration. Independently exercising the SSH/MCP launcher is not equivalent to confirming direct Codex session tool injection; test each separately.

## Linux-owned workload SSH identity (0.18.0)

The Windows/WSL transport key authenticates the client to BoxWeave MCP. The Linux server owns a **separate** workload SSH identity and retains the private key. Before forwarding `create_container`, `bwmini/mcp_gate.py` appends its own validated public key to `ssh_keys`. Optional additional keys must be distinct, valid Ed25519 public keys; the host's key cannot be displaced. This prevents the upstream missing-keys private-key fallback.

The Windows client script now configures only the Codex MCP transport after a verified SSH/MCP read; it never writes `AGENTS.md`, `AGENTS.override.md`, or WSL Codex guidance. Older managed guidance blocks from previous versions are **not automatically deleted**. The former `-WslGuidance` option no longer applies.

`$ServerHost` and `$ServerPort` near the start of the generated script are editable to the IP/DNS name and port **reachable from Windows**. Keep the separately verified `$ServerKey` host key pin unchanged. Never disable strict host-key checking or silently reset a pin.

The old `archive/0.5.1` generated script was shorter because it downloaded most setup logic from a fixed Git commit at runtime. The current self-contained script remains longer, but eliminating AGENTS merging reduces its size without introducing runtime code downloads. A short downloader would require independently verified, source-bound artifacts and separate testing.

Only **new** boxes created via the BoxWeave MCP gateway receive this automatic key. Existing guest authorization, external Provider clients, and host trust are unchanged. Real Windows/WSL onboarding and MCP create/connect/Python acceptance remain unverified.

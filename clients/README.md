# BoxWeave client enrollment

## Windows ChatGPT/Codex desktop + WSL SSH (v0.17.14)

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

## Managed workload SSH identity restoration (0.17.8)

The **client transport identity** (generated in WSL `~/.config/boxweave`) authenticates to the BoxWeave MCP launcher; it is **not** the key the server uses to SSH into a newly created box. The installed server generates a distinct `/home/boxweave/.ssh/id_ed25519` key pair, retaining the private half exclusively on the server. Only its validated public half, originally exported by the old `archive/0.5.1` setup workflow, is source-bound into the new generated setup script. The generator refuses absent/invalid `managed_public_key` before publishing any script.

After SSH enrollment and the real MCP read probe, the script safely merges the BoxWeave-only `<!-- BEGIN/END BOXWEAVE MANAGED -->` instruction block into the effective Windows Codex `AGENTS.md` (or nonempty `AGENTS.override.md`). It preserves unrelated instructions and refuses duplicate/malformed markers or conflicting managed blocks in both files. Existing files are backed up before atomic replacement. Other agent settings and MCP entries remain unchanged beyond the existing explicit `-Replace` contract. A separate **opt-in** `-WslGuidance` switch writes the same public instruction to the selected WSL user's `~/.codex/AGENTS.md` (or populated `AGENTS.override.md`), with atomic file replacement, private backups, and refusal on ambiguous/symlinked files. This is guidance-only WSL assistance; the current unified generator still registers the **Windows** `config.toml` and does **not** automatically install a WSL-native MCP registration.

For newly generated Windows WSL bridge scripts, after the operator reviews the generated source:

```powershell
.\boxweave-setup.ps1 -Distro Ubuntu -AlreadyEnrolled -Replace -WslGuidance
```

Use `-AlreadyEnrolled` **only after enrollment is verified**. The `-Replace` switch is needed only when an existing BoxWeave MCP transport entry differs. Avoid rerunning this Windows bridge installer solely for guidance if you have independently changed your Codex WSL Agent registration to a Linux-native `/usr/bin/ssh` command: it may replace a working client configuration. Instead obtain and carefully merge the publicly exported managed key into your WSL Codex guidance under its actual user home, or use the additional script in a disposable/reviewed environment first.

The server's `bwmini/mcp_gate.py` reads **its own** managed identity's public half as the configured non-root user and checks that at least one `create_container.ssh_keys` element equals it. It rejects a malformed/missing public key, a wrong (including client transport) key, duplicates and an unavailable managed identity before forwarding to Containarium; additional individually provided valid public keys are permitted. Tool discovery describes the condition, and errors distinguish `MANAGED_PUBLIC_KEY_REQUIRED` from `MANAGED_IDENTITY_UNAVAILABLE` without printing the expected public key or private material. This applies to **new create calls**. Existing box `authorized_keys` is not rewritten, and failing `connect` is not silently repaired. The correct key depends on which server/layer hosts the MCP launcher.

Historical foundation: `archive/0.5.1/server/clients.py` published `container_public_key`, `clients/windows/setup.ps1` and `clients/posix/setup.py` installed agent instructions, and `server/smoke.py` bound the exact public key to create/connect/Python validation. The historical October 22 nested Hello World pass is not a test of today's client or container. Next live acceptance: inspect the existing `hello-world-test-container`, compare its authorized public key fingerprint with the server-managed public key, repair that box through a separately reviewed, authorized procedure if different, and verify actual MCP identity and Python stdout `hello world\n` with exit 0. Do not create a replacement box, weaken SSH policy or claim the present failed run passed.

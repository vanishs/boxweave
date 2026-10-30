# BoxWeave client enrollment

## Windows ChatGPT/Codex desktop + WSL SSH (v0.17.4)

The installed server supports the same generation implementation through a **terminal wizard** or **explicit non-interactive arguments**:

```sh
sudo boxweave client
sudo boxweave client --agent codex --transport wsl --admin-user luka --output /home/luka/boxweave-setup.ps1
```

The wizard asks for Agent and transport; presently the integrated profile is `codex/windows/wsl` (Codex itself stays on Windows). The generator reads the root-owned **public** server identity and writes a new mode-0600 PowerShell script, never a JWT or a private key. Omit `--output` to write a new `boxweave-setup.ps1` in the current directory; an existing file is refused rather than overwritten. `--stdout` emits public script source for a reviewed transfer. CLI generation never enrolls keys or edits client machines.

Copy the script from your SSH-admin account on the BoxWeave server to the Windows machine via an authenticated transfer, inspect it, and execute in **Windows PowerShell**, using the actual WSL distro name:

```powershell
powershell.exe -NoProfile -File .\\boxweave-setup.ps1 -Distro Ubuntu
```

The script checks or generates an Ed25519 key only under `~/.config/boxweave` in that WSL distro, pins the server's reviewed Ed25519 host key, and runs the existing Ubuntu administrator SSH login with `sudo /usr/bin/python3 /usr/local/lib/boxweave/default/online.py enroll --public-key ...`. It defaults to password-only SSH administrator authentication, avoiding excess-key disconnects, and may request SSH/sudo credentials. For SSH-key-only administration use `-AdminAuthentication publickey -BootstrapIdentity /absolute/wsl/path/to/admin_key`. The restricted `boxweave` account does not gain shell access. If the server requires separate administration, run the script with `-NoEnroll` to print a public enrollment command, authorize it on the server, then rerun with `-AlreadyEnrolled`. Never send the client private key or JWT to the server.

After public-key authorization, the script performs MCP initialize, tools/list and a **real read-only** list_containers tool call over WSL SSH. Only after it succeeds will the script updates the Windows ChatGPT/Codex desktop app's shared user-level `config.toml` at `%USERPROFILE%\\.codex` (or `CODEX_HOME`). It never executes a bundled WindowsApps binary or requires a standalone Codex CLI. If the entry exists, inspect it first; `-Replace` authorizes replacing the simple named MCP table only, while other TOML settings are preserved and previous config backed up. Ambiguous TOML must be migrated manually. Restart Codex to verify tool recognition. Neither the generator nor setup configures public routes, edits Incus storage, resets an SSH host pin, or modifies the Containarium daemon.

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

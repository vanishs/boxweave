# BoxWeave management CLI

The new `boxweave.sh` is a **read-only management and diagnostic entry**. From a reviewed source checkout use `bash boxweave.sh <command>`; from the verified installed release use its bundled `boxweave.sh`. It uses the package-local Python interpreter when present and does not elevate privileges. All successful responses are bounded JSON.

| Command | Meaning |
| --- | --- |
| `version [--components]` | Local VERSION and optionally independently observed Incus/Containarium client versions; no server inference |
| `status [--prefix PATH]` | App release/user-services/loopback observation using distribution health; no provider probe |
| `doctor [--prefix PATH] [--mcp]` | Status and host CLI versions, confinement unverified; `--mcp` explicitly checks authenticated MCP discovery/read |
| `box inspect NAME [--project PROJECT]` | Exact Incus read-only query and selected non-secret metadata; no suffix guess |
| `client` | Generate a reviewed Windows Codex + WSL SSH setup script interactively or from explicit flags; requires root to read installed public identity; no key enrollment on the server |

Exit: `0` for successful observation/ready status, `1` for read failure/failed status, `2` for malformed arguments or an incomplete/missing app. A product version is **not** a verified running release. CLI health does not prove effective confinement, boot readiness, Agent GUI, packet enforcement, public TLS or inference.

Old job/selftest, provider upgrade, and unchecked key/token commands are not recreated as unsafe aliases. The selected Windows Codex + WSL client generator is a new source-bound public-only implementation. Application mutations continue through explicit `python -m bwmini` plan/confirm actions. [Scope](product/SCOPE.md), [host bootstrap](HOST_BOOTSTRAP.md), [tests](verification/REMEDIATION_TEST_QUEUE.md).

| box diagnose NAME [--project PROJECT] | Exact read-only instance/log observation and proc mount denial guidance; parent value unverified |

Doctor now includes local container detection and unverified parent/effective nesting state. [Nesting guide](NESTING.md).


## Windows Codex + WSL SSH

`sudo boxweave client` uses a terminal wizard. `sudo boxweave client --agent codex --transport wsl --admin-user luka --output /home/luka/boxweave-setup.ps1` accepts complete non-interactive choices. Both modes generate the same reviewed PowerShell script; no SSH access or remote enrollment happens during generation. Run it on Windows to use an explicitly selected WSL distro, enroll a public key through an existing Ubuntu SSH/sudo administrator, validate read-only MCP and add the Windows Codex MCP entry. Separate approval: `-NoEnroll`, manually run the displayed server command, then `-AlreadyEnrolled`. Existing Codex entry replacement is opt-in `-Replace`. See [client setup](../clients/README.md).

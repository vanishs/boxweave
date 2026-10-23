# POSIX clients and WSL

`clients/posix/setup.py` supports macOS and Codex/Claude Code CLI in WSL. Generate WSL setup with `boxweave client --os windows --agent codex --runtime wsl` and paste it inside the intended Linux user/distribution. Python 3.11+ is required.

Native Windows agents using WSL only for SSH use the PowerShell installer instead. Standalone Linux is outside the initial 0.2.0 matrix. See [client setup](../../docs/CLIENTS.md).

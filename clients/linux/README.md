# Linux client adapter (planned)

**Not implemented in this release.**

This location is reserved for a client-side Linux installer. The server bootstrap `install.sh` should not be used as a Linux desktop client installer.

Planned client capabilities: detect SSH and the user's agent configuration, create a dedicated key in `$HOME/.ssh`, pin the Ubuntu server's public host key, enroll a restricted transport key, and update only the selected agent's MCP configuration with backups.

Do not bake Ubuntu-specific server assumptions into this future client adapter; support multiple Linux distributions where practical.

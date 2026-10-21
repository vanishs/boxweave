# macOS client adapter (planned)

**Not implemented in this release.**

This location is reserved for a future macOS setup script that uses the system OpenSSH client, locally generated private keys and agent-specific configuration writers. Expected requirements:

- Discover the macOS home directory and available SSH tools without hard-coded usernames.
- Keep private keys on the Mac; pin the Ubuntu SSH server host public key before enrollment.
- Enroll a restricted public key using an already authorized SSH login.
- Support multiple agents through separate adapters rather than writing Codex files unconditionally.
- Preserve and back up existing agent configuration.
- Handle macOS Keychain integration as an optional enhancement, not a prerequisite.

No `curl | sh` installation command is published yet because no macOS installer exists.

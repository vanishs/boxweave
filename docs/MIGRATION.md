# Migrating from the earlier Codex/Containarium bridge

The pre-BoxWeave prototype used paths such as `/etc/codex-containarium-bridge`, a `start-containarium-mcp` launcher, and timers named `containarium-hostkey-sync.timer` and `containarium-token-rotate.timer`.

**Do not run competing old and new host-key synchronization timers.** On an existing manually configured server, back up your config and inspect the old timers:

```bash
sudo systemctl list-timers --all | grep -E 'containarium-hostkey-sync|containarium-token-rotate|boxweave' || true
sudo systemctl disable --now containarium-hostkey-sync.timer containarium-token-rotate.timer
```

If either old timer is not installed, `systemctl` may print an expected error. The new installer refuses to enable BoxWeave while an old timer is still active. It does not silently uninstall old files or delete existing credentials.

Then install BoxWeave using the new README instructions. Re-run the generated Windows PowerShell setup under your Windows profile. The new client SSH key defaults to `boxweave`; the MCP configuration table becomes `[mcp_servers.boxweave]`. The existing Codex files may still contain a legacy registration and managed policy block. **Review and remove them manually after confirming BoxWeave connects**, so the new and old servers do not appear side by side.

The installer leaves the historical Ubuntu JWT files, `known_hosts` backups, old SSH authorized keys, and old service definitions untouched. Clean them up only after confirming no remaining clients use them. The upstream Containarium installation is not renamed or removed.

For an entirely new machine, no migration is required.

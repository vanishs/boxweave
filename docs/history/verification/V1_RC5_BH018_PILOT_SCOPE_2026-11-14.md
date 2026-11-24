# RC5 development pilot: BH-018 only

Source integration from PR #146 pinned to `d5402591bc119d21ea6e7621caa52a63b2e4168d`; this branch copies only the source-level BH-018 Exec classification, atomic finalization and the focused native/installed test tooling. It does **not** merge BH-019 unsafe deep AppArmor/BPF work, nor sign BH-019 or S2.5 HOST_GO.

**Safety:** retain existing immutable UNKNOWN and original Incus Operation IDs; do not replay/unlock, disable AppArmor, grant privileged nested containers, or alter a shared Incus host. On host upgrade back up the original SQLite ledger and stop the old service before changing binaries, following `install.sh` and existing service runbook.

**Gate:** separate GitHub-hosted Ubuntu24.04 Incus7.6 installed HTTP MCP crash/WAL/ACK matrix and native S2/S2.5 gate must all pass before publishing an RC5. The historical intermittent installed-MCP reconnect/fresh-Exec readback failure remains under investigation; a single source-level green is not acceptance. Publication is an immutable **development prerelease**, not HOST_GO or completed BH-019. Full research/evidence remains in Draft PR #146.

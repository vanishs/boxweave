# H3A installer phase and optional OAuth continuation

Base: 25468750739c8b9e0322477155c4236af83317d9. Box A 9d374e18-196a-493d-bf1f-2d582589f04d, independent detached worktree /root/pr188-h0/h3a-source. Original H2C workspace and RC13/RC14 runtime fixtures retained.

## Changes and proof

- Normalize only one trailing slash on an otherwise canonical HTTPS DNS origin; direct CLI validation remains authoritative. Bounded correction, defer, or precise continuation; never echo rejected input or blindly repeat uncertain writes. PasswordConfigured is independent of Enabled; old candidates use the verified Doctor password observation. Existing password retained; no controller reinstall or restart.
- Phase-specific failure for invalid result/native readback/installed identity/Doctor. Guided completion verifies reviewed binary SHA, Project, service identity, active state and matching Doctor build. Existing native approval, strict scope, and default mode are unchanged.
- Operation 377a7d75d420b3b2f0cd6912b5ac9f8d: guest exit 0; initial 7 wrapper tests, Go OAuth regressions in internal/state and cmd/boxweave, candidate build and actual restricted-user CLI on disposable SQLite PASS.
- Operation 65e6b452e8bbfad2adf1e7b304793a78: guest exit 0; final 9 wrapper tests and real CLI repeat PASS. Real invalid origin did not change stored password/epoch; normalized continuation changed only origin and epoch once, preserved password hash, then disable retained password. Generated private fixture credential and temporary ledger removed by fixture. No external tunnel or client authorization claimed.
- CI extends clean installed lifecycle with exact candidate, installed controller status, real isolated OAuth continuation and unchanged live OAuth status/PID. Existing read-only/stale native preview and guided installs remain required.

## Boundaries

Full fresh activation on clean Legacy/Scoped VM still requires CI. Box A retains inherited LXC drop-in and staged RC13; do not weaken it, rebuild or replace the original. No physical L0 configuration changes. No new default-mode or adoption policy. H3B not implemented.

Latest parent CI 45524336026: Scoped and Legacy normal lanes passed; Scoped retirement failed SCOPED_PEER_V2_ROOT_OUTCOME_UNVERIFIED; Legacy offline failed the WAL/SHM/journal absence assertion. Quality, source, build and Forward checks passed. These are open H2 blockers, not silently reclassified or repaired in H3A.

# R-01 / BOI-2: guarded compare-and-swap guest patch primitive

**Stage:** internal implementation and tests only. This module is not yet exposed as an MCP write tool and does not claim cross-agent conflict protection. The user authorized completion of Phase 2 with evidence-driven incremental commits.

The fixed guest script under `bwmini/box_patch.py` replaces a maximum 1024-byte existing regular file only after an exact SHA256 match. It verifies bounded relative paths below the exact target's home directory, rejects all symlinks and hard links, fsyncs a same-directory temporary file and uses atomic rename. The script does not interpret arbitrary shell input. It rechecks the original inode just before rename; external non-cooperating writers still require the durable cross-entry claim layer (P1-B2) before a production MCP mutator is exposed.

Tests: `tests/test_box_patch.py` covers success/readback, stale-hash refusal, symlink/hardlink refusal, directory traversal, excess size, and failed response classification. Genuine installed MCP, shared coordination and live guest acceptance remain separate L/U gates. No host install, Provider mutation or stable version promotion is part of this slice.

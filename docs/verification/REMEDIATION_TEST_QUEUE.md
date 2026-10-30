# Remediation validation and next-Agent queue

This is an append-only *planned vs executed* ledger. Do not write PASS unless the command actually ran, its exit/status was inspected, and the source/environment are recorded. The repository edit session operates through the GitHub connection **without a runnable trusted checkout**, so it cannot claim routine Linux or browser tests.

| Change | Test to run in authorized environment | Present evidence / acceptance |
| --- | --- | --- |
| 0.10.0 baseline, R-00/R-01 | `python scripts/check_knowledge.py`; link/README/VERSION/lock/scope consistency; `npm --prefix web ci && npm --prefix web run build` | U — not run in this source-only edit. Check especially retained historical links and IDs |
| Bootstrap/installer R-02 | Shell syntax, explicit-help/no-side-effect dry run, malformed inputs, existing Incus/Provider intact, isolated Ubuntu 24.04 fresh-host+reinstall+repair, signed/pinned upstream provenance, no-secret output | U — no host installation authorized |
| Management CLI R-05 | Read-only help, no config, missing service, version mismatch, forbidden target, no secret content, MCP-down; exact exit contracts | U — no host/user manager available |
| App distribution | `.venv/bin/python scripts/validate.py` and `tests/test_distribution.py`; isolated package build/install/upgrade, DB backup/marker, re-run and admin setup | U — old historical evidence does not count for new source |
| Web/clients | `.venv/bin/python scripts/validate.py --browser` seven fixture flows, plus separately approved four-profile Windows live registration/GUI | U — fixture is not real client evidence |
| Provider/Incus/Tenant | Independent MCP initialize/discovery/read; pinned upstream capabilities; authorized policy/packet, boot/confinement, public TLS | U — requires explicit live authorization |

Keep source SHAs, exact commands/exits, stdout/stderr redaction, any retained fixtures, known failures and required permissions in the actual Agent's result. Do not run live test drivers without their original scope and cleanup approval.

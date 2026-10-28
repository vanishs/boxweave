# TASK-003 completion plan and evidence

Authorized continuous TASK-002/003 run, SRC-063. TASK-003 scope is the selected Windows-native Codex path; the other3 selected profiles remain TASK-007, model setup TASK-008, release packaging TASK-012/013. Existing user-observed native TUI/read invocation is SRC-057, not inferred from CLI registration.

Acceptance before implementation: provide a local fresh-directory key preparation command with explicit private ACL and no private-key read/export; authorize only its public half through an existing approved bootstrap channel on the exact non-root test backend. Keep unrelated authorized keys, restrict new key to fixed MCP launcher, refuse conflicting/unsafe paths, be idempotent and atomic with recovery backup. Independent native diagnosis verifies trusted probe bytes and actual handshake/read, inspects actual selected Codex registration without enrolling/repairing, reports disabled/missing/mismatched registration separately and leaves GUI/overall limits explicit. Test rejection paths and unchanged personal configuration/trust/key data. Run real fresh-key prepare -> authorized restricted public-key install -> trusted bootstrap -> native MCP/CLI recognition -> independent diagnosis. Reuse approved backend/18447 route; no new public exposure, private-key transfer, host setting, inference or sign-in copying.

## Delivery — version0.4.0, SRC-065

TASK-003, including the remaining TASK-003b2 path, is complete for the selected minimal Windows-native Codex profile. The authorized bootstrap is an explicit two-stage public-key workflow, not a new host credential API or an unauthenticated one-click install. Automatic acquisition of bootstrap credentials, other profiles and release packaging are excluded here. Existing trusted-identity registration, replacement/preservation and provider identity parser remain tested; TASK-007/008/012/013 keep their own scope. A direct upstream MCP key permits launcher access, not per-tool least privilege; retain the single trusted operator assumption.

### Trusted entry points and workflow

Deliver prepare-identity.ps1, diagnose-native.ps1, authorize_native_key.py and bootstrap-native.ps1 through the independently authorized operator channel; compare their reviewed source hashes before execution. They are trusted entry points, like bootstrap itself; a digest fetched alongside an untrusted download is not authentication. bootstrap continues to verify both registration/probe payload digests before execution. diagnose-native independently verifies the probe's exact bytes and runs a task-private copy; it never executes enrollment/registration repair. Distribution/signing/installer work remains TASK-012.

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File clients/windows/prepare-identity.ps1 -KeyDirectory C:/private/fresh-boxweave-key
```

Parent must already exist, selected directory must be fresh/local and not beneath a reparse path. The command creates an Ed25519 noninteractive key with native ssh-keygen, protects the new directory and private file for current user/SYSTEM only, reports public/private file paths, and leaves remote enrollment pending. It never reads/displays/copies the private file. Existing key directories are refused, not repaired/overwritten. ACL/keygen failure leaves visible incomplete state; inspect it and use a fresh directory, do not infer remote enrollment.

Transfer only id_ed25519.pub and the reviewed Python authorizer via the existing authorized MCP/bootstrap channel to the exact test backend. As its configured non-root user:

```bash
python3 authorize_native_key.py --public-key-file client.pub --expected-user bw-v2-mcp-nested --expected-hostname bw-v2-mcp-nested-container
```

No remote-home/user override is accepted. Validate full binary-format public Ed25519 key and current user/hostname, safe owned home, private .ssh/files and fixed installed launcher. New entry is restrict,command=/usr/local/bin/boxweave-mcp; no general remote shell. Preserve other keys, refuse a conflicting existing unrestricted copy, limit lock wait10s, serialize cooperative writers, compare before publication, atomic replace/fsync with original public-authorized-keys backup. No host sshd/firewall/credentials change. Quiesce other key editors: this is not atomic CAS against non-cooperating writers. An interrupted publication may be ambiguous; inspect original public entry/backup, rerun idempotently only after checking scope. Keys/backup/state are retained; no automatic revocation or cleanup.

Complete the reviewed existing public descriptor using the local identity path, previously trusted host pin/known_hosts, exact Codex version and registration/probe digests. Run existing bootstrap with explicit AgentHome/CodexExecutable. Then:

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File clients/windows/diagnose-native.ps1 -Descriptor C:/reviewed/trusted.json -PayloadDirectory C:/reviewed/windows -AgentHome C:/private/codex-home -CodexExecutable C:/reviewed/codex.exe
```

Diagnosis is independent observation: actual initialize/initialized/discovery/read and selected native registration compared against expected exact transport/paths. Connection success can coexist with missing/mismatched/disabled registration and returns exit2 for that incomplete selected setup. Exit0 means connection passed + registration matching; agent_ui remains unverified and overall incomplete without human evidence. GUI is not fabricated, no inference/sign-in is performed. Invalid config/payload/trust/runtime is reported narrowly without raw CLI/remote exceptions. User-private TEMP and AgentHome remain requirements; probe cache ACL inherits private TEMP. No edits to selected or personal configuration/trust/keys by diagnosis.

## Actual evidence

- Non-root project-LXC full unittest discovery:64 tests OK18.550s,exit0. Six new enrollment cases cover public-key format/private-text rejection, wrong identity, unsafe paths, preserved other keys/backup/restriction/idempotence, concurrency, publication failure and bounded lock. Task002's58 tests remain passing.
- Windows PowerShell5.1 prepare runs create local keys only. Final fresh-native-key-v3 succeeds with private-file ACL current user/SYSTEM. On dedicated bw-v2-mcp-nested-container (created1793212095), source-matching authorizer installs only its public half, returns changed=true, then changed=false; fingerprint SHA256:9ZgPnuqGfmdKusUVPeuUItI7wWPVNXlqZRYV4u8u2lc. Existing18447 mapping/pin reused, no new route. Real trusted bootstrap into isolated codex-home-v3 returns verified_cli/connection passed,exit0; actual diagnose-native returns matching/passed/overall incomplete,exit0. Native Codex0.165.0-alpha.1.2, upstream MCP0.114.2/protocol2024-11-05.
- tests/windows_native_enrollment.ps1 verification-02 passed: actual matching connection/registration, missing/disabled/mismatch independent of working connection, pin/tamper rejection, fresh Unicode-space key directory/private ACL, refuse existing directory, unchanged enrolled/personal config and trust. Final verification-03 repeats against the final fresh-v3 enrolled profile and generalized port fixture; its actual result is recorded in the final handoff.
- Existing windows_native_registration.ps1 regression-01 passed with the new independently enrolled identity: integrity/pin/schema/result failures, actual MCP/CLI, no-Codex-PATH/explicit executable, Unicode/space home, conflict/preservation/backup/idempotence; default personal config digest unchanged. Existing windows_connection.ps1 AST/3 no-network checks passed.
- Existing human native TUI connected(2 tools) and actual list_containers is SRC-057. That observation is separate from these new isolated CLI profiles; no fresh-key GUI, instruction/model readiness, other version/profile or clean-Windows install is claimed.

Final verification-04 against codex-home-v3 passed exit0 after switching diagnosis to validated mcp list inspection, including malformed TOML reported CONFIG_UNSUPPORTED/registration unverified instead of incorrectly classified missing. It also passed the matching/missing/disabled/mismatch, pin/tamper, private-ACL Unicode key and preservation checks. No new sign-in or user interaction was required.

Intermediate failures retained honestly: authorizer first syntax check found a missing lock-finally clause; corrected before any enrollment. Initial Windows ACL test detected OpenSSH's additional generated-file principals. A generic .NET ACL write required an unavailable privilege; no elevation/host repair was used. The product now changes only its newly created key's DACL through native icacls and verifies principals; fresh-v3 and negative tests pass. Earlier failed/local test keys and public backups are retained privately, never exported or staged. No private-key/sign-in/JWT read/copy occurred; normal SSH signing is credential use, not export.

Source authored in assigned dev LXC and exact-byte synchronized; reviewed hashes in task003-source-hashes.json. Root/metadata version0.4.0 is the rewrite, distinct from supporting installed BoxWeave/provider versions. Existing public demo and active user's old test/client homes remain unchanged. No new box, route, provider/core repair, policy write, inference or push. TASK-002/003 completion does not certify all ACs, release, WSL/ZCode, installer or latest-version whole TASK-005 integration; those remain separately tracked.

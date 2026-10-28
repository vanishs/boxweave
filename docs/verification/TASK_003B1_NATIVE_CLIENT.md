# TASK-003b1 — native executable selection and cooperative client acceptance

Version 0.2.2; parent 2a0fcd97c24fbf47c5c5c1129044da4a64305c47, branch rewrite/v2. Sources SRC-055..058. This bounded child of TASK-003b covers REQ-001/002/003/016/017 and matching AC subsets. It does not complete automatic identity bootstrap, four-profile support or phase 5.

## Scope and acceptance

The cooperative test exposed that an app-integrated terminal can find codex.exe while a double-clicked Windows launcher cannot. Product bootstrap and registration now accept optional -CodexExecutable, an operator-selected absolute local .exe path. The bootstrap forwards it only to the hash-verified payload. Registration rejects relative, missing, directory and reparse paths; checks the descriptor's exact CLI version; and uses that same executable for subsequent config validation. Omitting the parameter retains PATH-based Application lookup and fails with RUNTIME_UNSUPPORTED before home publication when unavailable. No global PATH edit or silent scan of app-internal directories is performed.

Acceptance for this slice: actual registration succeeds with Codex absent from PATH and an explicit executable; missing runtime and relative executable fail before AgentHome creation; existing integrity/conflict/idempotence/preservation checks remain passing; native client loading and an actual read-only call are separately recorded. Source is authored in the assigned project LXC; actual native-runtime checks run on Windows. No new resource, exposure or model request is needed for this implementation iteration.

## Reproducible native check

Use a private fresh test root and a reviewed public connection descriptor referencing an already-authorized local identity and known_hosts. Do not distribute private keys or sign-in data. Example paths are operator inputs, not download locations:

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File tests/windows_native_registration.ps1 -ConnectionDescriptor C:\reviewed\connection.json -TestRoot C:\private\fresh-test -CodexExecutable C:\reviewed\codex.exe
```

Actual run: Windows PowerShell 5.1; codex-cli 0.165.0-alpha.1.2; .runtime/native-registration/run-06. The test removes Codex from PATH while leaving Windows/OpenSSH tools, verifies missing/relative runtime rejection with no home created, then performs real registration using the explicit executable. Existing 15 reported checks also pass: payload integrity, real native MCP read, actual CLI recognition, Unicode/space home, idempotence, conflict preservation, explicit replacement preserving other settings and approvals, original-byte backup, unsupported inline form, tampering, pin, schema, malformed result and unchanged personal config digest. Exit 0. No user sign-in file is read or copied. Actual API/MCP peer for this run is the dedicated backend below, not the previous management endpoint.

Non-root assigned project LXC /home/boxweave-tenant-dev/rewrite-v2-phase4-40a600a: .venv/bin/python -m unittest discover -s tests -v, 41 tests OK, 11.586 seconds, actual exit 0. Existing Windows connection preflight is recorded separately. Only client and version metadata change; no frontend behavior changes.

## Dedicated backend and user-observed TUI evidence

SRC-055/056 separately authorized and prepared the isolated native test entry, dedicated restricted launcher key and TCP 18447 -> 10.206.30.8:22. Exact backend bw-v2-mcp-nested-container, createdAt 1793212095, native networkTenant boxweave-v2-test. Route has no LAN-only source filter. SSH host pin was obtained through authenticated management inspection, with strict verification retained. This historical setup is distinct from this code iteration and from the existing fixture Web route 18446.

Actual strict SSH MCP initialize/initialized, tools/list (101 tools), list_containers and get_container succeeded: protocol 2024-11-05, upstream containarium-mcp-server 0.114.2. get_container identifies v2-live-hello-container / v2-live-hello, createdAt 1793213048. The server is the retained upstream MCP backend used by the v2 integration, not a newly rewritten MCP protocol server.

SRC-057 is human-reported cooperative evidence: after normal sign-in in the separate test home, /mcp displayed boxweave connected (2 tools); an actual requested list_containers returned exactly one running v2-live-hello-container, IP 10.138.33.2, CPU 1, memory 1GB, disk 8GB. This establishes native TUI loading and read-only invocation for the tested setup. The test did not create or modify containers. Other app-provided servers appeared too; isolation means a separate Codex home/config and dedicated BoxWeave connection, not absence of all app integrations. Desktop GUI loading, instruction loading, model completion and every other profile remain unverified.

The ignored operational launcher initially hashed the whole config and rejected normal first-run TUI/settings additions. It was corrected to fingerprint the effective MCP registration plus the trusted descriptor and CLI version; real read-only connection was revalidated. That task-specific launcher and its key paths are not a product installer or committed release payload. Product code does not depend on ready.json or that launcher. Do not read auth.json, replay login or infer authentication failure from a config checksum change.

## Remaining boundaries

TASK-003b1 is complete for explicit native executable selection and the recorded cooperative read-only TUI evidence. TASK-003b2 remains: implement reviewed automatic bootstrap identity provisioning as a reproducible product path. Manual authorization of a lab-only public key is not that implementation. Parent TASK-003 remains partial; TASK-005 has real Web/worker/MCP/workload evidence and native read-only client evidence, but its full enrollment prerequisite remains incomplete. TASK-006 dependencies are not waived.

The tested backend, inner workload and approved routes are retained. No cleanup of resources/credentials is inferred. The current source fix is not deployed over the test user's active profile, so its reviewed old payload descriptor remains valid. No commit/push in the legacy checkout, host/provider upgrade or broad security change. Additional Codex versions, WSL, ZCode, installer/update/release acceptance and RISK-020's provider flag mismatch remain open.

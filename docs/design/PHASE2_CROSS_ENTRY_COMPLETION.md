# Phase 2 shared-entry protection: v0.41.0 source acceptance

[Master roadmap](FEATURE_ROADMAP_MASTER.md) ·
[R-04](MULTI_AGENT_BOX_LEASE_CONFLICT_V1.md) ·
[Phase 2 status](../verification/PHASE2_SOURCE_ACCEPTANCE_2026-11-02.md)

**Scope:** owner-authorized source/fixture implementation. No root-owned protection
policy was installed or enabled on the real host. Do not interpret this as a
production multi-Agent guarantee.

## Delivered source

- The existing Web Access worker (Tenant policy patch and TCP/UDP port
  publish/unpublish) acquires atomic canonical `tenant:`, `port:`, `box:`
  and `name:` claims after original preflight and before Provider dispatch.
  Independent destinations remain concurrent. Missing shared DB / stale
  root-published scope fail before remote side effects.
- Original app Access receipts remain authoritative. Only terminal
  ACK+observed policy-guard/route results release a dispatched claim.
  Transport loss, unverified observation and durable UNKNOWN retain the claim.
- The protected SSH/HTTP Gateway advertises `boxweave_box_lease` only in
  root-policy protected mode. Acquire/renew/release/status use the existing
  private coordinator and exact trusted target identity; max 600 seconds.
  The optional lease token goes only to claim acquisition for a tracked
  `boxweave_task_submit`, not to the guest process or private receipt DB.
- `boxweave_box_patch` reuses the **same** gateway tracked dispatch path,
  request-key deduplication, original trust checks, shared resource claim,
  and fixed small SHA-256 compare-and-swap guest helper. It is available
  only in protected mode. Its source release includes `box_patch.py`
  in the standalone/root reviewed distribution. Original receipt stores
  metadata and exit code, not source text, token, commands or stdout.
- Existing direct `connect(exec)` remains unchanged on legacy installations
  where root protection is disabled; in enabled mode direct/unclassified
  mutations fail closed. No new daemon or in-Box agent.

## Limits / unproven

1. Shared coordination requires a reviewed same-UID Gateway/Web app deployment,
   recent root-published Provider scope, preprovisioned 0600 SQLite,
   app `coordination.enabled`, and root-owned Gateway policy. These activation
   and rollback mechanics are **not** automatically applied by this release.
2. Out-of-band SSH, direct external Provider, old root helper versions, and
   Web paths outside listed lifecycle/BoxAction/Access operations cannot be
   claimed protected; no per-Agent authentication isolation is added.
3. A lease that expires never releases an UNKNOWN dispatch-intent claim.
   `boxweave_box_patch` protects at most 1024 UTF-8 bytes of one regular file
   under the exact target's home directory. It is not a generic binary upload
   or a replacement for host-native POSIX authorizations.
4. Project-wide exact source/fixture CI and a disposable Box command run are
   separate from live installed SSH/HTTPS/Web dual-client contention,
   protection policy enablement, mixed-old-version rollback and external
   gateway bypass testing. These remain U/BLOCKED until demonstrated.

## Executable source checks

```sh
python3 -m unittest discover -s tests -p 'test_access.py'
python3 -m unittest discover -s tests -p 'test_gateway_tasks_gateway.py'
python3 -m unittest discover -s tests -p 'test_gateway_protection.py'
python3 -m unittest discover -s tests -p 'test_box_patch.py'
python3 scripts/check_knowledge.py
python3 scripts/validate.py
```

**Acceptance:** source and verified-package checks must PASS at the final SHA;
retain separate CI and Box evidence with actual exit codes. Phase 2 functional
source slice is substantially implemented, but overall installed safety
acceptance stays PARTIAL pending the above explicit gates.

# Same-host restore key and MCP compatibility operator runbook (v0.42.11)

[Design and security contract](../design/MCP_COMPAT_AND_LOCAL_RESTORE_GUARD_04211.md) · [Verification](../verification/MCP_COMPAT_RESTORE_04211_2026-11-03.md)

## Diagnosing an upstream MCP upgrade failure

From an authorized BoxWeave MCP client, call **`boxweave_diagnose`**:

```json
{"source":"compat"}
```

The gateway reads the already initialized Provider's `tools/list`, at most eight pages, **without executing a Box command**. The response includes a bounded fixed-tool subset assessment, concrete missing/changed fields, a SHA-256 catalog fingerprint, and the validated MCP-server version if it was advertised during `initialize`. It never claims that the Containarium **daemon** version or response semantics were verified from this fingerprint.

- `MCP_COMPAT_TOOL_MISSING`: a required BoxWeave tool name was not advertised.
- `MCP_COMPAT_SCHEMA_CHANGED`: a key input field/type or a newly required parameter does not match the local expectation.
- `MCP_COMPAT_SUBSET_MATCH`: the sampled input subset matches; **network, permissions and actual response types are not proved**. Check the existing `boxweave_diagnose(source=gateway, recent=...)` event classifications.
- `UPSTREAM_DISCOVERY_UNAVAILABLE`: catalog could not be read; do not conclude an upstream breaking change.

No diagnostic result authorizes repeating a mutation with an uncertain outcome.

## Using the same-host local age identity guard

**Default DENY.** The privileged administrator, not the Agent, provisions the exact allowed restore. The upstream `restore_box_backup` tool must actually be present in the pinned Provider's advertised catalog; otherwise there is nothing to restore through this gateway and an upstream version integration is a **separate** project.

1. **Prepare a protected private key** from an existing backup process. Confirm the selected backup ID really exists, is a whole-Box record and corresponds to the source Box. Never send private key bytes in chat, a repository, a URL or a CLI argument. Do not generate a key that cannot decrypt the already existing backup.
2. Inspect existing owners and ACLs of `/etc/boxweave` and `/home/boxweave/boxweave`; do not replace an existing safe directory. Under the normal BoxWeave-managed host account/group, the operator provisions:

   ```bash
   sudo install -d -m 0750 -o root -g boxweave /home/boxweave/boxweave/.restore-keys
   sudo install -m 0640 -o root -g boxweave /path/to/existing-authorized-age-identity \
     /home/boxweave/boxweave/.restore-keys/demo.key
   sudo -u boxweave test -r /home/boxweave/boxweave/.restore-keys/demo.key
   ```

   Source identity file must have **exactly one** valid age private key, as required by upstream. Restrict the source file too; treat temporary copies as secrets.

3. Create a root-owned, group-`boxweave` mode `0640` `/etc/boxweave/restore-policy.json`, listing **only** the restore you intend to approve (maximum 16, no duplicate tuples):

   ```json
   {
     "version": 1,
     "allowed_restores": [{
       "box": "demo",
       "backup": "demo-box-20261103T180000Z",
       "as": "demo-restored",
       "key_id": "demo.key"
     }]
   }
   ```

   Check its owner and permissions. The gateway rejects a missing/foreign policy, unsafe directory, symlink/hardlink or key without the expected group visibility. If the BoxWeave gateway service runs under a nonstandard group, correct the deployment ownership first: no fallback to world-readable keys.

4. After administrator approval, the Agent can call **`restore_box_backup`** with:

   ```json
   {
     "box": "demo",
     "backup": "demo-box-20261103T180000Z",
     "as": "demo-restored",
     "key_id": "demo.key",
     "confirm": "demo-box-20261103T180000Z"
   }
   ```

   This intentionally has **no** `age_identity_file` (host path), no key bytes and no `force=true`. The gateway checks exact policy/confirmation, file metadata and readability; only then supplies the fixed local path to the existing Containarium MCP implementation. Its API still transmits the identity to the daemon for that one restore request, subject to its own authorization checks.

5. Inspect the actual backup/target state after a timeout; **do not automatically retry**. Remove the spent exact approval from the root-owned policy when the operation is complete, and keep independently protected off-host backup/key copies.

This is a **BoxWeave Agent gateway guard**, not a replacement for the upstream CLI/REST permission model or host root security. A remote Controller+MCP-host setup is explicitly out of this version's support scope.


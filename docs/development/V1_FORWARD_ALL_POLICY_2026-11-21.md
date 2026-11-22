# v1 one-time Host Forward authorization

Owner-approved post-RC10 policy adjustment. Baseline v1 0ea1feec61829bdbb065a796c10d302119d9d15b. No new native backend, tool, schema migration, live Host deployment or stable release.

## Contract

Trusted service environment BOXWEAVE_FORWARD_EXTERNAL_ALLOW:

| Value | New listener authority |
|---|---|
| Empty (default) | Loopback only (127.0.0.1 default, ::1 explicit) |
| Existing interface/protocol/IP:port entries | Exact assigned host address and port; wildcard entries invalid |
| Single literal all | Caller-selected valid TCP/UDP ports, unique assigned up non-loopback host addresses, explicit 0.0.0.0 or :: |

An operator selects all once at service startup and must separately permit restricted Project proxy devices. The MCP caller explicitly acknowledges each external bind with confirm_external_exposure; this is not independent human authentication. Wildcards can include current/future public interfaces and privileged ports. IPv6 dual-stack behavior is native-dependent. No automatic wildcard default, arbitrary destination, generic proxy device, firewall/ACL change, TLS or connectivity guarantee. Destination remains exact owned Box guest loopback port.

Removing the policy prevents new external creates, not existing native listeners. Explicitly remove existing owned forwards before withdrawing Project proxy capability. No new Host management tool. The original idempotency, native readback and UNKNOWN rules remain.

Either wildcard conservatively reserves the whole same-protocol/port across address families against all non-revoked claims, including pending/uncertain originals. Different protocols and ports remain independent. Native Incus still checks actual foreign/kernel listener conflicts. No replay or forced cleanup of UNKNOWN.

## Evidence

- Isolated development Box 1025aeed-11d1-42bd-af7b-8e76176d06d6, work_ref v1-forward-policy-1121, exact baseline checkout 08de64727ef63c1d3bd8641c9544bce7 exit 0.
- Focused native/state/control tests 2a465acc82cb113e83ac47fd1b32aa1b / ea9b64c3-384c-4d3f-a8a7-46c858739fb0 exit 0 FORWARD_CORE_UNIT_PASS.
- Extended native harness compiled and targeted tests repeated 423b275d74048b928f68492b94889ef0 / 47e42755-4551-4cc3-a59e-08ed97a4d517 exit 0 FORWARD_NATIVE_HARNESS_COMPILED. Guarded native tests not claimed executed by compilation.
- Native acceptance requires legacy exact-mode and broad-mode real TCP/UDP packets including IPv4/IPv6 wildcard on distinct bridge and loopback addresses, arbitrary port, overlapping claims, replay, occupied native listener, cleanup and original fault/dependent Delete regressions, all on isolated disposable CI. No public Internet reachability claim.

Final exact-head native and quality review PENDING. No merge or stable release authorization. Existing RC10/RC11 acceptance covers previous policy.

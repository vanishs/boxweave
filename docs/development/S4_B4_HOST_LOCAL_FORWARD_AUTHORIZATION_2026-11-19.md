# S4-B4 host-local forwarding authorization and fault closure

Status: implementation and Box tests passed; exact-source disposable native CI required before formal S4_B4_PASS. This supersedes the temporary loopback-only implementation boundary in the initial B4 report. No shared host change is authorized.

## Authorization boundary

- Default listener is 127.0.0.1; explicit ::1 is supported. Non-loopback addresses require both the authenticated caller's confirm_external_exposure and a trusted host-local startup allowlist. An MCP field alone cannot authorize exposure.
- The operator supplies BOXWEAVE_FORWARD_EXTERNAL_ALLOW through the trusted service process environment (operator-managed systemd drop-in for an installed service). Empty means deny non-loopback binds. Comma-separated entries: interface/protocol/IP:port. Invalid configuration refuses server startup. No wildcard, CIDR, port range, loopback alias, multicast, unspecified or link-local bind.
- The controller checks the exact protocol/address/port entry and that the live address is assigned uniquely to the named up non-loopback interface before admission and again before native create. It does not change firewall, ACL, NAT or management MCP listener. This is an address assignment restriction, not ingress-interface filtering or proof of physical LAN/public reachability.
- Target remains the exact owned Box and guest 127.0.0.1. Removing an allowlist entry blocks new creates, not existing running proxies; explicit authorized deletion and observation do not need the old entry.
- Deployment still requires operator-approved restricted Project proxy capability and firewall/route review. No production public bind, host change or release is authorized.

## Fault and regression invariants

The original immutable Box ledger identity (including tombstones) is available for authenticated replay only. Fresh mutations require a live owned row and exact native preflight. Proven no-dispatch failure, cancellation and ADMITTED startup recovery settle the original and listener reservation atomically. UNKNOWN always retains the pending claim. No automatic repair of historical uncertain claims.

The disposable native workflow exercises genuine parallel competing MCP create/delete, exact replay, IPv4/IPv6 non-loopback TCP/UDP on the CI bridge, consent/wildcard/address/port refusal, an occupied host TCP listener, and an actual SIGKILL of a worker after native create/delete terminal before ACK and after durable ACK before finalization. Reopen original SQLite and validate UNKNOWN, original-ID replay, fresh-key denial, exact native state and packet evidence. Trusted fixture teardown only, never production UNKNOWN replay.

The S4-A old native-log-loss negative now verifies the bounded private archive fallback, then explicitly removes only that disposable original archive to preserve the real typed-unavailable negative.

## Local evidence

Scratch L0 Box df927bc9-957e-4eb4-b579-10aa9a57645a; work_ref s4b4-closure-20261119. Full state/native/control/cmd tests exit 0 (original c96d8c4747f8b89e9bcb3f33ba8b5394). Final targeted tests, quota probe build, bash syntax, diff check exit 0 (original 7c8ae14ddba05d99ad05260a1c7f33a3). Native process-kill and packet tests are opt-in and not claimed locally executed on shared L0.

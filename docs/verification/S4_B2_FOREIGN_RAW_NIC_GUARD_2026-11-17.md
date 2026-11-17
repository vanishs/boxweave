# S4-B2 — native foreign raw NIC guard experiment, 2026-11-17

**Status: S4_B2_PARTIAL. B3 NOT STARTED.** This isolated test addresses the known foreign unfiltered NIC spoofing failure; it never modifies shared L0.

## Objective and native method

Earlier [native #44072000205](https://github.com/vanishs/boxweave/actions/runs/44072000205) observed foreign-Project unfiltered NIC traffic forging owned A IPv4/IPv6 source and reaching B. The root bridge auditor checks the Incus global UsedBy consumer inventory, but needed a repeatable foreign NIC negative.

On a brand-new disposable Incus 7.6 GitHub Actions runner only, the S4-B2 workflow creates a foreign Project and attaches its default Profile NIC to the test bridge using unfiltered raw parent. A separate actual Incus Go SDK root client reconstructs complete A/B/C native UUID, MAC, IPv4 and IPv6, then calls nativePeerBridgeAuditWithClient. Test succeeds **only if the guard rejects a foreign/non-BoxWeave consumer**. The workflow removes the NIC/Project before resuming original native ACL tests and exact cleanup.

This test alone is not a packet non-reachability proof nor a secure installed root/SQLite worker handoff, and cannot prevent an out-of-band privileged writer from racing after the snapshot.

## Pre-native evidence

- Sole Draft PR #153, source base fdf643afc5a6e36e195de26d3390c02dc147831f.
- Existing one-use L0 dev Box UUID 745d1550-97e8-4a02-9f19-4ef356a0274e, user-1001, Incus 7.6 and Go 1.26.9.
- gofmt, focused Go, YAML parser, diff check and full make verify (including race) PASS; original BW Exec 8a61d7643624c11ac4cc6aebd2c5d241, native 59a3000a-3083-4f70-bdf7-dfaef03680ba, final SUCCEEDED.
- **Exact new native CI result is NOT YET CAPTURED**. Gate remains S4_B2_PARTIAL, not B3.

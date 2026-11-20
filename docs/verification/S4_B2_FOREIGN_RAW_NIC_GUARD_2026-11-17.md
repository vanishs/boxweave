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


## Original Incus 7.6 native result — PASS for **root inventory refusal only**

The exact committed native run [#44121571100](https://github.com/vanishs/boxweave/actions/runs/44121571100), job `134174039947`, source `4418ca463cc366ad2efda6954b579d8959318679`, finished **SUCCESS**. The real native `GetNetwork().UsedBy` contained both `/1.0/profiles/default?project=bwowned` and the unexpected `/1.0/profiles/default?project=bwforeign`, plus three owned instance references. The product root auditor returned `foreign, duplicated or ambiguous peer bridge consumer`; CI logged `S4_B2_FOREIGN_RAW_NIC_ROOT_AUDIT_DENIED=PASS` and `S4_B2_FOREIGN_RAW_NIC_AUDIT_REFUSED_AND_EXACT_CLEANUP=PASS`. The original three-Box SDK stopped-barrier operation IDs were `c3ad44d7-5bc5-46c2-bd76-612af6e052c9`, `788c328a-67c9-47dd-935c-52e2f0143b40`, `5e9dedf8-8a64-4246-8dd1-cfff9056f822`, `a505e7f6-4bbd-40b3-aaec-45dc54a6182f`, `7fba558c-8c86-418d-9336-951866af1059`, `3297ce7f-3082-4015-b69b-433fa85bd4e1`. The independent original cleanup `S4_B2_QUIESCENT_EXACT_CLEANUP=PASS` and TCP/UDP four-family exact DROP+conntrack proof were also logged.

**Narrow verdict:** the native host root audit *does see and refuse* a foreign unfiltered raw parent NIC in this tested configuration. It does **not** prove the currently non-root installed MCP controller invokes that root audit before ACL writes, or prevent an authorized external root writer from racing between audit and action. Therefore `S4_B2_PARTIAL` remains; B3 is forbidden.

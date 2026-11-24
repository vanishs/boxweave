# S4-B2 - real kernel conntrack entries: exact pair purge and third-Box preservation

Date: 2026-11-17. Status: S4_B2_PARTIAL, B3 NOT STARTED.

## Why

The previous native stopped-A/B/C root test confirmed the four exact original-direction IPv4/IPv6 conntrack queries, protected read-only SQLite ownership and the fsynced no-replay root journal, but all four queries were empty (deleted=0). The success did not establish that any existing kernel entries were deleted.

Netfilter conntrack-tools documents kernel flow insertion with -I, bounded original/reply tuples, protocol and timeout, and exact filtered deletion with -D. See https://netfilter.org/projects/conntrack-tools/conntrack-manpage.html and the upstream source test https://nest-open-source.googlesource.com/manifest_repos/conntrack-tools/+/refs/heads/main/doc/cli/test.sh

## Dedicated root native evidence contract

A new test-only helper seeds six original-direction conntrack entries exclusively in the disposable GitHub Actions Incus 7.6 root host, with all three owned Box NICs still stopped and autostart disabled:

- A -> B IPv4 TCP ESTABLISHED and B -> A IPv4 UDP.
- A -> B IPv6 TCP ESTABLISHED and B -> A IPv6 UDP.
- A -> C IPv4 TCP and C -> A IPv6 UDP, as unrelated sibling-Box controls.

Each test entry has a fixed unique source/destination port, explicit original and reply tuples and a short kernel timeout. The real kernel must confirm all six before the one-shot journal dispatch. The existing native root actual-Incus/SQLite/bridge/ACL/DROP/zero-port proof must then purge all four A/B original directions, with DeletedScopes exactly 4 and a matching root-native digest. The helper must independently confirm those four are absent and both A/C sibling entries remain present. An exact tuple+port cleanup is registered even if only part of test setup succeeds; never flush conntrack globally.

The native CI workflow requires these non-SKIP markers:

- S4_B2_NATIVE_KERNEL_SEEDED_TCP_UDP_DUALSTACK_AND_FOREIGN=PASS
- S4_B2_NATIVE_KERNEL_EXACT_PAIR_FOUR_SCOPES_DELETED=PASS
- S4_B2_NATIVE_KERNEL_FOREIGN_TCP_UDP_FLOW_UNTOUCHED=PASS

The root journal still requires four direction probes, durable COMPLETE_VERIFIED, exact native digest and no replay. It must reject writable SQLite, MAC drift, foreign UUID and host bridge live ports. The helper never runs on shared L0: it requires root, GITHUB_ACTIONS=true, the explicit disposable native probe marker and exactly three owner test Boxes.

## Limits

The kernel entries are test-created TCP ESTABLISHED/UDP records, not actual network application sessions. This tests real kernel deletion of pre-existing *entries*, but does not prove installed live TCP/UDP business-session cutoff or per-NIC nft convergence after restart. It also does not prove cross-Project conntrack isolation: A/C is a non-target sibling-Box control. Installed controller/root authority, Scriptlet fault handling and independent full B2 signoff remain pending. No new MCP Peer writer or shared L0 host mutation. B2 stays PARTIAL; B3 remains blocked.

## First real Incus 7.6 runner finding and correction

Source a4a16ccd7dc9d74a87c0b4afcbe4c3a1d592d18b, native [Actions #44182928794](https://github.com/vanishs/boxweave/actions/runs/44182928794), job 134391747875 FAILED before the kernel seed. Native conntrack-tools v1.4.8 returned exit code 2 with:

    Illegal option '--family' with this command

This test failure also revealed a real production bug: nativePeerConntrackPurgeExact passed -f to conntrack -D, which conntrack-tools rejects. Earlier zero-session tests never executed this delete path. The official manpage specifies -f/--family for -L; IPv4/IPv6 numeric IP tuple selectors infer the family for -I and -D.

The correction removes -f only from kernel insert and delete calls. Bounded -L -f ipv4|ipv6 -s exact -d exact remains unchanged, as do re-read, UNKNOWN no-replay journal and exact cleanup. The Go runner unit test enforces that -D has exactly five arguments with only the original source and destination, and rejects unbounded commands. New exact-SHA CI is required.

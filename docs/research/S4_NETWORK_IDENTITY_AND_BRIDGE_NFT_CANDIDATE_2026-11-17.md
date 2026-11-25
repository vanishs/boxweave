# Moved: S4 network identity and Bridge nftables candidate

This research was reorganized into a broader knowledge base to avoid implying that S4-B2 should change its accepted implementation path:

**[S4 networking knowledge base: current path, security lessons, and future options](S4_NETWORK_SECURITY_LESSONS_AND_ALTERNATIVES_2026-11-17.md)**

The current Incus Bridge + NIC ACL + Scriptlet + scoped conntrack design remains the selected implementation in [Draft PR #153](https://github.com/vanishs/boxweave/pull/153); the fixed host-veth + nftables pair filter remains a **future research candidate only**.

This compatibility pointer preserves links to the original research file; no runtime, host or network change is authorized.

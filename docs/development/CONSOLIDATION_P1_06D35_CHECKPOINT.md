# P1 06D35: explicit Scoped admission and recovery fixtures
Parent: 5089c1508e987610f11c55899d28b365beac693a.
Converted existing Scoped create, lifecycle, process-cut, pin-repair, retirement and peer-terminal fixtures to exercise Scoped admission and recovery directly. Shared recovery test helpers accept an explicit mode while preserving Legacy defaults for Legacy-specific fixtures. Existing mixed-history guard and original historical peer lineage witnesses remain unchanged.
Focused State tests PASS (13.446s), original operation f9f89a2fbc216da788a45c77c88d2e60, exit 0. Edit operation 626f5ca81e3c2d582c00e0c392c6f815, exit 0. Full staged verification and remote-tree identity recorded in PR evidence.
This is source-test conversion, not native proof, retirement authorization or P1 completion.

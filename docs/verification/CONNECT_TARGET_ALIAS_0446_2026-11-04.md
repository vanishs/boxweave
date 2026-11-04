# v0.44.6 connect(exec) target naming repair — evidence

**Date:** 2026-11-04 UTC. **Live context:** BoxWeave installed Gateway 0.44.3, Containarium daemon 0.128.0 / Incus 7.5.1, host `dev`. **Published fixed source:** `20cd8782f8762b9ae51ae03527624d46da967784`. **Status:** old-version live cause verified, S/T/Release PASS, updated installed 0.44.6 L unverified.

## Exact real-Box reproduction

| Target and selector | Actual result |
| --- | --- |
| `bwreaccept-1104-container`, `box="bwreaccept-1104-container"`, `exec="id -un && hostname"` | `TRUST_OR_WORKLOAD_KEY_GATE` |
| Same target, `box="bwreaccept-1104"`, `exec="id -un && python3 -c 'print(\"hello world\")' && hostname"` | exit 0: `bwreaccept-1104`, `hello world`, `bwreaccept-1104-container` |
| `bwqa-1104-cmb-container`, `box="bwqa-1104-cmb-container"` | Same generic trust-gate error |
| Same target, `box="bwqa-1104-cmb"` | exit 0, exact expected username/hostname |
| First target, logical selector, `host="10.100.0.22"` (incorrect IP) | `TARGET_OVERRIDE_REFUSED` |
| First target, `boxweave_box_list_directory(path="")`, physical vs logical name | Physical generic error; logical succeeds with home entries |

Read-only `get_container(username="bwreaccept-1104")` confirms the actual physical name, logical user, RUNNING state, IP `10.100.0.21`, createdAt `1793790679`. `debug_container` showed backend host user and `containarium-shell` present, with no sshd rejections. The bounded gateway diagnostic trace `bcc8a9aefe0fcb579df074c2528efe23` was only classified `UPSTREAM_JSONRPC_ERROR`, `root_cause=not_verified`; that alone did **not** identify the root cause. The controlled live naming A/B probes together with reviewed old gateway source identify the mismatch without raw host logs or private credentials.

## Root cause and fix

In prior Gateway `mcp_gate.py`, `connect` passed `args.box` untouched as `get_container(username=args.box)`; Containarium expects a logical **username** at that API boundary, whereas `list_containers` shows a concrete `<username>-container` name. A failed Provider lookup yielded an MCP JSON-RPC error with no `result`; the Gateway's result dereference produced `KeyError`, masked as generic `TRUST_OR_WORKLOAD_KEY_GATE`. This is not evidence of transient host-key trust issues.

v0.44.6 uses `resolve_managed_box` to accept either a registered logical username or its **exact registry-verified** concrete `name` with matching `user`. A name also matching a different logical target fails `BOX_TARGET_AMBIGUOUS` before Provider dispatch; unknown targets fail closed. After resolution, Gateway queries upstream using the logical selector; the unchanged `trusted` security intent validates **name, createdAt, user, IP, RUNNING status and host/user overrides** before forwarding the normalized logical name to `connect`. Malformed/upstream failed reads return typed `TARGET_READ_FAILED`; invalid trust registry content fails closed. The bounded Box read/list facade is normalized in the same way; other task/lease tools still keep their existing selectors. No key, Provider, Incus, sshd, firewall, or host trust mutation was performed.

## Executed testing and release

- **L on the old installed gateway:** two independent live Boxes, two naming variants each, exact username/hostname and Python output, wrong-host negative, and read-list positive/negative. Name mismatch verified.
- **T on real Ubuntu 24.04 test Box:** `bwreaccept-1104` cloned the exact candidate SHA `20cd8782f8762b9ae51ae03527624d46da967784` to `/home/bwreaccept-1104/boxweave-connect-0446`. `python3 -m unittest tests.test_online tests.test_gateway_compat_restore_flow -q`: **28 tests, exit 0**. `python3 scripts/check_knowledge.py`: **205 documents, 26 REQs, 26 ACs, exit 0**. These are source tests, not an installed new Gateway.
- **T CI:** [GitHub Actions run 41855559707](https://github.com/vanishs/boxweave/actions/runs/41855559707) exact SHA, verify and publish both **success**. Full Python 3.12 **736 tests**; Python 3.14 **73 tests**; seven real Chromium fixture workflows, frontend build, **57** Python modules verified within `boxweave.zip`, and offline Python 3.14 imports all passed.
- **Release:** [v0.44.6](https://github.com/vanishs/boxweave/releases/tag/v0.44.6) published non-draft with **25 assets**, including `boxweave.zip`, `release.json`, `SHA256SUMS`, `mcp_gate.py`, `box_reads.py` and installer. Its git tag resolves to the exact source SHA above. No previous immutable tag was retargeted.
- **U / further gate:** The actual outer host still runs the older installed Gateway. After an expressly authorized installed upgrade, re-run both name forms on the *same* exact Box, test read-file/list, reject wrong host/unknown aliases, and record installed/running version with stdout/exit. No installed 0.44.6 guest access can yet be claimed. R-02 task tracking and independent nested workloads remain separate open gates.

**Resources:** Original test Box `bwreaccept-1104-container` remains RUNNING at 2 CPU / 4GB RAM / 30GB disk, original IP and creation time, with only a task-owned source checkout added. Older operator failure scenes were not changed. No real host/Provider/SSH/trust settings updated; `PUBLISHED_RELEASE` remains **0.43.1**.

# v0.42.10 pinned upstream shell retrieval — verification record

**Date:** 2026-11-03. **Scope:** source and isolated Test Box, no real owner-host mutation.
**Status:** SOURCE IMPLEMENTED / TESTED; limited real root-sandbox proof; full installed host-hop acceptance remains open.

[Decision](../design/CONTAINARIUM_UPSTREAM_MCP_FOLLOWUPS_2026-11-03.md) · [Master roadmap](../design/FEATURE_ROADMAP_MASTER.md) · [Risks](OPEN_RISKS.md) · [Agent workflow](../process/AGENT_TEST_WORKFLOW.md)

## Identity and upstream compatibility

- Code base: BoxWeave main `5fd03c5b8df6d95de1581789e9a1aa92520bd26d` plus owner-approved decision branch, v0.42.10 candidate.
- Upstream source: exact Containarium `main` commit [e5deb167](https://github.com/FootprintAI/Containarium/commit/e5deb167c973033d9a847abdcb2420e917095546) compared against the released `v0.120.1`.
- Official `scripts/setup-ssh-container-proxy.sh` in **both refs** SHA-256: `93365c56cd70aafac9880bc4dfcc4b1bcbb55dfc86e37b17b207c8245989f04a`.
- Extracted 106-line `containarium-shell` in **both refs** SHA-256: `2212651e2083b80d76363523bf794ec4d530f8eb74bbac0def0c347bb544caa6`.
- Pinned Containarium **binaries** remain `0.120.1`. This is a byte-equivalent wrapper retrieval change, **not** integration of Containarium v0.127 or a compatibility certification.

## Scope of source implementation

- Removed `bwmini/containarium-shell` from repository and Release/root helper inventories; new Release assets must exclude it.
- Download by **immutable upstream commit**, verifying both original setup script SHA-256 and extracted wrapper SHA-256. Only the wrapper heredoc is extracted. **The full upstream privileged setup script is never executed**.
- BoxWeave-managed local Provider: fetch/install only if missing, automatic with no extra flag. External Provider: inspect only unless operator opts in via retained `--repair-containarium-shell`; new writes require supported local CLI version proof. New Provider upgrade is forbidden.
- Foreign script, symlink, unknown source, unsafe registry, checksum/format drift, and unsupported Provider version fail closed for **optional host-hop only**; core installation continues and reports an incomplete host-hop result. Preserve old login shells and prior managed-file identity.
- Updated `SERVER_INFO_VERSION` plus source, frontend and package metadata to `0.42.10`.

## Executed checks

**Isolated test Box:** `bwshell-1103-verify-container`, freshly created Ubuntu 24.04, CPU 2 / RAM 4GB / disk 30GB, Python 3.12 and dedicated virtual environment. Parent Provider/Incus and owner host were not upgraded.

| Check | Actual observed result | Evidence class |
| --- | --- | --- |
| Direct fetch of two immutable upstream refs, extraction and SHA-256 | Identical setup hash and 106-line wrapper hash | T PASS (network + bytes) |
| Focused shell/installer/adoption/release-boundary unit tests | `python3 -m unittest tests.test_containarium_shell tests.test_online tests.test_online_adoption tests.test_arch0_boundary_guards.BoundaryTests.test_root_helper_assets_match_source_release_and_installed_copies -q` → **39 PASS** | T PASS (fixtures) |
| Full Python regression suite with locked dev dependencies | `.venv/bin/python -m unittest discover -s tests -q` → **661 PASS** | T PASS (fixtures) |
| Knowledge and syntax checks | `python3 scripts/check_knowledge.py`, `python3 -m py_compile bwmini/online.py`, `bash -n install.sh`, `git diff --check` → PASS | T PASS (source) |
| Actual root filesystem/network operation in test Box, root-owned temporary path | Upstream pinned source fetched and verified, shell installed; `installed_from_pinned_upstream_registered`, root owner, mode 0755, expected hash | L PASS (bounded sandbox only) |
| Repeat on matching root-owned temporary target | `present_managed`, no downloader invocation, no duplicate registry entry | L PASS (bounded sandbox only) |
| External Provider without opt-in, missing temporary target | `missing_external_unmodified`, no file write | L PASS (bounded sandbox only) |

## Additional actual root-sandbox checks (same disposable Box)

- Executed root-owned temporary release manifest generation with a placeholder test-only ZIP: **18 verified entries**, no `containarium-shell` asset.
- Executed **external Provider opt-in** with a local test-only CLI that prints the pinned `0.120.1` version: verified official pinned download and install, `installed_from_pinned_upstream_registered`. No real external Provider was changed.
- Executed the **pin-mismatch** branch against a locally cached source by altering only the expected test hash: `status=incomplete`, `code=CACHED_ARTIFACT_CHANGED`, no target created. The independently usable core install path does not depend on this optional success.

## Remaining gates and safety

**NOT proven:** end-to-end host-hop SSH and wake, sshpiper, per-account Incus sudo authorization, cross-tenant denial, nested Incus installation, completed installed `v0.42.9` → `v0.42.10` upgrade, published GitHub Release asset integrity, or independent stable pin promotion. These remain explicit real-environment acceptance work. A passing direct `connect(exec)` is **not** host-hop SSH proof.

Current source/fixture evidence does not justify changing the operator's Containarium version or overriding ownership checks. Retain the immutable historical `v0.42.9` release and any already installed account login shell; never silently delete it. GitHub PR CI, merge and immutable published-release verification are separate gates from the source results recorded here.

[Back to owner decision](../design/CONTAINARIUM_UPSTREAM_MCP_FOLLOWUPS_2026-11-03.md)

## 2026-11-03 post-publication and real installed upgrade acceptance (supersedes pending checkpoints)

**Status: PUBLISHED / L PASS for the bounded release and installed upgrade; SSH host-hop positive and negative access remain U.** This is a later evidence checkpoint. Earlier "not proven" entries above were true before the following checks; they are not the current state of the release.

- **PR and CI:** [BoxWeave PR #52](https://github.com/vanishs/boxweave/pull/52) squash-merged as `6ac4cb6986fe76a810321538e4af9fc3a1bfb602`. PR [CI run 41679887611](https://github.com/vanishs/boxweave/actions/runs/41679887611) and main [CI + publish run 41680412941](https://github.com/vanishs/boxweave/actions/runs/41680412941) both completed successfully.
- **Immutable publication:** [v0.42.10 Release](https://github.com/vanishs/boxweave/releases/tag/v0.42.10) published on 2026-11-03, non-draft/non-prerelease, tag and main both at `6ac4cb6986fe76a810321538e4af9fc3a1bfb602`. The release contains **21 assets**, **no `containarium-shell` file**. On the test Box, the official downloaded `release.json` verified SHA-256 `1b45dea27e88efd943db303a6e88000bea8500d9f097ff1f716ccb2c847d100d`, source matching merge SHA, version `0.42.10` and 18 managed file entries excluding the shell.
- **Fresh real legacy install on authorized disposable Box:** after applying `security.nesting=true` to the exact **`bwshell-1103-verify-container`** and restarting **only that Box**, the official published `v0.42.9` streamed `install.sh --version 0.42.9` installed real local Incus, PostgreSQL and Containarium `v0.120.1`, finished `state.json: phase=ready, version=0.42.9, provider_mode=standalone`, and installed the upstream wrapper with expected pinned digest into `/usr/local/bin/containarium-shell` under root.
- **Real release upgrade:** ran the official published `v0.42.10` `install.sh --version 0.42.10` on the **same test Box**. Finished `state.json: phase=ready, version=0.42.10, provider_mode=standalone, source=6ac4cb6986fe76a810321538e4af9fc3a1bfb602`. The observed JSON `containarium_shell.status=present_managed`, `code=null`, `host_hop_ssh_verified=false`. The previously installed wrapper remained root:root mode 0755 with unchanged SHA-256 `2212651e2083b80d76363523bf794ec4d530f8eb74bbac0def0c347bb544caa6`; its old `managed_files` identity was retained. No wrapper file existed in the new `0.42.10-<source>` release directory.
- **Installed service health:** `sudo boxweave status` reported installed version `0.42.10` and local authenticated Web challenge HTTP **401** as expected; `sudo boxweave doctor` returned exit **0**, `overall=passed`, **8/8 required checks passed** (public descriptor, Web reachable, host timer and user manager, app health and app serve/worker units). `boxweave-provider.service=active`. `/usr/local/bin/containariumd version` returned **`Containarium v0.120.1`**. Optional authenticated MCP positive check was **not requested**; no owner-host Provider upgrade occurred.
- **What remains unaccepted:** actual real host-hop SSH session and wake-on-SSH, sshpiper, cross-tenant negative authorization, additional Provider-version compatibility, offline real full application bootstrap with missing optional wrapper, and default stable-channel promotion. `PUBLISHED_RELEASE` deliberately remains `0.42.1`, so the unqualified default installer does **not** yet select `0.42.10`. The versioned `--version 0.42.10` route is fully published and was actually used.

**Resource policy:** the disposable Box is the only nested installation/test target, distinct from existing user Boxes. No real owner-host Containarium binary, signing authority, public listener or stable selector was changed.

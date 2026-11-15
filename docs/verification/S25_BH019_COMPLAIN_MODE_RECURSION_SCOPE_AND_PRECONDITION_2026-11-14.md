# BH-019 — requested repeated nesting with AppArmor complain: test scope and independent precondition failure (2026-11-14/15 PT)

**Status: COMPLAIN RECURSION NOT TESTED.** This is an accurately classified partial preparation and real native start failure, **not** proof that complain succeeds or fails. The direct L0 physical-host Incus command channel needed for the exact request was **not available** through the connected BoxWeave L0 RC5 MCP. Only the BoxWeave `box_exec` interface is connected, and it runs a typed command **inside an existing owned Box**, never in the host's Incus administrative context. BoxWeave's confined Project and native adapter refuse arbitrary `raw.lxc`/unknown AppArmor configuration. This report must not be relabeled as a native L0 7.6 complain-control experiment.

## Prior results / why requested matrix is new

- **Previously tested:** ordinary nonprivileged `security.nesting=true` Boxes, new own AA policy view (relative `unconfined`) vs deeper enforced-label `(enforce)` Box inheriting an AA namespace at level1; several read-only N×M tests and an older distinct `raw.lxc=...unconfined` control, with negative parser243 under deeper Incus.
- **Not previously tested:** an identical **`complain`-mode Incus-generated per-Box AppArmor Profile at each nesting depth**, with each layer launching an independent Incus daemon and its own next Box.
- **Upstream semantics:** Incus `raw.apparmor` appends **profile entries** to a generated AppArmor profile. It is not a first-class `AppArmor=complain` instance key and is not equivalent to switching that profile's runtime execution mode. AppArmor complain is a **per-profile** enforcement mode; a global complain switch would alter a shared host's security enforcement and was **never attempted**. No production, shared host or inherited L1 policy was modified.
  - https://linuxcontainers.org/incus/docs/main/reference/instance_options/
  - https://www.apparmor.net/man/master/apparmor/
  - https://linuxcontainers.org/incus/docs/main/reference/projects/

## Experimental scope: only one new fully owned outer disposable Box

Connected `BoxWeave_L0_RC5` (Incus 7.6 controller). Created fresh `bh019-complain-scope-1114`, exact UUID `b3501c8a-b32d-480f-aa2b-2313f0f00899`, with nonprivileged default, `security.nesting=true`, 1 CPU, 1536MiB memory, 10GiB disk, initially internet=false; temporarily enabled egress **on this exact Box only** for package/image retrieval, then completely removed the Box. No `security.privileged`, no `raw.lxc`, no host/global AppArmor mode change. Scope `BH019-COMPLAIN-SCOPE-20261114`, agent `bh019-complain-probe`.

Inside the one-use Box:
1. Observed Ubuntu 24.04.5 LTS, AppArmor namespace explicitly named this one-use Box, `.ns_level=1`, current relative label `unconfined`. There was **no** preinstalled Incus, `incusd`, or AppArmor management package, and no preexisting `/var/lib/incus` socket.
2. A dry-run resolved Ubuntu's Incus **6.0.0** package and AppArmor tooling, 28 additional packages. Installed *only inside this disposable Box*. This **6.0** differs from the real 7.6 inner/outer Incus versions under BH-019, so any result is strictly exploratory.
3. New `incus admin init --auto --storage-backend=dir` initially failed due to missing `dnsmasq` executable after no-recommends package installation. Then installed `dnsmasq-base`; a second identical auto init **correctly refused** with `Storage has already been configured`, revealing a partially initialized but valid `dir` pool. To avoid false idempotence assumptions, individually created `incusbr0` (success) and attached the owned `default` Profile's root disk and private NIC (both success).
4. Launched one internal **nonprivileged** `bh019-complain-child1` from official `images:ubuntu/24.04`, with `security.nesting=true`, **default AppArmor (not complain)**. The image was fully retrieved/unpacked. Actual `incusd forkstart` exited 1; `incus info --show-log` showed:
   - `cgroup2_devices ... bpf_program_load_kernel: Operation not permitted`;
   - **`newuidmap: write to uid_map failed: Operation not permitted`**, requested map `0 1000000 1000000000`;
   - `Failed to set up id mapping` → child state STOPPED, never reached a running AppArmor subject.
5. Tested one ordinary nonprivileged Incus config contrast, **without modifying AppArmor**: `security.idmap.size=65536` on exactly that stopped child. Subsequent real `incus start` again failed `forkstart`; logs still showed the identical huge `newuidmap 0 1000000 1000000000` request. A separate readback proved the desired setting **65536 had been stored**, while `volatile.idmap.next` still contained a `Maprange=1000000000` for both UID and GID. This is a **specific stale/ineffective native map observation in this v6.0 lab**, not evidence that AppArmor complain fails, and not sufficient evidence to prescribe direct manipulation of `volatile.*`.
6. Native AppArmor generated profile file `/var/lib/incus/security/apparmor/profiles/incus-bh019-complain-child1` existed (20,464 bytes), but child never ran. No mode switch, profile load/replace/remove, global security toggle, or raw override was attempted. The AppArmor `aa-status --json` listed `incus_dnsmasq-incusbr0` as `enforce`; unrelated distro `complain` profiles cannot be mistaken for the requested child.
7. Deleted exactly the stopped inner `bh019-complain-child1` through the *disposable Box's* independent Incus. Separate `incus list --format=json` returned `[]`. Then stopped/deleted exactly the one-use outer Box via its original L0 controller operation receipts. Confirmed final authenticated inventories unchanged: **L0 4 historical Boxes, L1 9 historical Boxes**. No new UNKNOWN and no historical UNKNOWN replay.

**Result interpretation:** The case **never reached a running first child** due to a v6.0 UserNS UID mapping prerequisite (with independent cgroup-device log), before per-child AppArmor complain could be applied or contrasted. This does not disprove the owner's idea. It does prove that repeating identical child-create attempts in this specific disposable Box would be low-information without first obtaining a valid and supported ID mapping. It also demonstrates **L0 BoxWeave MCP ≠ direct L0 host-level native Incus administration**. An exact 7.6 complain-recursion experiment needs a separately isolated parent Incus admin scope and an independently verified profile-mode switch only for its own one-use profiles; never turn off shared host/global AppArmor policy to simulate the experiment.

## Original operation receipts (all outer SUCCESS; classify inner errors separately)

All **25** authenticated original L0 BoxWeave Operations at `work_ref=BH019-COMPLAIN-SCOPE-20261114` were terminal `SUCCEEDED`, each with the separate original native Incus Operation ID; **21 Execs** had their original guest outputs read where relevant. Note that wrapper processes returned guest exit0 after capturing **inner native command exits 1** in structured JSON. This is **NOT** a PASS for failed `incus admin init` or `incus launch`.

| Experiment / action | Original BoxWeave ID | Native Incus ID |
| --- | --- | --- |
| One-use Box create | `20b2442e0297926e632c22af5621fc9d` | `f13ee7f3-af07-4fff-8560-e465391c5de4` |
| initial packages/sockets preflight | `8ceec203a69f63ac4959b72c81ce2c9b` | `7566de82-8788-41f5-afd3-0a5681a6685a` |
| apt-cache package candidate | `194468c1e18453584cb9d9c8efb0f547` | `200c71ae-7ddc-466c-9eab-a03b3386acd0` |
| temporary one-Box egress on | `be8298ca969d92711e0adabc8cf0ca36` | `c9179b3f-5d21-42d2-a4c8-10cc17226d55` |
| apt install simulation | `2865ffd3ff16b2d9f247d39ca8f872f7` | `126e0955-4488-4096-bd35-46834af2bdf4` |
| install Incus6 + AA tooling in one-use Box | `ed861a04c53e46d74252f6e19071910e` | `a4c59a36-1683-4494-a5ed-fe4af02f0a1d` |
| local incusd readiness | `22ae0a9fa9ae535555f7ee22425b6c01` | `2ad904c2-b3d7-42d9-a434-6898c683085e` |
| first init: missing dnsmasq | `7d6ffd314d851b79202076c051af06f9` | `8ec52022-d3c2-42f5-8b42-d27347774ffd` |
| verify missing dep / partial pool | `490a54effcd7e009aecd53c4547c51d7` | `f83f3ac4-4f16-47e1-8159-d185588064b8` |
| install dnsmasq in one-use Box | `447b548911630196731ce8c153a88daa` | `76904132-af1a-4822-b924-3cdfdd7ed290` |
| second init: partial state, no blind retry | `d4f7ec93299638a0a655f4ea0b611310` | `55b3b41c-f302-46c0-bfe9-3c4dfa97310e` |
| read pool/net/default profile | `11653da7b006120457b159ef6db6107e` | `207807c8-0979-40f2-ba49-14a9adcfd5f9` |
| create isolated guest incusbr0 | `41be2a2d6a58b88c28a2bc82c1c01e4b` | `e3f58dfd-ddb3-45c7-86d6-bf0fdb491a0e` |
| add private root disk | `f9c9b4bbf5013074484bd15d95b88067` | `a39c88f9-b8ff-4302-bba4-b8348dd75a44` |
| add private NIC | `3387982d22b483f03a3dd1a80ca7250d` | `8601b458-03ea-4668-b053-6f5b2679ead1` |
| real first child launch: inner forkstart fail | `ef4beb3426b276860e75c16bb09602c2` | `f9cb5636-e91b-4bd4-9faf-be7f813c7e85` |
| source log from exact child | `9ec6ffddbe8e152fcd2dacd439fdee2b` | `fe0112ce-2c14-4b59-971b-1d6a4a56f286` |
| set stopped child's ordinary idmap size | `e6a31a0261d4c3ab19de4db0a88cb7e4` | `82d67e2f-76b5-44b1-a36a-d9d1765c02e6` |
| retry start once: still forkstart fail | `8a200a447531056dabe73f25d4947b35` | `ad4cfb43-5fff-4346-99bf-5f24d1afd1c4` |
| source log after retry | `98c5ce7fa76c977007a8b3941ab5d799` | `353d24af-d1f8-4a4a-8244-0615d31b15dc` |
| mode + native stale idmap readback | `2110da6e10bc4cc313cb366aa89a8a2f` | `6aa2a780-fdb8-4b7b-8fd5-7a7d3f688fcc` |
| exact inner child delete | `a1fd46354bc6d7c6646394b89613800d` | `416b1ae0-e05a-4025-a625-cce75bf0a01b` |
| independent inner inventory: `[]` | `d0d3cf89ebcf5f0c67ba248de23a4015` | `59035a05-733b-4743-ba31-17a513f645c7` |
| exact outer Box stop | `1f697634fcbff6afd2918cd2c9b4c443` | `f7adacfa-2acb-4922-879e-8ff87b9d6683` |
| exact outer Box delete | `e3a6c9e96d881f020bf15d7419aeffe7` | `ab687dd4-526e-45cb-ac23-1a8ae43db537` |

**Safety:** no privileged container, global security-mode change, host AppArmor profile change, shared Incus project weakening or inherited L1 policy mutation. Test Box deleted, no residual installed software on host. **BH-019 BLOCKED; PR #146 remains Draft and unmerged.**

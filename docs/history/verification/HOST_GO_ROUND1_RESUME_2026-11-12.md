# HOST_GO Round 1 resumed Box-first installed/native acceptance — 2026-11-12 UTC

**Verdict: ROUND1=BLOCKED; LAB_PASS=NO; HOST_GO=NOT_AUTHORIZED.** This is a continuation of the **same user-invoked Round 1**, not a trigger for Round 2. It materially strengthens native security evidence without claiming missing same-head SDK client operations, direct public IPv6 WAN or fully independent I+N+V.

## Scope, exact versions and actors

- Draft PR #133, tested source `e8782860f9c43a21af1193ff9cb9225f3619f81c`, branch `feat/v1-s25-native-readiness-20261111`; **installed candidate SHA-256 `68a1f2b8e9b6e5b5ecbfbc88d5ec3de4549989c944b55159a828ddf0c1c31c31`**, embedded `main.commit=e8782860f9c43a21af1193ff9cb9225f3619f81c` and `version=1.0.25-hostgo-r1`. Go 1.26.9, pinned official MCP/Incus SDK modules. Two independent existing isolated Boxes (`bwracc-1106-v2` and `boxweave-test-ser2`) rebuilt the **bit-for-bit same candidate digest**, with `CGO_ENABLED=1 go build -trimpath -ldflags '-X main.version=1.0.25-hostgo-r1 -X main.commit=e8782860f9c43a21af1193ff9cb9225f3619f81c'`; no existing test checkout was reset. `make verify` on the clean exact-source `boxweave-test-ser2/work/hostgo-r1-exact` PASS: tracked format, YAML, mod tidy-diff, vet, mod verify, all-package CGO race tests.
- Finder/native actor 1: `bwracc-1106-v2` disposable lab (no original native guests), signed Incus 7.6 APT `1:7.6-ubuntu24.04-202611072126`, approved Zabbly key fingerprint `4EFC590696CB15B87C73A3AD82CC8797C838DCFD`; official Go tarball `go1.26.9.linux-amd64.tar.gz` independently checked against official SHA `42d158b4d8f7b61ac0a830567c940a86098fb7aac52e467a5ebec03ef5cc2f8d`. Restricted service user `boxweave` UID 1002 primary group `incus`, `user-1002` Project with private bridge/ACL `bw-hostgo-r1-v2-exclusive`.
- Finder/native actor 2: separately isolated **already nesting-enabled** `boxweave-test-ser2`, genuine Incus 7.6 APT `1:7.6-ubuntu24.04-202610301520`. Historical `boxweave` account there is **not** an approved confined account, and was **not changed**. Instead a new restricted test-only Incus actor `hgr1lab`, UID 1004/primary group incus, obtained independent `user-1004`, own bridge `incusbr-1004` and exact ACL `bw-hostgo-r1-ser2-exclusive` without touching existing `demo-1-container` in Project `default`. Actor cannot read the foreign default Project.
- Dedicated owned candidate nested L1 (in `ser2`): native UUID **`81b57e94-f2b5-4424-b507-c1a2813ba293`**, unprivileged `security.nesting=true`, 2 CPUs/3072MiB, private/filtered NIC. Inner Incus 7.6 signed APT `1:7.6-ubuntu24.04-202611072126`, service actor `boxweave` UID 1001, restricted `user-1001`, `incusbr-1001`, ACL `bw-hgr1-l1-exclusive`. Exact same candidate SHA installed with `install-stage`/`install-activate`; real nonroot `doctor` READY. Inner MCP listens only on `127.0.0.1:18650`.
- A **separate root/native readback process** (not the restricted test actor) independently compared exact parent UUID, nonprivileged nesting, owned ACL `used_by`, native Project/default Profile, empty inner child inventory, installed executable SHA and loopback listener. **Not a separately hosted independent I+N+V Verifier for Box mutations**; final BH-015 remains PARTIAL.
- Observed test time through `2026-11-12T11:06:56Z`. This report itself commits later; a documentation-only Git commit changes Git HEAD but does not retrospectively change the installed candidate's embedded SHA or its binary bytes.

## Preserved gateway UNKNOWN and safety

- Previous two managed Box creations still `UNKNOWN` on read-only original tasks:
  - `fb7b54a1f5a06da16bc7c617e5ae4b27`, key `a10f7112f1a04cc78d4dfda6b7110201`, `bw-hostgo-r1-finder-1112`
  - `3fede508363e572babcf61e6177ea974`, key `b10f7112f1a04cc78d4dfda6b7110202`, `bw-hostgo-r1-verify-1112`
  Both remain UNKNOWN even though earlier read-only gateway get/debug reported `missing`. No replay, deletion, adoption, same-name recreation or receipt removal.
- Historic UNKNOWN `7e1cbf043d1fd4ea3d9ec0093a04371e`, `f811afeeb778c422fe99b9881baa40f2`, `192a6601c54078f8eaf1437a6a3dc13b`, `a2fdf885b87a0b66fb3a17f913316c55` untouched and not considered resolved.
- All authorized native fixture creates had affirmative Incus CLI acknowledgement and exact UUID readback. No uncertain native write was retried. No physical host, parent shared daemon, historical `demo-1`, foreign Project or old hook worktree modified.

## Real results — keep I, N and V separate

| Scenario | Direct witness | Grade / status |
|---|---|---|
| Signed Incus 7.6, owned Project, restricted OS identity, reviewed root/profile/ACL/image; wrong review SHA refused with zero ACL effects | Two separate isolated native environments; inner L1 and outer v2 8/8 native prerequisites, `doctor ready=true`; ACL `used_by` exclusive; no foreign default Project access | Installed CLI + N + root readback; **PASS for bounded prerequisite subtests** |
| Real installed exact-candidate MCP boundary | v2 and inner L1 unauthorized/malformed HTTP: missing Bearer 401, invalid Bearer 401, invalid Host/Origin 403/403, wrong path 404; loopback `127.0.0.1:18650` | **I+N partial, HTTP negatives PASS**, no authorized SDK create |
| Default ACL/NIC no explicit Internet vs individual Internet-enabled NIC | Native A `hgr1-a` (UUID `b0821421-8348-43a9-99d6-71fc5b019a33`) no egress: v4 WAN timeout and v4/v6 gateway ping 0/1. B `hgr1-b` (UUID `c86c153c-4277-49bc-b234-d28acb061c19`) with *only its own* NIC egress allow: direct unproxied v4 HTTPS 200, v4/v6 gateway ping 1/1; A remained denied. | **N witnessed**, not same-head MCP lifecycle |
| Same-bridge sibling IPv4 and IPv6 isolation | A→B and B→A real IPv4/IPv6 ICMP probes 0/1 in each direction; native `used_by` includes only A, B and owned default Profile | **N negative PASS** |
| Foreign Project guard | Restricted `hgr1lab` denied Project `default`; B→preexisting foreign `demo-1-container` private IP ICMP 0/1; foreign native instance remained running, unchanged | **N negative PASS within this lab** |
| Source IPv4/IPv6 and MAC anti-spoof | On owned B, legitimate private v4/v6 gateway ICMP 1/1 before/after. Private-only transient non-owned v4/v6 source probes failed, forged MAC probe failed, original IP/MAC safely restored. Actual native `nft` device source/MAC filtering rules read back. | **N packet negative with positive controls PASS** |
| Actual direct public IPv6 WAN | B with egress allowed and the outer L0 both failed direct IPv6 HTTPS, despite actual IPv6 route; v4 control success. No provider WAN IPv6 witness. | **NATIVE_WAN_ENV_BLOCKED_OR_UNDIAGNOSED**; BH-009 BLOCKED |
| Separate official SDK MCP Agent: 13-tool discovery, authenticated Viewer denial, original-key Create/Exec and V native postcondition | Official SDK test binary built at SHA-256 `c3618f75f6e0f4e859aa02850b96ac8a5f2b284b671200b1441c4d9c0d272a58`, but tool-level safety checks blocked attempted credential-file handling/creation. The agent did **not** run authenticated mutation; no L2 was created. No credential value copied into messages, artifacts or public logs; no auth shortcut. | **I+N+V BLOCKED**; don't infer from the installed MCP 401/403/404 |
| Native environment false start and recovery | Fresh `bwracc-1106-v2` parent Box had `security.nesting=false`; owned hgr1-a UUID `f35945a1-b6c5-49a5-a569-8d6b222675ac` failed START with `Operation not permitted` on proc mount/cgroup BPF, confirmed STOPPED via original identity. **No parent config permission bypass**. Switched to pre-authorized nesting-enabled `ser2` and succeeded with nonprivileged L1+inner Incus. | **Environment limitation identified; test fixture cleaned** |

### Current case disposition (no newly complete formal BH case)

| Case | Round 1 status |
|---|---|
| BH-001 | PARTIAL — exact current source/binary/Incus facts, 13-tool official SDK ListTools still absent on this installed candidate; original operation inventory crosswalk incomplete |
| BH-002 | PARTIAL — two isolated host inventories and quota warnings; physical dedicated host not approved |
| BH-003 | PARTIAL — actual native installed first-Box prerequisite 8/8, wrong SHA negative, foreign Project denied; full independent rollback not reaccepted |
| BH-005 | PARTIAL — installed HTTP negatives; no actual authenticated official SDK protocol/MCP client |
| BH-009 | **BLOCKED** — native IPv4 positive, v4/v6 ACL, spoof and isolation positives/negatives passed, but exact installed MCP packet lifecycle + independent V and actual IPv6 public WAN not witnessed |
| BH-010 | PARTIAL — nonprivileged L1 with separately installed pinned v1/Incus, inner 8/8; no MCP-created L1 or L2 Create/Exec |
| BH-011 | PARTIAL — OS Project and sibling/foreign denies proven native; Agent/UUID spoof and parent socket negatives not fully signed |
| BH-015 | PARTIAL — distinct native observer/readback, no separately hosted matching-digest I+N+V verifier |
| TC-011 | PARTIAL/NOT_REACCEPTED — old acceptance retained; new two-client work/identity negatives not attempted |
| BH-017 | NOT_AUTHORIZED — no dedicated real physical-host approval or mutation |

## Cleanup, retained provenance and exact next work

- `ser2` native-only A and B were independently verified by UUID, stopped/deleted, and the original foreign `demo-1-container` confirmed RUNNING. Inner candidate L1 `81b57e94-...` was verified to have **zero inner native children / no original MCP writes**, then explicitly stopped/deleted by exact owned UUID; `user-1004` instance list now empty. Test-only `hgr1lab` account, empty Project/default Profile, ACL and approved image remain **retained only in the disposable outer Box** for safe continuation; the existing historical `boxweave` account, hooks and demo guest unchanged.
- In `bwracc-1106-v2`, failed-start owned native guest `f35945a1-...` was read back STOPPED then deleted; no instances remain. Candidate v1 service disabled/stopped, exact binary left installed for reproducible **disposable lab only**, not a physical deployment or stable release.
- Raw non-secret L1 proof retained on `boxweave-test-ser2`: `/tmp/hgr1-r1-inner-doctor.json` SHA-256 `48d70051405308304f983d6b7c4b7b29f092b6d81004a8eb7738cdf8bb843cda`; `/tmp/hgr1-r1-inner-install-activate.json` SHA-256 `205bd7ffa535512c4ad2de5fc90c20fd04fec70eec9430496bbafa6d1d450960`; `/tmp/hgr1-r1-inner-prereq.json` SHA-256 `efa5f5772eba8767d2f9f43c4bf8584908fe2356058a42857457188a69b84c56`. `/tmp/hgr1-r1-*` and `/tmp/hostgo-r1-*` local logs are disposable Box-only evidence; no secrets committed.
- **Safe prerequisite for Round 1 PASS:** get a genuine IPv6-public-capable, isolated native provider and an owner-approved authenticated official SDK test credential path that obeys tool safety checks; then re-run same binary I+N+V with 2 owned siblings/layer, direct public IPv4 and IPv6, foreign/peer/spoof/UUID/roles, and a separate independent verifier. No UNKNOWN redrive or parent security relaxation.
- For Round 2, only unrelated safe work may continue after a **separate user invocation**; no carry-over or implied ROUND1_PASS. **next_ordinal_requires_new_user_turn=true**. Final Actions must run **after these Box tests** for the updated report commit, and any new docs head must not be misrepresented as the earlier embedded candidate.

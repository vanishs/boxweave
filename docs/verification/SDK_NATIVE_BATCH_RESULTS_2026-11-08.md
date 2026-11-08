# Incus 7.6 nested SDK minimal batch — real execution and cleanup evidence

**Date:** 2026-11-08  
**Repository branch:** `v1`  
**Result:** **PARTIAL ACCEPTANCE, NOT v1 P0 PASS**. Narrow native SDK/lifecycle/file/network tests succeeded; **a runtime NIC-ACL update permission gap is CONFIRMED**; several adversarial and L0 tests remain unexecuted. This is a disposable **nested L1/L2** experiment, not a running v1 BoxWeave controller.

**Supersedes the initial blocker assumptions** in [minimal batch plan](SDK_MINIMAL_BATCH_2026-11-08.md) and [previous lab probe](SDK_NATIVE_BOX_PROBE_2026-11-07.md). Do not overwrite the original failed-create logs; the correction is an additional verified state transition.

## Scope and instrumentation

- **Outer L0**: `dev`, BoxWeave `main` / source guidance 0.50.4, Incus daemon **7.5.1**. No L0 Incus upgrade/restart, no new L0 Box created. A separate older L0 create intent `bwv1sdk-1107-a` is **still NOT reconciled** here.
- **Existing disposable L1**: `bw-debug-0460-lab` (outer UUID `a51196d3-54f9-4d67-ae6f-7b22a1aca7e9`), Ubuntu 24.04, native **Incus server/client 7.6**, own directory storage pool and managed bridge. All actions below occur *inside this Box only*.
- **Standalone compiled dependencies**: Go **1.26.10**, official Incus Go module **`github.com/lxc/incus/v7@v7.6.0`**, official MCP Go SDK **`github.com/modelcontextprotocol/go-sdk@v1.8.0`**. Go binaries and source were compiled in the test Box, not committed to `v1`.
- **User/project isolation**: disposable non-root `bwv1sdk1108` (UID 1003), member of `incus`, **not** `incus-admin`, auto-confined Incus Project **`user-1003`**. Test child `bwv1sdk-t1-1108` UUID `071584fa-eca2-41d7-bd30-06e2ba1d9e28`; test child `bwv1sdk-t2-1108` UUID `b066e9df-9588-49ff-9bf2-f61cce599fed`.
- **Network addresses observed from native daemon:** A IPv4 `10.121.243.127`, IPv6 `fd42:cd60:de16:c35c:1266:6aff:febb:66ff`; B IPv4 `10.121.243.153`, IPv6 `fd42:cd60:de16:c35c:1266:6aff:febe:a662`. These are **ephemeral test values, not product defaults**.
- Test logs/scripts remain under **`/home/bw-debug-0460-lab/bwv1-native-probe-1108/`** within L1 (not exposed as MCP host-log endpoints). Independent native `incus list` / device / volume readbacks supplement the actual command exit codes.

## First: recover original Cowsql failure, prove its true state

The original single `incus init` returned **exit 1** with `Lost connection to the event listener ... unix.socket.user: i/o timeout`. Native logs then reported **`no available cowsql leader server found`**, including failure to delete original operation `d93e53d0-59fd-41c7-a527-aa95e1f697ef`. A partial instance-storage path existed. Native `incus project list` repeatedly timed out despite `incus.service` being `active`; available memory and disk space were ample after the event. **Causal attribution (nested kernel, daemon, event listener, storage, main gateway) is not established.**

One controlled **inner Incus daemon only** recovery was executed in L1:
1. Preserve last daemon journal and actual storage path inventory. Stop **inner** `incus.socket` and `incus.service` once.
2. Make an **offline database backup** of `/var/lib/incus/database` inside L1. Path: `.../recovery-1108/incus-db-offline-20261108T085000Z.tar.gz`. SHA-256: `b93a8995037be90d504d1fa2b4c80e4450b2b0aa004df11df42712aadc590a2a`. Archive remains **root-readable only**.
3. Start **inner** Incus once, with no manual Cowsql data edits/recovery SQL and no L0 restart. The stored `recovery.log` recorded `STOP_RC=0`, `BACKUP_RC=0`, `START_RC=0`, `PROJECT_READ_RC=0`, `INSTANCES_READ_RC=0`.
4. Native readback discovered the **exact original instance** `bwv1sdk-t1-1108` already existed and was **STOPPED**, UUID `071584fa-eca2-41d7-bd30-06e2ba1d9e28`, with one native storage volume. **No repeat create was dispatched.** It was started **once**, yielding `START_EXIT=0` and native `RUNNING`. This proves a failed caller receipt is not proof of zero native effects.

**Outcome: PASS database read recovery and original identity reconciliation.** **Not** a proof that the underlying stall is fixed permanently. A second child creation later completed without a repeat of that stall; this is limited evidence only.

## B: native SDK, lifecycle, Exec and guest files

| Test / original evidence | Actual result | Status |
|---|---|---|
| Official MCP SDK, typed Tool auth (earlier batch A; not rerun) | `go test -run TestMCPAuthentication` exited **0**; separate admin/viewer actors read through `req.Extra.TokenInfo`; invalid, expired and subsequently revoked Bearers rejected | **PASS — standalone SDK only** |
| Official Incus SDK restricted credentials | `ConnectIncusUnix(...unix.socket.user)` sees only `user-1003` but unscoped calls still target `default`; `UseProject("user-1003")` required. Admin socket denied at OS layer; foreign Project config denied; `api.StatusErrorCheck(err,404)` true on missing owned instance | **PASS — scoped SDK/OS tests** |
| Native child A actual create/start | Recovered original unique UUID STOPPED; one start `START_EXIT=0`, native RUNNING, 1 CPU, 512 MiB | **PASS after reconciliation; initial create response was FAIL/ambiguous** |
| Native child B creation | Exactly one `incus init` from locally cached Ubuntu image and one start; original `child2.status`: `CHILD2_INIT_EXIT=0`, `CHILD2_NATIVE_GET_EXIT=0`, `CHILD2_START_EXIT=0`, `CHILD2_FINAL_READ_EXIT=0`. UUID `b066e9df-9588-49ff-9bf2-f61cce599fed` | **PASS — nested native lifecycle** |
| Official SDK nonzero Exec | `native-batch-b`: Operation **`3809686b-b949-4992-adec-1bd43bc9bedf`**, submit **481 ms**, stdout `probe-out`, stderr `probe-err`, native return **7**, output completion observed | **PASS**; native `Operation.Status=Success` must **not** be conflated with guest exit 0 |
| Official SDK delayed Exec | Operation **`e2e5b071-cb36-4ff3-8e6d-0acf31b2471e`**, submit **96 ms** for an approximately 2-second command, native exit **0**, captured `delayed-output` | **PASS — early operation ID and final output**; not a post-worker-crash recovery test |
| Official SDK guest file | `CreateInstanceFile` on owned scratch file, `GetInstanceFile` exact bytes and regular-file type, `DeleteInstanceFile`, then native 404 readback | **PASS — ordinary file create/read/delete** |
| Official Incus SFTP `O_CREATE|O_EXCL` candidate | Attempt to exclusively create an **already existing** owned file was denied (`file exists`), and a second read showed **unchanged original bytes** | **PASS this narrow negative**. **NOT** CAS under concurrent writers, symlink/hardlink race or conditional replace/delete |
| B standalone test process | `batch-b.status`: **`B_EXIT=0`**; complete original output in `batch-b.log` | **PASS batch process only** |

**Remaining B-native UNKNOWN:** concurrent exclusive-create races, symlink/hardlink substitution, atomic hash-conditioned replace/delete (ordinary native file API lacks CAS), truncated file read hash coverage, Exec worker crash/cancellation, long-poll concurrency, guest UID/GID capability and ETag update concurrency. These are **not** implied by ordinary SDK functionality.

## C: real Bridge NIC ACL and networking

The native user Project inherits a **managed Bridge `incusbr-1003`** with both IPv4 and IPv6 enabled. No OVN or host-global firewall editing was used.

| Step | Independent native/packet observation | Status |
|---|---|---|
| **Baseline, no ACL** | A→B IPv4 ICMP succeeded; B→A IPv4 ICMP succeeded; A→B IPv6 ICMP succeeded. Default Bridge **does not isolate peers by itself**. | **PASS baseline observation; security policy not yet in force** |
| **Attach empty NIC ACL** | Host-local native admin created dedicated test ACL `bwv1sdk-acl1108` in `default`. **Restricted** user successfully attached it as A's NIC override. With no rules/default reject, A→B and B→A **both failed** over IPv4 and IPv6. | **PASS — native Bridge NIC intra-bridge deny** |
| **Allow A→B, deny B→A** | Admin added **only** A's egress `allow destination=10.121.243.153` and matching IPv6 destination. IPv4 and IPv6 **A→B ICMP succeeded**, new **B→A ICMP failed**, indicating stateful replies work in this tested topology. | **PASS — native directional packet test; rules changed by admin, not runtime user** |
| **TCP/UDP packets** | Python client in A printed `PASS V4 TCP`, `PASS V4 UDP`, `PASS V6 TCP`, `PASS V6 UDP` with exact echoed `ACK-PROBE`; B server independently logged success on all four sockets. | **PASS packet exchange, PARTIAL CLI execution tracking:** exterior `incus exec` client process became stopped, and after safe `SIGCONT` it printed replies but finished `CLIENT_EXIT=124` with native operation read `EOF`. This does **not** prove durable SDK operation receipt. |
| **Revoke A→B** | Admin removed both exact egress allow rules. Native ACL readback again showed empty rules; new A→B IPv4 and IPv6 ICMP each failed. | **PASS — new-flow revoke**; long-lived established TCP flow not tested |
| **Runtime user creates/updates ACL** | `sudo -u bwv1sdk1108 incus network acl create ...` and `... rule add ...` each **denied** with `User does not have permission for project "default"`. Same user **can attach** the admin-created ACL to its own NIC. | **FAIL for v1's dynamic `box_peer_set` under default `incus`-only Project policy** |
| **Try separate fresh confined Project with `features.networks=true`** | Created second guest-local `incus`-only user `bwv1acl1108` and empty Project `user-1004`. Before any instances, host-local admin `incus project set user-1004 features.networks=true` failed: **`OVN is required for projects with "features.networks" enabled`**. Under `features.networks=false`, restricted ACL create/read still denied. | **BLOCKED in this Bridge/non-OVN setup**; not proof every different Incus topology is incapable |

### Proxy devices: distinguish native denial from installer-controlled rights

A restricted `incus` user initially received **`Proxy devices are forbidden`** when adding instance proxy. Host-local admin temporarily set **`restricted.devices.proxy=allow`** on **only** the disposable `user-1003` Project. The same restricted user then successfully added/updated its own TCP and UDP `proxy` devices:

- **TCP**, host-local `127.0.0.1:18764` → guest B `10.121.243.153:18762`, tested actual HTTP **200** from the L1 host. A preliminary `connect=tcp:127.0.0.1:18762` configuration had returned an **empty reply**; correcting the guest service to bind `0.0.0.0` and pointing proxy to B's observed guest IP gave the successful readback. This is a topology-specific test, **not** evidence that arbitrary proxy target IPs or public binds should be exposed in BoxWeave.
- **UDP**, host-local `127.0.0.1:18765` → guest B `10.121.243.153:18763`, actual datagram acknowledgment **`bwv1-ack:probe-1108`**, matching expected bytes.
- **Network clean PASS:** original `net-clean.status` = `NETWORK_CLEAN_EXIT=0`; TCP/UDP proxy devices removed by the restricted actor, guest temporary service stopped, A's temporary NIC override removed, ACL independently confirmed `used_by: []` and deleted, original `restricted.devices.proxy` setting **unset/restored**; both host loopback listeners absent on native readback.

**Conclusion:** P0 native host loopback TCP+UDP forward is feasible **with an explicit and deliberately bounded Project proxy grant**. **External LAN/public bind, guest-IP reassignment, restart persistence and L0↔L1 chained forwarding are UNTESTED**. Narrow BoxWeave authorization must still refuse foreign devices, arbitrary target IPs and implicit public bind.

## Exact cleanup: confirmed versus unknown

**Confirmed:**
- The two L2 children were each checked against their original native **UUID**, stopped, observed **STOPPED**, deleted in sequence by the restricted actor, and verified absent by **independent admin native inventory**.
- `l2-delete.status`: **`L2_DELETE_EXIT=0`**; `l2-delete.log` logged both `VERIFIED_ABSENT` transitions.
- Native `incus list --all-projects --format csv` returned **no instances**; native `incus storage volume list default --project user-1003 --format csv` returned **no owned container volumes**.
- Temporary ACL and proxy devices removed and per-Project Proxy grant restored, as described above.
- L1 Incus remained responsive following cleanup. No restart or upgrade of L0/main or deletion of the outer L1 Box was performed.

**Last optional cleanup — CHECK ORIGINAL RECEIPT BEFORE ANY FURTHER ACTION:** after the two L2 instances were deleted, a guest-local script `~/bwv1-native-probe-1108/final-owned-clean.sh` to remove **only** disposable OS users `bwv1sdk1108`/`bwv1acl1108`, confined Projects `user-1003`/`user-1004`, and their unused auto-created networks/images was *submitted*. The BoxWeave `main` gateway returned an internal failure before a result could be confirmed, and even follow-up read-only tools temporarily failed. Therefore **execution of that optional script and final user/Project/network absence is UNKNOWN** here. Inspect **`final-owned-clean.status`**, `final-owned-clean.log`, the `bwv1-finalclean1108` tmux session and **read-only** `incus project list`/`incus network list` before running any cleanup again. Do not claim zero residual test Project metadata until observed.

The backed-up database and real test logs are deliberate L1 artifacts, **not** exposed to model clients as host diagnostics or removed blindly.

## Architecture decisions now grounded in evidence

1. **Keep official SDK-first.** Working native SDK Exec/file paths, typed 404, restricted Unix user and actual NIC/proxy devices remove the need for generic wrappers. `UseProject` must explicitly select the verified restricted Project even on `unix.socket.user`; `GetServer().Auth=trusted` is not Incus administrator authorization.
2. **Native proxy default `block` is expected, not a compatibility defect.** Host-local installer can explicitly grant **`restricted.devices.proxy=allow`** for the owned restricted Project (with user-approved threat model and BoxWeave's own strict host-bind/target policy). No generic Caddy/REST/forward daemon needed.
3. **P0 dynamic Bridge ACL grant is a real unresolved privilege boundary.** On a native confined `incus`-only user Project with `features.networks=false`, ACL objects live in `default`; a restricted user may attach those ACLs but cannot create/update their rules. This daemon required OVN to turn on `features.networks=true` for a separate Project. **Do not implement `box_peer_set` by silently running the public MCP server as host root/`incus-admin`**, and do not mislabel admin-executed ACL changes as a restricted-service PASS.
4. **Smallest pending choice, not automatic scope growth:** independently investigate whether an official **restricted TLS credential with targeted ACL permissions** can satisfy this precise default-Project ACL mutation without exposing unrelated resources. If native authorization cannot safely express that, propose one **host-local, narrowly scoped ACL mutation boundary** with immutable owned identity/rule validation and audit; explicitly request owner approval before introducing a privileged runtime helper or weakening P0 network isolation. Generic firewall/ACL editor, OVN migration and arbitrary Incus admin are **not** implied.
5. **Fail-closed operation semantics are necessary even when upstream SDK is correct.** Original `init` failed with an event/DB timeout yet the instance really existed; only readback after controlled native recovery established identity. Never auto-replay an unknown mutation. Distinguish observed guest nonzero exit from native operation `Success` and native packet successes from failed outer CLI response.

## Remaining minimal gates before real v1 acceptance

- **P0 BLOCKED DESIGN:** restricted actor's native Bridge ACL mutation permission for production `box_peer_set` (plus owner decision if native credential route does not work).
- **P0 UNTESTED:** malicious symlink/hardlink and concurrent file create, true conditional CAS replace/delete; nested worker restart/crash + original Exec output retrieval; dynamic guest-IP change and stale address reassignment; anti-spoof with claimed/mismatched IPv4/IPv6; ACL behavior on established flows; native fail-closed during ACL update.
- **P1 UNTESTED:** external real LAN TCP/UDP client, public bind firewall, L0 host and L1-L2 stacked forwarding, multi-depth nesting with the actual v1 controller, and installed release artifact evidence. Do not promote native lab evidence to external-LAN or release PASS.

**Source context:** [tagged SDK method review](../research/SDK_GAP_CLOSURE_2026-11-07.md), [per-tool SDK mapping](../research/SDK_TOOLS_NATIVE_MAPPING_2026-11-07.md), [Bridge NIC ACL limitations](https://github.com/lxc/incus/blob/v7.6.0/doc/howto/network_acls.md), [Incus Project restrictions](https://github.com/lxc/incus/blob/v7.6.0/doc/explanation/projects.md), [native proxy device](https://github.com/lxc/incus/blob/v7.6.0/doc/reference/devices_proxy.md).

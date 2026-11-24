# PR #178 N4 installed fault and traffic acceptance (2026-11-23)

Status: isolated installed acceptance and dependency fix verified; exact-head hosted CI is the next gate. Draft only; no merge/release authorization, no N5 sign-off.

## Baseline and scope

Parent c69792c5a05a677baae6656dfe991f928a55a227: all four PR workflows SUCCESS (45327817595 source, 45327817601 quality, 45327817579 candidate artifacts, 45327817679 installed lifecycle). Scoped job 138221529690 independently emitted all N2/N3 required markers, including native tracked TCP/UDP, installed continuous TCP, replay, lifecycle and boot-policy. No N3 failure required repair.

All N4 real effects took place inside existing disposable nested acceptance Box 8d93d497-95e7-46af-925c-c6e08f08eb5a (pr178-n3-installed-r2-1123), with independently installed candidate and Incus. The earlier inherited unconfined outer profile was not changed; physical Host parity is NOT_TESTED. Only inner loopback Forward binds were used, no exposure or production L0/other Boxes changed. N3 cold boot and completed-ACK-loss recovery already passed; see V1_PR178_N3_INSTALLED_PEER_ACCEPTANCE_2026-11-23.md.

## Native evidence (L0 Operation IDs, guest exit 0)

- 54c6e14cef6ab260a6643647a40a3b69: four newly created n4-a/b/c/d; C->D directional allow; default A/B denial; TCP and UDP forwards on both 127.0.0.1 and ::1 to D, all four CONFIRMED and independently native_verified. During A->B grant, reverse grant, revoke, regrant: IPv4/IPv6 UDP positive/negative probes, C->B/A->C denied, reverse and C->D allowed. Four persistent Forward clients completed 447 rounds without a failed exchange. All four ACL identities and running/autostart state retained.
- 6f92326f241b73768fa7a5c509a92cdc: concurrent exact-key revoke pair returned original 3246cf5020dc6c4a0f2277826e477511; competing D->C grant refused, later separately admitted 0f968472537659cbf56eecdeb883cf00 SUCCEEDED. Original established IPv4/IPv6 UDP sockets blocked revoked direction; reverse and unrelated sockets continued. Fixture uptime witnesses unchanged. N3 already proved TCP sockets.
- 72a15bdb6a10e34291059c4e2fadfd8b: revoke remaining reverse, stop/delete B, recreate B with different UUID, ACL, MAC and both addresses. Former B de8abef8-7967-4800-9ad5-88610d78fc11 -> 02246f1f-4297-40c9-9271-7b69968eb2ee. Exact old revoke replay bca028fc25ffb411c9f82c765cfa061a returned prior receipt; new B isolated. Persistent D loopback Forward TCP sockets retained. No forced address reuse; retained-address mechanism tested, not pool-exhaustion parity.
- df2b9069873136adac371431081b566d: test-only, fixed-pair real native ACL CAS on grant followed by injected lost acknowledgement, after durable intent. Actual source ACL changed, target and unrelated ACLs exact preimage, Guard unchanged. A->B and B->A packet probes denied for IPv4/IPv6; C->D allowed. Four persistent Forward clients 177 successful rounds, zero errors. Original 53254acbb11eeaf598faaca6eadf1778 became UNKNOWN after controller restart; direct recovery refused incomplete proof, root boot restore refused unpublished original. Original journal has only step-00-intent.json; no reapply. Postflight fc865654233913a45d02beeca0805052 verified five retained claims and four CONFIRMED Forward rows.

## Defect and fix

Original postflight aa3d3294166f9d8af9045e505b0af405 returned correct BLOCKED/RETAINED recovery diagnosis but the next MCP call disconnected (guest exit 1). Root helper refuses unpublished Peer journal at startup, restarts, and the controller's systemd Requires dependency caused controller disruption. Keep native guard/Incus and Root prerequisites unchanged; change only scoped controller dependency from Requires to Wants (retain After and scoped flag). The database admission barrier and trusted Root preflight still fence the entire project's mutations. No automatic recovery, native authorization relaxation, or journal removal.

- 9dc549e168ef5ddfbbcb56dbfe930f90: targeted unit generation, retirement and Forward policy tests exit 0.
- a9736e9c73789e71239dd2967dd93738: lab-only exact unit dependency delta, preserved original unit, reloaded systemd and restarted controller; Wants includes scoped lifecycle, Requires does not. This is not an in-place installer upgrade or production mutation.
- 18c2c06b10675c242e850f9673cfaa8b: 12 successive authenticated MCP inspect calls report UNKNOWN/BLOCKED/RETAINED and native_retried=false; 12 new Peer admissions refused; controller MainPID unchanged; original state, all five claims, journal file SHA and count of admitted Peer originals unchanged. Both N4 degradation markers passed. Fault lab intentionally preserved with incomplete journal and helper refusing startup. The ongoing helper refusal is expected; cannot settle partial original without trusted additional evidence.

- f9f4b3972bdfa299c4c8fe65733cb2d7: full Go suite on exact N3 source plus dependency fix, unit regression and CI script changes, exit 0 (state 134.948s); native partial fixture separately compiled and executed above. Shell syntax and git diff checks passed.

## CI handoff

Fresh hosted scoped matrix runs N2 and N3 unchanged, then N4 native traffic/Forward, concurrent UDP, same-name reincarnation, actual partial native ACL fault (opt-in Go test binary; never installed in product), and degraded MCP stability. Fault is deliberately last; CI must show exact UNKNOWN phase and all eleven N4 markers. Legacy matrix remains unchanged. The exact new integrated CI script is pending hosted execution; isolated local results use the documented candidate and tested dependency delta, not a claim of exact new installed binary.

## Remaining boundaries

Physical L0, LAN/public Forward, pool exhaustion and forced actual address reuse not tested; no unsupported claims. N3 mocked stale ETag/guard/conntrack ACK fault regressions supplement the N4 actual first-ACL ACK-loss experiment; not all partial-stage effects are actual native fault injections. Incomplete histories remain fenced and refuse cold boot, with stable diagnostic API after controller dependency fix; exceptional repair requires trusted evidence. N5 final integration and release review remains separate, #177 stays frozen. No merge to v1.

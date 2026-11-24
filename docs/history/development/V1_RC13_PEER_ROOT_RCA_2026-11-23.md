# RC13 Peer Root lifecycle / blast-radius — real incident RCA

**2026-11-23 | Draft investigation, not an implemented fix.** Exact binary/source `v1.0.0-rc.13` / `6c995687ae38b5cd0e23c464dee84f40a33a01ef`. Linked to the already-existing [unified diagnostic PR #177](https://github.com/vanishs/boxweave/pull/177); base `v1`. No new Host mutation, release or merger is authorized.

## Summary

Actual L0 client → L1 server → L2 HTTP Forward **passed** real TCP/IPv4 packet checks and unrelated third-party denial. But Root PRS authorization performed an intentional **shared-ACL full-Project stop/restart** before the ACL change, changing all bridge consumers' `boot.autostart` to false and killing a temporary HTTP listener on an **unrelated** owned Box; its old Forward remained correctly `CONFIRMED/native_verified=true`, because device verification is not application readiness. This is primarily a blast-radius and lifecycle contract issue, **not** a Peer security bypass.

The original Peer had also been stuck in `ADMITTED` pending the absent Host-local one-use helper, holding source, target and ACL resource claims and denying even operation_cancel. A Host operator explicitly staged/approved the exact original and then execution succeeded. Unit remained installed/enabled at close; whether a subsequent grant/revoke would be blocked is a **source-derived risk, not reproduced**. A fresh authenticated L1 MCP successfully created its own L2 Box and Forward.

## 1. Evidence and topology

| Layer | Resource | Observations |
| --- | --- | --- |
| L0 | RC13 + Incus 7.6, Project `user-1001` | `ready=true`; live daemon external forward policy `all`; proxy `allow` |
| L0 client | `rc13test-cli` UUID `bf276e23-35f9-4be2-9fbc-0ceb3de39f7c`, `10.249.1.18` | Before Peer: to L0 server `:18181` TCP timeout, exit 28; after Peer: received `OUTER_SER_OK` |
| L0 server / L1 Host | `rc13test-ser` UUID `ea6951dc-9b3f-416b-8c2b-0c40e8fc746a`, `10.249.1.9` | `security.nesting=true`, independently installed Incus 7.6 and **same** RC13 binary, Project `user-1001`, Native 8/8 / MCP ready |
| L2 | `rc13inner-http` UUID `fa4ea366-f3b8-4827-b43a-6084eb93f345`, `10.165.230.186` | Created through L1 authenticated RC13 MCP, Running/Exec/IPv4; Python HTTP `127.0.0.1:8000` returned `INNER_RC13_HTTP_OK` |
| L1 Forward | Original `d2c590605b199eaa137a4aa80f4065cd` | L1 namespace `0.0.0.0:18080` → L2 loopback `:8000`; `SUCCEEDED` and `CONFIRMED/native_verified=true` |
| Separate L0 test | `http-check-1123` UUID `e09771dc-3346-4b51-9be2-8536da517bc9` | **Different L0 Host** `:18080` Forward `1b219282a60e0cef63951beff6e68758`; temporary Guest Python listener disappeared during PRS |

**Live packet proof:** L0 client → `10.249.1.9:18080` → L1 Forward → L2 HTTP returned `INNER_RC13_HTTP_OK` (L0 Exec `6e5f8df87f9607bf169dfe4fec71e871`, Guest exit 0). Unapproved third Box to same Forward timed out (`23ccdfca3a429de177849cdc34187c20`, Guest exit of shell 0 after confirming inner curl exit 28). Earlier unapproved third Box to L0 server `:18181` also timed out.

**Peer original:** `50b99fa7dc8de5d9f5317a87387cab6d` held `box:source`, `box:target` and `acl:default/bw-user-1001-acl` claims. Before helper: ADMITTED, `systemctl show` reported `LoadState=not-found`; `box_exec` and `operation_cancel` refused `original unresolved resource claim is still locked`; `operation_recover(inspect)` correctly refused `ORIGINAL_NOT_UNKNOWN`. Host owner executed the documented exact `native-peer-root-stage` → immutable `native-peer-root-plan` → `native-peer-root-unit-stage` → local service start → restricted `native-peer-root-execute`. Original transitioned ADMITTED → DISPATCHING → VERIFYING → SUCCEEDED, `PEER_ROOT_SCOPED_GRANT_KERNEL_VERIFIED`, Peer CONFIRMED. Never reissued UNKNOWN or altered SQLite directly.

**Observed availability effect:** After authorization, all three L0 Box rows were Running but `native_options.boot.autostart=false`. On unrelated `http-check-1123`, Guest curl to previous `127.0.0.1:8000` exited 7; its native Forward device still confirmed. Restarting only the Guest HTTP process (`7518027f7d8e9dc84f4109d9c53785b1`, Guest exit 0) restored body `BoxWeave HTTP OK`. The external physical Host packet was not re-checked afterward.

## 2. Exact source RCA (immutable RC13)

| File / function | Observed code contract | Consequence |
| --- | --- | --- |
| `internal/native/peer_acl_isolation.go` — `isolatePeerACL` | Enumerates **all** instances on exact owned Project bridge, validates identity, sets `boot.autostart=false` on **each** one, then stops **each** before allowing global shared-ACL change. | Intentional fail-closed full bridge isolation, even for unrelated Box. |
| `internal/install/native_peer_root_grant_apply.go` | Root-only guarded original journal, stopped bridge check, ETag ACL PUT, exact conntrack purge and nft verification, then controlled resume of previous running consumers. `nativePeerRootResumeGrantVerified` requires resumed consumers still have `boot.autostart=false`. | Cannot safely “fix” by merely restoring autostart or skipping Stop; proofs and uncertain-failure containment depend on this state. |
| `internal/install/native_peer_root_service.go` | Generates one-original/one-SHA bounded root service. `NativePeerRootUnitStage` uses `O_CREATE|O_EXCL` and never silently replaces/starts an existing unit. | After successful original, the lifecycle for safely retiring this unit and approving a **different** original needs review; no second grant was attempted in this run. |
| `internal/control/forward_tools.go` | `port_forward_list` description explicitly says `native_verified` proves current exact native proxy device only, not Guest packet or external LAN. | Proxy state was **not incorrect** despite dead HTTP process. Doctor/operator messages need distinct readiness evidence. |
| `cmd/boxweave/main.go` + `internal/native/status.go` | CLI process environment affects reported policy; running service gets `BOXWEAVE_FORWARD_EXTERNAL_ALLOW=all` from systemd. | Standalone L1 `doctor` reported disabled while authenticated live MCP correctly reported all — already owned by #177. |
| `docs/design/NETWORK.md` | Forwards are per-layer instance proxy devices; Peer ACL is separate. | L1 `0.0.0.0` is not L0 physical Host binding; two `:18080` listeners are distinct namespaces. |

**Causal chain:** shared ACL change → root stops all consumers and disables autostart → transient unsupervised HTTP exits → Boxes resumed Running but process not restarted → proxy device remains valid → HTTP refused while `port_forward_list.native_verified=true` continues to mean **device only**.

## 3. Remaining items, severity and routing

| Priority | Candidate | Verification status | Implementation owner |
| --- | --- | --- | --- |
| P1 | Peer approval should warn and record affected **whole-Project** Box set, stop/restart and autostart changes, Forward dependency impact and volatile workload risk. | **Reproduced** actual process loss; safety barrier intentional. | This PR |
| P1 | Root one-original unit completion, status, retirement/rotation and approval for a next different original without history loss. | Unit left enabled; exclusive write behavior **source confirmed**. Future reuse failure **NOT TESTED**. | This PR, Host-only with independent approval |
| P1 | Define boot.autostart state policy: preserve original/unset, deliberate permanent false, safe re-arming only after COMPLETE, Host reboot/daemon restart and UNKNOWN. | All three false post-PRS; reboot not tested. | This PR, Owner decision before behavior change |
| P2 | Separate proxy-device verification, Guest listener, L1/L0 packet witnesses; make remediation discoverable. | Device confirmed + Guest listener absent observed. | #177 Doctor/Recover, no duplicate backend |
| P2 | ADMITTED Peer waiting on missing helper, locks, cancellation refusal and exact Host approval next step. | Reproduced and resolved by authorized Host workflow. | #177 Doctor/Recover |
| P2 | Standalone CLI versus live daemon policy provenance. | Reproduced in L1. | Already in #177 |
| P2 | Guest Exec Operation SUCCEEDED while Guest exit 124 on apt HTTP timeout; guest source path detection. | Reproduced, HTTPS guest-local fixed. | #177 diagnostic taxonomy / operational guide, not automatic APT |
| P2 | Invalid instance name underscore should return actionable field validation. | Reproduced with safe hyphen retry. | #177 or small UX follow-up |
| P2 | Nested security provenance: L1 unconfined but L2 AppArmor enforce; no full namespace proof. | Both views observed, no holistic security witness. | Existing security acceptance + #177 coverage |

## 4. Safety, product and acceptance boundaries

- Do not remove the full-project security barrier without proof that no live affected NIC retains a stale shared ACL, nft or conntrack state under failures. Do not silently reenable autostart: success/UNKNOWN safety invariants explicitly depend on false.
- Root remains Host-only and one-original SHA-bound. No public MCP root shell, no broad ACL reconfiguration, no guessing after ACK loss, no SQLite state hacks. New Doctor is strictly read-only and the one repair CLI is #177's existing `recover` design.
- Do not confuse ordinary Guest application processes with a managed Box lifecycle. A transient `nohup` listener will not survive a stop/start. Consider suggesting systemd-managed Guest services rather than an unapproved in-Guest agent.
- Preserve evidence and run affected tests in a fresh disposable L1/L2 clone or separate Box; do not modify installed physical L0 or revoke existing test Peer without explicit Owner signoff.
- Not covered: complete IPv6/UDP matrix, physical Host external packet after restoring the old test service, post-reboot persistent autostart, quota witness, all-layer AppArmor namespace independence, long-running load or 1024 real Boxes.

**Related:** [#177 real incident diagnostic matrix](https://github.com/vanishs/boxweave/blob/fix/rc12-rc13-guided-uninstall-review/docs/development/V1_RC13_NESTED_PEER_DOCTOR_CASES_2026-11-23.md). See the [remediation contract](V1_RC13_PEER_ROOT_REMEDIATION_PLAN_2026-11-23.md) and [next-Agent handoff](V1_RC13_PEER_ROOT_AGENT_HANDOFF_2026-11-23.md).

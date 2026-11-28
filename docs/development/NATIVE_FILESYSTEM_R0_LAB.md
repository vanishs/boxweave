# Native Filesystem R0 lab — initial execution record (2026-11-28)

Scope: independently owned new nested lab only. No changes to outer RC15, existing development Boxes, parent daemon, host policies, or release. This is an in-progress feasibility checkpoint, not R0 acceptance.

## Exact identities

- PR #202, branch feat/v1-native-filesystem, base v1 at 82f0d4fdb0be1bd0e0681f39df70f6bcc13a4f3b (merged #201).
- Outer authenticated RC15, commit c0ad27a11fd1644e7e9755900c2c9b5421f84c9f, schema 7, native Incus 7.6, Project user-1001. Host status tracking b4ee801d8a109a79acf6506efaf7e8d6. Host storage quota not independently verified; do not stress physical capacity.
- New Box native-fs-r0-incus77, UUID/incarnation 0784a712-72fa-442e-8a04-188174990203, CPU 4, memory 4096 MiB, disk requested 30 GiB, nesting=true, autostart=false. Original create 4e4b5e2327afd3c6b269ff328fea642c; native ab2f9608-7774-48b8-8e3f-9f0cad235397; SUCCEEDED.
- Preflight Exec dee5b2826d385c574fd5b1a2dc766eac: exit 0; Ubuntu 24.04.5 LTS noble, x86_64, curl available, no Go or Incus initially. Guest-reported /proc/self/attr/current was unconfined; no profile change was requested or made. This is not evidence of enforcing ancestor AppArmor or production nesting safety. No privileged or host policy changes.
- Zabbly stable package source signed with verified documented fingerprint 4EFC590696CB15B87C73A3AD82CC8797C838DCFD. Apt reported candidate 1:7.7-ubuntu24.04-202611270613 for incus, incus-base and incus-client. Initial install Exec bede86d17cc7184f7b1cefb3e0161aaa completed with guest exit 1 after successful prerequisites/repository setup: a local version check omitted Debian epoch 1:. Incus was not installed by that command.
- Corrected exact-version install original fb96d83570f3666a42d29fb4d3fea217, native e65feb74-51d6-418e-8b85-ff50dfd64f85, Tracking 7aa4946c73bb60e17c07ceb68a37048a. Latest observed VERIFYING; output pending. Inspect that original and actual guest exit/output before further work. Do not replay it.
- [Probe source](native-filesystem-r0/probe.go.txt), commit 25970f3e7c196aed3ef773f93f78c123f1408576. Exact source uploaded to /root/native-fs-r0-probe.go using exclusive file apply original 98d63a102eeb080bba778d93761608e8; SUCCEEDED with native readback, no native Operation ID for SFTP.
- Build attempt refused before admission with RESOURCE_BUSY (corrected installation original still active). No build operation, no SDK execution or new native result. Do not describe this as a compiler failure.

## Next bounded research steps

1. Inspect corrected install original, read stdout/stderr and guest exit; verify dpkg exact version and local daemon version. Inspect fresh inner daemon inventory before initializing. Use only fresh inner fixtures and Unix connection; no remote listener. If normal nested initialization or launch fails, preserve precise error and do not weaken confinement.
2. Prepare isolated Go module in /root/native-fs-r0-sdk with exact Incus SDK v7.7.0, pkg/sftp v1.13.13 and Go 1.26.10. Copy retained probe to main.go, use go mod tidy/build, retain go version -m and source/binary digest. Compilation alone is not native acceptance.
3. Initialize fresh inner daemon with minimal dir storage and no public listener; create one never-started inner Linux container named fs-r0-fixture. Verify original native UUID, then run probe with its name, UUID, Stopped and never-started label. If launch is permitted without changes to policy, repeat on Running and stopped-again with separate probe paths. Do not reuse earlier probe output directories or replay uncertain writes.
4. Probe records SFTP advertised extensions, exclusive collision, >1 MiB observed streaming read/write, digest, offset read, attributes, two independent SFTP handles, ordinary/Posix rename outcomes and native state. No proof of cross-Box copying, CAS, snapshot consistency, hostile path races, crash durability, restricted Project or full lifecycle coordination from this program. Follow with distinct bounded native experiments for those claims only as justified.
5. Preserve exact output and actual failures in this record. Record downstream R0 risks and fallback decisions; do not turn unsupported operations into PASS. Keep lab for continuation.

No source dependency bump, product implementation, test acceptance, merge, or release was performed by this checkpoint.

# P1-06d27/d28: pin repair collision policy and schema DDL isolation

Parent 681e97000552b0f00ca9dcd23b970682e951415c.
Owned Box 9d374e18-196a-493d-bf1f-2d582589f04d, /root/pr191-p1-d27.
Resume b5b47b6db2c3d2637a418f49fe6df7a1 exit zero.
D27 separates Legacy archive exclusions from Scoped collision checks,
preserving shared exact binding/fence/owner SQL and original transaction.
Edits b65163d9054bf48fc4e675076c515bc2 and
47794950a4a5ad2a7420eec3432fe39b exit zero.
Verification 03e9d34b6d781da704a9fbba8842e186 exit zero:
seven focused state tests with subtests including new duplicate
terminal claim refusal, full recovery 1.528s/control 17.232s, vet, format.
D28 isolates three legacy table and two index DDL strings without changing
schema version, SQL text, ordering or initializer. Adds missing-retirement
table and changed-retirement index refusal across controller/recovery/root
audit entry points, preserving exact original UNKNOWN receipt and claim.
Edit 8b34cd1ff79ce931a4ecba00644d211d; verification
d694a216ac0aa4aa7782654628604a25 exit zero: reconstructed store.go
byte-identical, seven incompatible-layout/version cases times three entry
points plus six neighboring test cases, vet and format 1392 files.
Staged source tree 33cbeed2de95d5764a9cb912ccd1cb028f6cdc6d.

Earlier BDH-only 56253b5 S2 job 140217004897 success confirms real
Legacy alternate root-pool marker but guest public IPv4 attempts all four
curl28/HTTP000, baseline UNVERIFIED. No environment-only root-cause claim.
Scoped alternate-pool exact job 140222397266 on 738b269 queued at prior
observation; no native PASS yet. 681e970 seven workflows queued/running.
Reuse existing BDH transfer, bounded operation and exact CI observation;
no candidate promotion or new harness work. Preserve legacy tests and
DDL entries until dependent conversion/native evidence. No P1 closure.

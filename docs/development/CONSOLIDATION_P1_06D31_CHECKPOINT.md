# P1-06d31: Scoped lifecycle pin and generic recovery fixtures

Parent d36915287229941801b01d8231f21ebd33e02979 (BDH candidate).
Owned Box 9d374e18-196a-493d-bf1f-2d582589f04d,
 /root/pr191-p1-d31-batch. Prior D31 inventory original
2ba45f4c9259c20607a689f5efacbdd8 consumed on resume.
Extend the existing exact lifecycle pin recovery regression with a
Scoped subtest using schema-2 intent, quarantine reservation, ACK,
per-incarnation ACL and Scoped Create finalization. Preserve the Legacy
subtest and every original missing/wrong/stale claim or fence assertion.
Four generic state/control files switch admission to Scoped without
changing assertions or expected original receipts.
Edit b004f8e1a96d4d02ba6674f5c50ee134 exit zero.
Real use of BDH staged batch: full State and Control, vet, format, exact
parent/tree and private per-check logs in /tmp/pr191-d31-batch-evidence.
The batch and exact remote tree result will be recorded in PR; local
source tests are not native or independent CI evidence. No schema change,
runtime removal or P1 closure. Hosted public IPv4 remains unverified.

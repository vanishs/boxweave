# P1-06d21: shared network audit primitives

Base 7ae85690b2c0cd3f46d8b73e604b0938a947d7ac.
Move the common binding type and canonical name/IP/MAC validators into
network_audit_common.go, separate from the Legacy ledger reader. No
predicate or caller changes. Exact extracted content checked. No runtime
or schema retirement yet; retain all existing acceptance requirements.

Original abe59fef0bed41fc51c1b7312a655e36 exit1: tracked-only git grep
missed new untracked file. Original c83a66916bee2efea161afd14e0187ea exit1:
Legacy reader still needed strings import. Fixed. Original
6848ab401b759a00868401315dc5519d exit0: focused state (9.233s) and control
(0.449s) tests, both package vets, format PASS. Two bounded status reads
(first wait budget expired, no premature output read). During wait,
one exact-head CI list returned all seven runs after first list had
returned only four; none terminal. No skipped proof or efficacy claim.

Correct d16 transposed native/Box UUID from original installed job
140124937862 and exact script argument order oid, native, rid, uuid.
No change to original evidence or result. Alternate pool still blocked;
hosted public WAN unverified. No P1 closure, merge or release.

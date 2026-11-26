# P1-06d24: separate retained address hold readers

Parent 0ee24ab091e802bed688151f9a0c40fbf5b39131.
Owned Box 9d374e18-196a-493d-bf1f-2d582589f04d, /root/pr191-p1-d24.
The Scoped reader and common bounded collector no longer contain the
Legacy archive-exclusion policy. Legacy selects its unchanged exact
SQL and arguments only after validating old history; Scoped retains
the transitional global mixed-history refusal. Both use the same
snapshot and retain original UNKNOWN claims. No schema or runtime
retirement, native effect, admission change, or gate waiver.

One oversized edit admission refused; bounded core edit
e23c69b496a8309cc4af38b262eda3a9 and regression edit
d9aab2aaa92d72e7b3bc7bb87820fa36 exit zero.
Verification 3082607aa1bb39828ebae7d5fce3a9ae exit zero:
six focused state tests including original UNKNOWN holds, three mixed
history subtests, full control (17.268s) and diagnostic (0.008s),
vet and 1387-file format PASS. Original test output consumed.
Staged source tree 384acf0076aadaae73e37036d47d07ede08d62e1
recovered and verified before evidence addition. D25 inspection
4995696e25cad56ff2da1f216a84dda8 exit zero and output consumed.
Prior exact-head CI S1 PASS, remaining six pending/running.
Learning: reuse bounded edits and output/tree verification; no new
BDH asset or performance claim. Independent CI still required.

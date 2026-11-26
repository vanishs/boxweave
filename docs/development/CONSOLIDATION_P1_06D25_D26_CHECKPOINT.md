# P1-06d25/d26: controller status and Legacy lifecycle isolation

Parent dbe6e6030910f65aca8c416737a459ca4d48cbc0.
Owned Box 9d374e18-196a-493d-bf1f-2d582589f04d,
 /root/pr191-p1-d25. Resume 9401347d34c6b0b6b4ce5bcdf8ff7c95 exit zero.
D25 moves the Legacy-only retirement status observer to the existing
retirement module, retaining readiness, missing-helper and advisory
capacity semantics; Scoped capacity remains separate. Edit
2690c4f427ff69f4375a581db123ea64; test e8d8a01b3a237fb689efa316d3991336
exit zero: full control 16.994s, diagnostic 0.008s, vet, format.
D26 moves the complete Legacy Create and Delete finalization bodies into
one Legacy-only file; no changed predicate, dispatch, or native effect.
Edit 05dd1095f1998688f78a407c89d744f1; test
162a8c5204f0efbcdd73524c3250deca exit zero: exact body comparison,
full control 17.029s, recovery 1.700s, vet, format 1389 files.
Staged source tree 391ad4d56ec09e49e854b0406733b32914baa66f.
One read-only inventory exited 1 on a no-match grep; corrected inspection
1a1ec7f4da979b0cc293c28be2884568 exit zero. No product failure.
Altpool N2 run 45912024502: all 28 jobs queued at inspection,
exact altpool job 140222397266 queued. No installed witness claimed.
D25/d26 combined to reduce CI churn; keep all old acceptance gates
until equivalent evidence. Learning: existing bounded tool reads,
original outputs and tree verification reused, no new BDH asset.
Next: remaining Legacy recovery/navigation and startup boundaries,
complete dependent test conversion and direct schema transition only
with the required native evidence. No retirement waiver.

# P1-06d22: shared Create recovery settlement

Base 7fefc61613dce4e586eab551c929484140a2f3c0.
Separate the shared transaction settlement body from Legacy-specific
Create recovery checks. Exact body unchanged; retain both independently
proved recovery paths, original reason/event semantics, and all existing
ownership and UNKNOWN constraints. This is source separation only.

Box 9d374e18-196a-493d-bf1f-2d582589f04d, /root/pr191-p1b-auth-cursor.
Resume c841d8f24a40d1d56cfe4ef5e64869fd synchronized clean head but
inspection exited 141 (head pipeline); corrected read
53c4e4158a60b580743007d85ee9faa7 exit0. One guessed GitHub source path
returned 404; actual caller inventory 970e070aafa8094af44181221faa21a0
exit0. Test 503dfa0a504ddd70ce8f25ddba15fd06 exit0: exact extracted body,
focused state 1.488s, full control 17.532s and recovery 1.773s, vet,
format 1377 PASS. Two bounded status observations, first expired wait,
no premature output access. Verified staged source tree
518c32791a2b20612fddc9d5800cbf5b8f34ae92.

Early checkpoint saved before publication:
https://github.com/vanishs/boxweave/pull/191#issuecomment-7205139055
One PR write, approximate 20 underlying tool attempts since previous
durable commit; quota unknown. Actual bytes retained in Box, not backed
up by the comment. No simulated quota exhaustion or independent recovery
claim. Reuse previous evidence, no new BDH development.

No schema/runtime retirement or P1 closure. Alternate pool publication
remains blocked; hosted public WAN positive unverified. Independent CI
required for exact candidate. No merge/release.

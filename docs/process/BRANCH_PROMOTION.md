# V2 main promotion

Complete: remote backup, v2 publication, GitHub default branch and both local upstreams verified.

User authorization: SRC-092. Destination: existing public vanishs/boxweave GitHub repository. Old implementation d8a09dd180139159e035963502e7793a37e92bae will remain as legacy/pre-v2; v2 history rooted at4d352859f5c7e659d243634196b3b85265a14a9c becomes main. No history merge, force push, old-worktree file change, app deployment, release or public runtime mapping.

The existing MIT license is retained. Old main/install.sh URLs are deliberately retired by user choice. Historical reports keep original branch names, source hashes and tested revisions; current onboarding/install/legacy authority uses the new names. Branch names and directory names are separate.

## Publication review

Reviewed21 commits reachable from669e1f1:513 unique blobs,5,499,969 bytes, two binary screenshot assets. Pattern scan covered private-key headers, common GitHub/OpenAI/AWS credential formats and JWT shapes. Two findings are header-only synthetic rejection strings in tests/test_enrollment.py and tests/test_public_profiles.py; no key body. No other scanned pattern matched. This is a bounded heuristic review, not proof of universal secret detection. Ignored .runtime, private identity stores and untracked legacy archive were not read or published.

Portable knowledge check passed:70 Markdown files,16 authority roles,18REQ/18AC, all eight routes. Diff whitespace check passed. Product code/version/dependencies remain unchanged; the previous168-test/build/seven-browser phase7 evidence is prior evidence, not a fresh runtime test in this docs/license operation.

User completed GitHub main→legacy/pre-v2 rename. Fresh ls-remote verifies only legacy/pre-v2 atd8a09dd180139159e035963502e7793a37e92bae; default currently follows that backup. New main creation and default selection are next; no force push is needed.

## Verified completion

Initial v2 main publication: a3967704a5a26aaab96221adfdd4e20d0fae2505. GitHub repository settings update returned default_branch=main. Fresh remote heads agree with local main and preserved old SHA. Local v2 main tracks origin/main; old worktree is legacy/pre-v2 tracking origin/legacy/pre-v2; origin/HEAD points to origin/main. No force push or history merge. Existing directories and old tracked files are unchanged; only the preexisting untracked containarium-candidate.zip remains there, unread.

This final verification record is committed and normally pushed as a subsequent documentation commit; resolve its exact SHA through git log for this path. No v2 product release, running-service upgrade or application deployment occurred. RISK-604..609 retain their prior scope.

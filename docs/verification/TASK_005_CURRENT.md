# TASK-005 current-source acceptance plan

SRC-066 authorizes TASK-005/006/007 consecutively, with separate evidence/commits. Developer gate SRC-051 remains accepted; TASK-002/003 are completed scoped foundations. Legacy main stays read-only. This task revalidates rewrite0.4.0/e0fc675, not old0.1.1 evidence as if it were new.

Use existing boxweave-v2-test as application test host, bw-v2-mcp-nested independent managed MCP and its v2-live-hello workload. Bundle only current bwmini source, newly built Web assets, VERSION and the repeatable live_slice_check harness from assigned development LXC into a new private app test root. Reuse existing app credential/pin in place, never copy private keys or read JWTs. Fresh local SQLite databases, loopback18641/18642, no public route/service change. Success: actual HTTP and Chromium login/binding/plan/confirm/worker/identity/fixed Python hello/receipt/history/restart without replay, stdout hello world/exit0/live_mcp. Recheck real host-pin and workload-identity failure gates; preserve normal trust, zero hello on mismatch. Stop only owned test processes, retain private evidence. Native enrollment/diagnosis evidence is SRC-065 and human TUI/read is SRC-057, recorded separately from Web results. No release/four-profile inference.

## Actual current-source result — complete minimal TASK-005 (SRC-068)

Current rewrite0.4.0/e0fc675, additive schema3. Source authored/built as non-root assigned dev user; npm TypeScript/Vite build exit0, known harmless XYFlow directive warning. Runtime archive146716 bytes, SHA256 d3c2ca11aeb2622fa91076312da93538b2c0cb28d27b304f80f1174475853da5 verified byte-identical on app host before extraction to /home/boxweave-v2-test/boxweave-v2-task005-e0fc675/source. Only runtime source/dist/test harness/version bundled, no credentials/state. App test uses existing authorized SSH identity in place and a copied public pin, not exported keys/JWTs.

Repeatable harness commands from that source root (private config paths contain only reviewed metadata/identity paths and remain ignored):

```bash
PYTHONPATH=. /home/boxweave-v2-test/boxweave-v2-app/.venv-lab/bin/python scripts/live_slice_check.py --config ../http.json
PYTHONPATH=. /home/boxweave-v2-test/boxweave-v2-app/.venv-lab/bin/python scripts/live_slice_check.py --config ../browser.json --browser
PYTHONPATH=. /home/boxweave-v2-test/boxweave-v2-app/.venv-lab/bin/python scripts/live_failure_check.py --config ../http.json --pin-file ../known_hosts
```

Both success harnesses actual exit0: HTTP operation8d99830d8403470ea696f0c7d78c5d5a; Chromium operationd3c53726c10c4cbd8e821c3b1e0e595a. Real expected identity v2-live-hello/v2-live-hello-container, stdout hello world plus newline, exit0,evidence live_mcp; each audit count1 across API/worker restart, stored receipt/history rediscovered without replay. Chromium mobile has no overflow/browser errors. Dist is newly built from current source. Existing published fixture demo/services remain unchanged.

Real trust/identity gates exit0: deliberately wrong task-only public pin gives SSH255, protocol not started, normal pin unchanged; wrong expected workload hostname gives IDENTITY_MISMATCH with one whoami/hostname call and0 hello calls. Initial harness run failed because Path.absolute retained ../, so the replacement SSH argument did not match; fixed harness to resolve its public pin path and reran successfully. This was a test bug, not evidence of a trust bypass. Wrong pin is never installed as normal trust.

Runtime identities remain app created1793176101, backend bw-v2-mcp-nested created1793212095, inner hello created1793213048. Native Windows Codex0.165.0-alpha.1.2 actual fresh public-key enrollment/registration/independent MCP diagnosis is SRC-065; separate user TUI connected(2 tools)/read is SRC-057. Supporting upstream MCP0.114.2/protocol2024-11-05. Explicit developer acceptance is SRC-051 and remains valid. Prior64 Linux/native negative cases are recorded in TASK_003_COMPLETION; this harness-only iteration does not falsely call them fresh reruns.

TASK-005 minimal selected-profile chain is complete; other profiles remain TASK-007 and all-feature release TASK-013. No blanket installation/confinement/autostart or four-profile claim. Script source hashes: live_slice_check.py=5b10566c812993968b354711e2e2cde79a2a5ea7bbfd678e188ec76251fd6f49; live_failure_check.py=d6f8a298f89b667e1c9d7f5af5d8e071ed2440d5c92afc4764ce100bc2f94ba3. Cleanup is task-owned loopback process termination only; private state retained. No new public port or push. Continue TASK-006 in the same explicit multi-task authorization, respecting the separately approved dedicated lifecycle target/cleanup.

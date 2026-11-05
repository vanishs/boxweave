# v0.44.10 selftest private-plan handoff and diagnostic correction

**2026-11-05 UTC.** Scope: `boxweave selftest plan/run` installed host CLI -> unprivileged application, **not** the owner host's R-02 storage or multi-Agent R-04 coordination. Baseline immutable source: v0.44.9 commit `0ec0dc81d8cdc69d4fb9a8abaae0615cf41e4f24`. New executable version v0.44.10; exact release CI and independent installed acceptance must be recorded after publication, not inferred from source tests. Default stable `PUBLISHED_RELEASE=0.43.1` is unchanged.

## Original failure reproduced in v0.44.9 installed isolated acceptance

The host CLI ran `runuser -u boxweave` and passed `--plan /home/<operator>/private-plan.json` unchanged. The app user could not traverse the operator's private home directory, so a real confirmed plan failed `SELFTEST_PLAN_INVALID` before dispatch. Moving/chowning it to `/home/boxweave` worked, but added an unnecessary manual privileged copy. No Provider mutation occurred on the refused attempt. Source/evidence: [v0.44.9 independent upgrade and live nested test](REACCEPTANCE_0449_2026-11-05.md).

## Patch and security boundaries

1. The **root-owned host CLI**, after the usual public install metadata and explicit sudo gate, reads exactly one file requested by the caller: only a normal regular file with one link; it refuses a final-component symlink and special device/FIFO/directory. The file must be owned by the original sudo caller or the installed app user, and is limited to **8192 bytes**. It uses `O_NOFOLLOW`, `O_NONBLOCK`, `O_CLOEXEC`, `fstat` and a single bounded read; no user-controlled subprocess shell is involved.
2. The host sends those bytes through a **bounded stdin pipe** to its existing `runuser -u boxweave` child, substituting the internal `--plan -`. Plan contents and operator path are never placed in application argv, logs, stderr, reports or shared staging directories. The existing host `run_bounded` 0/2 bounded-output and timeout behavior remains in force.
3. The app user reads only the bounded stdin buffer (or a directly supplied file for explicit app CLI calls), accepts strict UTF-8 JSON and the existing **exact plan shape and confirmation hash**, and writes durable selftest step intents before any Provider create/stop/delete. A wrong confirmation still refuses; no silent retry or guessed original task effect.
4. The host produces distinct pre-dispatch typed codes `SELFTEST_PLAN_NOT_FOUND`, `SELFTEST_PLAN_UNREADABLE`, `SELFTEST_PLAN_UNSAFE`, `SELFTEST_PLAN_TOO_LARGE` with bounded remedies. The app additionally reports `SELFTEST_PLAN_INVALID` for malformed/invalid JSON and `SELFTEST_CONFIRMATION_MISMATCH` for a wrong checksum. Refusal does not invoke the Provider. No broad account chmod, credential disclosure, auto-copy or elevated application code execution.

## Operator example — normal private home works without chmod

Save the envelope and review its one-use plan and confirmation:

```bash
umask 077
sudo boxweave selftest plan > "$HOME/bw-selftest-response.json"
python3 - <<'PY'
import json, os
from pathlib import Path
home = Path.home()
response = json.loads((home / "bw-selftest-response.json").read_text())
path = home / "bw-selftest-plan.json"
fd = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o600)
with os.fdopen(fd, "w") as handle:
    json.dump(response["plan"], handle, sort_keys=True)
    handle.write("\n")
print("Review plan:", path)
print("Confirmation:", response["confirmation"])
PY
# After reviewing the plan and its hash:
sudo boxweave selftest run \
  --plan "$HOME/bw-selftest-plan.json" --confirmation YOUR_REVIEWED_64_HEX_DIGEST
```

The operator home and plan file can remain **0700 / 0600**, respectively. This command **mutates a disposable nested Box** only when explicitly run, and follows the existing result/cleanup/UNKNOWN evidence rules. Do not repeat a crashed run until inspecting its original receipt and exact Box identity. For cleanup, remove the two temporary operator plan files after the run is fully reconciled.

## Acceptance matrix (scope must remain precise)

| Check | Grade | Evidence / pending gate |
| --- | --- | --- |
| Reproducer on prior installed v0.44.9 | historical isolated L | Operator private plan refused before Provider dispatch; app-private 0600 staging worked. |
| Host private-plan ownership/path/type/size refusal, no reader leak | T PASS (candidate) | New tests for regular 0600 private home, symlink, FIFO, directory, missing, wrong owner, oversized file, malformed flags and no subprocess on refusal. |
| App stdin parse, invalid JSON and size gate | T PASS (candidate) | Valid plan crosses argparse and reaches live-only provider guard; invalid bytes and oversize refuse before dispatch. |
| Host typed failed-step, hash mismatch, and no echo of secrets | T PASS (candidate) | Host response safety regression checks and existing selftest journal/cleanup tests. |
| Exact source full Python/build/seven browser fixture lane | **T PASS** | Exact v0.44.10 worktree validation exit 0: **748/748 Python tests**, knowledge, locked npm build, and seven Chromium fixture workflows; `.runtime/validation/c982d92f014d4fa4a70b6bfb90338bd7/result.json`. Fixture PASS is not installed Web acceptance. |
| Official v0.44.10 release, CI and isolated upgrade/install | PENDING | Verify immutable tag SHA and 25 assets (or exact manifest count), then independently install official version. |
| Actual installed `sudo boxweave selftest run` from 0700 caller home | PENDING | On disposable host, assert no chmod/copy, exact `create→hello→stop→delete` and independently absent original child; test typed missing/symlink and confirmation mismatch as pre-dispatch. |

**R-02 outer-host cold-reboot receipt persistence: DEFERRED by owner**. No authority to reboot, restart the owner's `dev` host, clear R-02 state or enable global R-04 as part of this work. Historical R-04/UNKNOWN claims, owner-host services, existing test-created crash evidence and provider configuration remain unchanged.

## Sources

- `bwmini/host_cli.py`: bounded host-reader and app-user subprocess
- `bwmini/__main__.py`: app-user stdin/file plan parsing and typed errors
- `tests/test_selftest.py`: exact pre-dispatch and full-test regression
- [Feature matrix](FEATURE_MATRIX.md), [test evidence layer](EVIDENCE_LAYERS_0449.md), [current risks](OPEN_RISKS.md)

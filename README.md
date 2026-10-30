# BoxWeave v2

**Your agents. Your machines. Connected — with explicit trust and review boundaries.**

BoxWeave is a personal control plane for a **single trusted operator** working with coding agents and explicitly selected remote development environments. Its independent Web application supports project/environment binding, confirmed environment operations, durable receipts, native Tenant and TCP/UDP access workflows, a non-executing Prompt Graph, and reviewed client/model setup guidance. It connects to an **existing, separately administered** Containarium MCP service; it does not replace Containarium or Incus.

**Current source:** `main`, version **0.9.1** (`VERSION`). **Phase 7 maintenance is complete within its documented scope.** This is a source/release evidence statement, not a claim that every existing installation has been upgraded or that production deployment has been certified. Current gates are in [PHASE_STATUS](docs/process/PHASE_STATUS.md); actual phase-7 evidence is in [PHASE_07_REVIEW](docs/verification/PHASE_07_REVIEW.md); remaining limitations are in [OPEN_RISKS](docs/verification/OPEN_RISKS.md).

## What is supported

| Area | Current v2 scope | Important limit |
| --- | --- | --- |
| Coding clients | **Windows-hosted Codex and ZCode**, each using Windows-native SSH or an **explicit WSL SSH** distribution (four profiles) | This is not WSL-hosted Agent CLI support. Agent registration, GUI loading and model access are separate checks. |
| Server application | Non-root **Ubuntu 24.04**, CPython **3.12**, Linux **x86_64** locked package; React/TypeScript Web UI and separate worker | The locked wheels and recorded runs do not establish other Linux, Windows or macOS server support. |
| Provider | Reviewed Containarium MCP launcher, exact project/backend/resource identity and authenticated capability checks | Provider/Incus installation, restricted SSH launcher provisioning, JWT rotation, container host-key reconciliation and Provider upgrades are **external prerequisites**. |
| Console | Project bind/edit/unbind/rebind; resource observation and confirmed create/start/stop/delete; receipts; Tenant shared creation/ingress; TCP/UDP passthrough; Prompt Graph; ZCode GLM handoff; public client setup downloads | Actions depend on explicit configuration, permissions and actual Provider capabilities. Graph relations do not execute operations. Public profile generation does not enroll clients. |

**Not claimed by this version:** macOS enrollment; Cursor or Claude Code integration; WSL-hosted Agent execution; old all-in-one host bootstrap; Web HTTP domain-proxy editor; batch resize; automatic public TLS or source filtering for passthrough; production-ready public exposure; verified boot/confinement; model inference or SLA. Some are deliberate [scope decisions](docs/product/SCOPE.md), not automatically missing code. See the [cross-version source audit](docs/verification/ARCHIVE_051_TO_MAIN_AUDIT.md) for the explicit old-to-new dispositions.

## Choose the correct installation path

**Important:** The old `install.sh` / `boxweave.sh` workflow belongs to the historical [`archive/0.5.1`](https://github.com/vanishs/boxweave/tree/archive/0.5.1) implementation, **not `main`**. In particular, do **not** run an old `raw.githubusercontent.com/.../main/install.sh` command or use the archived host installer as a v2 updater. The current application installer installs **only the independent v2 app**, not Containarium, Incus, SSH forced commands, JWT/key timers or a TLS reverse proxy.

### A. Local fixture for exploration (no live Provider)

Use a prepared non-root Linux environment with Python 3.12, Node/npm, and the locked dependencies. These commands do not provision infrastructure:

```sh
python3 -m venv .venv
.venv/bin/pip install --require-hashes -r requirements-dev.lock
(cd web && npm ci --ignore-scripts --no-audit --no-fund && npm run build)
.venv/bin/python -m bwmini init-admin --config examples/fixture.json
```

Run the following in two separate terminals **from the repository root**:

```sh
.venv/bin/python -m bwmini worker --config examples/fixture.json
```

```sh
.venv/bin/python -m bwmini serve --config examples/fixture.json
```

Open **http://127.0.0.1:18640** in that same environment, log in using the locally created administrator, bind the fixture project, preview/confirm the fixed hello operation and inspect its receipt. The password is entered interactively. The fixture uses real HTTP/SQLite but **synthetic infrastructure**; HTTP and non-Secure cookies are for loopback review only. If port 18640 is in use, do not stop an unrelated service. The example's `.runtime` state is private local test data, not a live deployment.

### B. Install the v2 application with an existing authorized MCP Provider

Follow [INSTALL](docs/INSTALL.md) for the complete, reviewed **build → archive hash verification → non-root install → local administrator initialization → health/repair** procedure. This is not a one-command installation from a clean Ubuntu/Containarium host.

Prerequisites include an independently authorized MCP launcher and identity, exact permitted Provider/backend/project/targets, private external configuration and state paths, prepared Python 3.12/Linux x86_64 dependencies, a usable non-root systemd user manager, and a separately reviewed HTTPS front end if remote Web access is intended. [`examples/live.json`](examples/live.json) is **non-operational** placeholder data. The app listener must remain loopback-only; no public route is created by the installer.

The package entry points are `python3 bwmini/distribution.py build`, then a **separately trusted** `distribution.py install --archive ... --sha256 ... --prefix ... --config ... --activate`; see the exact parameters and independent digest requirement in [INSTALL](docs/INSTALL.md). A successful HTTP response is not an authenticated MCP check. Installed binaries, database schema and recorded evidence must be checked against the actual running release. Existing private test deployment(s) may still run an older version; see [RISK-605](docs/verification/OPEN_RISKS.md).

For a configured deployment, the application CLI exposes `python -m bwmini diagnose --config /private/reviewed.json` for a **read-only authenticated** MCP discovery/read check. It requires a genuinely working reviewed configuration; it is not an installer, host doctor or automatic repair tool. The full operator handoff is in [FINAL_HANDOFF](docs/process/FINAL_HANDOFF.md).

## Client onboarding and model access

After logging into the Web UI, **Client setup** generates a public descriptor and a local hash-gated registration script for one of the four selected Windows profiles. It does not connect to the Provider, enroll public keys or read client private keys. Independently review the downloaded source and expected digests, provide an **existing authorized identity and host-key pin**, and perform the local enrollment under the selected agent/runtime; see [clients/README](clients/README.md). A generated/downloaded script is **not** proof that Codex/ZCode has loaded the MCP server.

**Model wizard** records versioned ZCode Z.ai/GLM settings choices and produces native setup guidance. Credentials stay client-side. Completion of an MCP registration is not proof of ZCode model access or inference. See [REQ/AC-015](docs/product/ACCEPTANCE.md) and [model evidence](docs/verification/TASK_008_MODEL_WIZARD.md).

## Verify and understand the evidence

On the **assigned prepared non-root Linux project environment only**, run:

```sh
.venv/bin/python scripts/check_knowledge.py
.venv/bin/python scripts/validate.py
# Optional: seven real-browser / synthetic-infrastructure fixture workflows
.venv/bin/python scripts/validate.py --browser
```

The validation script does **not** install dependencies, run live Provider drivers or deploy services. Its browser lane requires existing Chromium/system libraries. Do not interpret historical recorded test results as a fresh run, or fixture success as real network, TLS, Agent UI or isolation acceptance. Consult [TEST_STRATEGY](docs/verification/TEST_STRATEGY.md), [FEATURE_MATRIX](docs/verification/FEATURE_MATRIX.md) and [OPEN_RISKS](docs/verification/OPEN_RISKS.md).

## Documentation and historical version

- [Documentation index](docs/INDEX.md) — current authoritative product, architecture, status, installation, test and risk references.
- [Archive 0.5.1 → main source audit](docs/verification/ARCHIVE_051_TO_MAIN_AUDIT.md) — code-based function mapping and known product/verification gaps.
- [R-00/R-01 validation handoff](docs/verification/R00_R01_TEST_HANDOFF.md) — checks that require an authorized repository checkout or prepared Linux environment.
- [Current status](docs/process/PHASE_STATUS.md) and [final handoff](docs/process/FINAL_HANDOFF.md) — the next maintainer's starting points.
- [Archive 0.5.1](https://github.com/vanishs/boxweave/tree/archive/0.5.1) — historical implementation and evidence only. Older setup instructions are **not** current v2 instructions. The `legacy/pre-v2` historical ref is also documented in [LEGACY_SYNC](docs/knowledge/LEGACY_SYNC.md).

The old `README` mixed obsolete phase-specific progress statements from versions 0.2.x–0.9.x with current behavior. Their scoped evidence remains in the [versioned verification reports](docs/verification/FEATURE_MATRIX_HISTORY.md) and [handoff history](docs/process/HANDOFF_HISTORY.md); it must not be used as a substitute for a current release or deployment check.

License: [MIT](LICENSE).
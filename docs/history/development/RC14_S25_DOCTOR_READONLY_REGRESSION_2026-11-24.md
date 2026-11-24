# RC14 release S2.5 staged Doctor regression (2026-11-24)

[Current status](../../STATUS.md) · [Installation](../../getting-started/MANUAL_INSTALL.md) · [Doctor/Recover](../../process/DOCTOR_RECOVER.md).

Original release run [45491519196](https://github.com/vanishs/boxweave/actions/runs/45491519196), source 06d2743, nested job 138790818972 failed immediately after native preparation (eight checks passed) and install-stage with preflight incomplete. Native S2 and both architecture builds passed; publication was skipped. The outer and inner lab scripts invoked Doctor before token creation or activation. #177 made Doctor observation-only; install-stage does not initialize SQLite.

Developer Box operation 2ec1daf3b435eaf6da498928d5e6d812, guest exit 0: exact v1 cbfc6d9 binary, disposable nonroot database. Doctor returned DIAGNOSTIC_DB_MISSING without creating SQLite; explicit token-create initialized schema 6 and Doctor observed it. Native readiness was not exercised in this offline reproduction.

Both layers now initialize with token-create, activate, and retain Doctor ready/native-ready assertions and actual HTTP checks. Explicit package prerequisites include iproute2, nftables and conntrack. Failed Doctor emits diagnostic reason codes, not credential material. Operation 8276b9d6046a144047e1e58e525ff32f, guest exit 0: 17 release tests and TestPR177D1CLINoSourceCreation passed; both scripts pass sh -n. No production Host changes or native-gate relaxation. Actual nested acceptance and publication require the release CI.

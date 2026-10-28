# BoxWeave contributor and execution instructions

## Project scope

BoxWeave integrates an infrastructure provider with MCP and coding-agent
transports. Read `README.md`, `docs/CLI.md` and `docs/INSTALL-UPDATE.md` before
changing those contracts. Repository code, comments and documentation use English;
respond to the user in their preferred language. Preserve unrelated working-tree
changes. Experimental evidence under ignored `outputs/` is not product code and
must not be added wholesale to commits.

Use the existing BoxWeave Platform MCP for remote execution and resource
management. Inspect resources before creating anything; reuse a suitable target
or create only a dedicated, authorized workload. Develop and test the workload
inside its assigned LXC. Do not modify unrelated instances, provider core
containers, host SSH settings, credentials or routes to make a test pass.

## Research using the BoxWeave MCP environment

When investigating questions about the BoxWeave project, first confirm that the
BoxWeave MCP environment is available using read-only discovery or health checks.
If it is available and a live experiment would help, present the proposed test
steps to the user before running the experiment. Explain the target resources,
expected changes, time limits, success criteria and any planned cleanup. After
the user confirms that plan, use the BoxWeave MCP environment for the agreed
testing or research within that scope. Do not treat environment availability as
authorization to create resources or change state. If the user has already
explicitly approved the same plan and scope, proceed without asking again.

## Timeouts and readiness

- A timed-out create request has an ambiguous outcome, not a confirmed failure.
  Query `list_containers`/`get_container` using the original logical username
  before considering another creation. Never duplicate, force-recreate, delete
  or rename a resource just because the client timed out.
- Running is necessary but does not prove provisioning or SSH is complete.
  Packages and SSH keys may still be installing. Recheck state and retry connect
  within the deadline, then verify `whoami && hostname` before doing work.
- Record one deadline per phase. Unless the user sets another limit, allow up to
  600 seconds per create/provisioning, installation or readiness phase and poll
  about every 15-30 seconds. Do not restart the allowance on each retry. Keep the
  user informed during waits. Capture background job state, logs and actual exit
  codes; a transport timeout or missing completion message is not an exit code.
- The installer's `--api-ready-timeout` bounds the API wait only (default 900
  seconds). When a user limits the entire installation, enforce that separately
  in the runner as well. Do not remove APT locks or start a competing installer
  while package configuration is still active.
- An active Containarium daemon can be waiting for core containers, PostgreSQL
  or an image download before listening on its API. Inspect progress and retry;
  do not restart or delete provider infrastructure to accelerate initialization.
  A definite failure requires diagnosis rather than blind waiting.
- HTTP 200/401/403 indicates API reachability only. Prove authenticated operation
  with actual MCP tool calls. Treat `status`/`doctor` exit 0 as an observation;
  inspect required service/timer states, installation exit code and the pending
  `/etc/boxweave/update-in-progress` marker before reporting installation success.
- Completed oneshots may be inactive/dead with Result=success and ExecMainStatus=0.
  Their managed timers should be active/waiting and enabled. Fresh oneshots may
  be unloaded after daemon-reload: only reset failed state when is-failed reports
  it. Preserve this first-install regression when changing the installer.

## SSH identities and targeting

- Use the verified full managed Ubuntu-to-container public-key line for the host
  whose MCP will connect to the workload, explicitly supplied via ssh_keys. A
  fingerprint is verification evidence, not the full authorization key. Do not
  omit ssh_keys and cause private-key material to be returned.
- Each nesting layer can have a different managed identity. Do not substitute an
  outer host's identity or the Windows/WSL transport key for the nested host's
  key. Verify public keys without reading/exporting private keys or JWTs.
- After IP reuse, host-key mismatches can resolve through the trusted Incus
  reconciler. Allow a bounded retry after synchronization. Do not disable host
  verification, edit known_hosts as a shortcut, or regenerate identities.
  Persistent authentication/connection failures require read-only diagnostics
  (including debug_container where supported) before further changes.
- Tool arguments usually take logical usernames; inventory may show the actual
  Incus name with a -container suffix. Follow the current tool schema, preserve
  exact identity and do not append suffixes speculatively. Local Incus inspection
  must explicitly select the local daemon and intended project. Clones may share
  UUIDs; bind sensitive actions to host, project, exact name and fresh evidence.

## Nested testing

Only run nested provisioning when requested or needed for the authorized task.
The normal smoke test is Ubuntu -> BoxWeave LXC -> inner MCP -> workload LXC ->
Python hello world. Use ordinary Ubuntu images and compatible installed Incus;
do not require a special image, Incus upgrade, cold-backup rehearsal, fixed UID
mapping or PCI/security audit before testing this basic path.

An inner MCP test must start the installed managed launcher as its configured
non-root user, perform a real initialize/initialized exchange and tools/list,
then invoke tools/call for creation and connect. Use a proper MCP client and keep
protocol stdout separate from diagnostics. Do not expose tokens or register a
new fixed server for every box. Direct Incus execution alone is not evidence of
authenticated nested MCP execution.

Check identity via inner MCP connect before executing Python, and preserve the
returned stdout and exit code. Leave a created workload available unless cleanup
was authorized; no network route or persistent application is needed for a
one-off hello-world command. Never infer verified LAN access from an internal IP.

The 2026-10-22 installation and nested smoke results are recorded in
`docs/TEST-BOXWEAVE-LXC.md`. They apply to that tested environment, not every OS,
release or security policy. Child AppArmor labels do not establish the full
parent policy; effective device filtering remains unverified without separate
evidence. Use older research as optional troubleshooting material, not mandatory
product gates.

## Versioning

VERSION follows A.B.C. Increment B when the change adds a significant feature
(for example the 0.4.0 client enrollment redesign); increment C for small
features and fixes. Reserve A for fundamental redesigns that break the product
contract. Bump VERSION in the same commit that introduces the change.

## Change validation

Keep installer reruns usable for update/repair: retain settings unless explicitly
overridden, stage/validate downloads before publication, replace files atomically,
and verify managed services. Do not silently upgrade provider/Incus packages or
erase unrelated configuration to repair BoxWeave. Document recovery limits.

Run relevant existing tests for code changes. Installer sandbox tests must run as
a non-root Linux user; they use temporary paths and fake host tools. Parse shell
and PowerShell changes without executing privileged setup. For documentation-only
edits, check links, consistency and formatting instead of adding mirror tests.
Never report a mocked result as a live integration test or claim a clean Windows
or bare-metal deployment from the nested LXC test. Commit/push only when requested.

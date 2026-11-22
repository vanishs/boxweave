# V1 native Project capability and Forward diagnostics

## Decision

An instance option and a Project-wide permission are different authority levels.
Keep the 22-tool catalog. Ordinary MCP cannot administer Projects, the native
admin socket, or systemd policies. Fresh, exact root-reviewed empty-Project
preparation now grants nesting and proxy device capability, with the prior
proxy value and explicit scope in the reviewed plan, journal and readback.
Service-only staging and normal in-place upgrades still preserve native
Project and unit/drop-in settings. No automatic grant of interception,
low-level configuration, privileged containers or new UID/GID ranges.
An operator may separately review an existing Project's policy without
recreating its profile or touching other Projects.

The service's external Forward policy is independent of the native Proxy
permission. It still requires explicit acknowledgement for non-loopback
binds; all means that public/future interfaces can be included. A successful
native proxy configuration is not a packet witness.

## MCP diagnostics

host_status.native.project_permissions reports only audited permission keys
and normalized observed values; native.external_forward_policy reports the
startup policy mode. Neither claims native kernel support or packet success.
Unknown/unavailable Project reads omit the permission map. Forward policy
refusal is pre-admission; a native permission refusal preserves its durable
original Operation, no native mutation receipt, and an exact FAILED class.
Nesting, interception, low-level, privileged and unavailable ID-map
reservation diagnostics retain the existing fail-closed native guards.

## Development evidence and remaining acceptance

Base cff71e94626cafc4a55c89311b68462325c32a05 (merged PR #168).
Dedicated development Box bw-capability-dev-1121, UUID
7a96bd4e-7f59-4b4e-91d6-4f598c686c81, work_ref project-capabilities-1121.
Go 1.26.9, gofmt/diff gate and full go test -p 2 ./... passed in original
53333b2c22101533b66b5cd480ceb597 (guest exit 0).

The preceding user HTTP fixture was internally healthy, but external
Forward rejected before admission. It therefore does not establish a
native Project denial: the old effective service policy must also be
checked. No L0 host configuration was changed during source development.

The focused hosted native matrix verifies fresh service-only stage does
not grant Proxy, approved native preparation does, stale/incorrect plan
does not write, disabled external policy refuses wildcard, actual HTTP
through exact wildcard listener on loopback and bridge IP, distinct
native Project denial, exact cleanup/rollback, and old-unit policy
preservation across upgrade/rollback. Native acceptance and merge are
pending until exact candidate workflow evidence is available.

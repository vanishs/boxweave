# BoxWeave web console

The console is independent of coding agents and the provider's own web UI. It
uses React, TypeScript and XYFlow; Flask/Waitress and SQLite; a dedicated Caddy
instance and systemd services. No agent runs or model calls occur in the console.
See [verified provider capabilities](WEB-CAPABILITIES.md).

## Installation and local administrator

The revised installer installs and starts Web by default, alongside existing MCP.
For unpublished changes, use the explicit reviewed-checkout option below. For
release installation, omit --source-dir and select a published reviewed ref. The
published older installer cannot install these changes on its own. Ubuntu 24.04 is the development target.

```bash
sudo bash install.sh --source-dir "$PWD" --user YOUR_UNIX_ACCOUNT --web-host YOUR_PRIVATE_IPV4
# HTTPS is the default. Default frontend/backend ports: 18443 / 18444.
sudo boxweave web init-admin
sudo boxweave web status
sudo boxweave doctor
```

Run administrator commands in an interactive terminal **on the installed host**.
Passwords are entered via getpass, never command arguments. There is one local
administrator, no registration form, and no default password. Use a unique
password of at least 8 characters (up to 1024). Resetting it invalidates every
web session:

```bash
sudo boxweave web reset-password
```

The root-owned CLI drops to the web identity before accessing the database.
Uninitialized sites show instructions but refuse protected APIs. Sessions are
stored server-side, expire after eight hours, and use HttpOnly, SameSite=Strict
cookies, plus Secure in HTTPS mode. Writes require CSRF and exact Origin checks.
Login allows five attempts per 15 minutes per backend address; because the proxy
is deliberately not trusted for forwarded identity, this is a shared limit for
proxied clients. It is deliberately conservative for a single-admin server.

The console URL is printed by installation. Address auto-detection requires one
unambiguous private host IPv4 interface (excludes common virtual bridges). Use
`--web-host` if ambiguous. Bind only the intended address. The installer does not
alter host firewall rules or prove LAN-only source filtering. No public hostname
or public internet exposure is required. Backend listens on 127.0.0.1 only.

## HTTPS and trusting the local CA

A separate Caddy instance, admin API disabled, serves the configured IP using
`tls internal`. It does not modify `/etc/caddy`, provider sites or port 80/443.
Only the Caddy binary is extracted from an APT-verified package into the web
release; its distribution service is not installed or enabled by BoxWeave.
Caddy CA/key data remain in `/var/lib/boxweave-web-caddy` across updates. Never
remove this directory as an update/repair shortcut. It contains private CA keys;
back it up securely, not in Git or public archives.

Export **only the public root certificate** using a trusted local host terminal:

```bash
sudo cp /var/lib/boxweave-web-caddy/caddy/pki/authorities/local/root.crt /tmp/boxweave-root.crt
sudo chmod 0644 /tmp/boxweave-root.crt
openssl x509 -in /tmp/boxweave-root.crt -noout -fingerprint -sha256
```

Transfer that public `.crt` through your trusted administrative channel, compare
the SHA256 fingerprint with the host terminal through an independent trusted
channel, then import it into the client's trusted root certificate store. Do not
copy `root.key`, `intermediate.key`, provider signing keys, or MCP credentials.
Do not trust a certificate merely because an untrusted webpage offers it.

- Windows: open `certmgr.msc`, Current User → Trusted Root Certification
  Authorities → Certificates → Import; select the verified root certificate.
- macOS: import into Keychain Access, open the verified certificate, and set its
  SSL trust as appropriate for your account.
- Ubuntu: install the verified certificate under
  `/usr/local/share/ca-certificates/boxweave.crt`, run
  `sudo update-ca-certificates`, then restart the browser. Firefox may require
  its own Authorities import depending on its trust-store settings.

Trusting this CA grants certificate-signing trust to its private-key holder.
Use a dedicated CA and remove client trust when retiring this server. The server
never silently switches to HTTP on certificate failure.

## Explicit HTTP mode

```bash
sudo bash install.sh --web-mode http --web-host YOUR_PRIVATE_IPV4
```

**HTTP sends passwords and session traffic without transport encryption.** Use
only with an explicitly accepted transport risk or a separately trusted encrypted
tunnel. Authentication, CSRF and authorization remain enforced; Secure cookies
are correctly omitted for HTTP. `BOXWEAVE_WEB_MODE`, `BOXWEAVE_WEB_HOST`,
`BOXWEAVE_WEB_PORT`, `BOXWEAVE_WEB_BACKEND_PORT` are corresponding environment
settings. Flags override environment, which overrides saved web settings, which
override defaults. HTTPS remains the fresh-install default.

## Tenant views

Navigation includes Overview, Tenants, Boxes, Network, Monitoring and Prompt Graph.
Tenant display metadata is editable; membership and isolation evidence come from
the Provider. Empty metadata is explicitly identified. Read the [Tenant workflow,
API and migration guide](TENANTS.md), including the authenticated Provider capability gate. On the patched contract,
both batch modes support shared/new/existing Tenants and Automatic creation.
Network -> Tenant Connectivity previews source/destination, forward/reverse/both
directions and exact ingress-only grants/revokes before confirmation. Tenant
detail and Monitoring show policy/Guard state; TCP remains not tested. Access
Prompt Graph cards store both ends and direction without executing policy writes.

## Managing resources

Boxes support list/search/filter/sort/details, batch create with shared or
individual configurations, selected-field CPU/memory/disk resize, graceful
start/stop and delete. Names and parameters are validated on the server. Creation
uses the installed host's explicit managed public key. Batch plans expire after
five minutes; the exact target list must be confirmed. Deletion has a second
confirmation and never uses force implicitly. A running box may require stopping
before deletion. Disk may grow, not shrink.

Two batch workers process at most eight unfinished batches (queued or running;
up to 50 targets each).
Each item has a durable receipt. Partial success does not roll back resources.
`submitted` means create accepted, **not** provisioned or SSH-ready. Use **Query
actual creation state** for submitted/unknown creates. Timeout means outcome
unknown, not failure. Unknown creation names stay reserved to prevent blind
recreation. Diagnose with the actual provider before any manual retry; v1 has
no automatic receipt override. Interrupted jobs become unknown on restart.

Network supports verified route/passthrough list, add and multi-delete only.
Targets are resolved from current box IPs. Existing route identities are not
silently replaced. Raw passthrough adds no TLS or source restrictions. Proxy
routes may expose a public hostname: review the actual scope in the confirmation
plan. The console's existence does not authorize exposing sensitive services.

Monitoring deliberately shows resource state and configured limits, rather than
runtime metrics: upstream v0.114.2 CLI metrics is a placeholder. There are no
fake metric charts or unsupported route toggle buttons.

## Prompt Graph

The visible BoxWeave canvas is a fixed context scope, not a task node. Create
named or unnamed graphs, add resource cards from resource pages, or add free-text
cards on the graph. A resource can appear more than once with different comments.
Select cards/connections to edit titles/comments; duplicate, move or delete cards.
Connections are undirected, and positions imply no task order. Deleting a card
also deletes its incident connections.

Resource-card dialogs show the target graph and allow choosing another saved
graph or creating a new one without first visiting Prompt Graph. All edited
content auto-saves to host SQLite. Blank unedited drafts need not be persisted.
Graph revisions detect conflicting edits; the losing editor retains its local
edits and can save a recovered copy or explicitly discard/reload them. Do not
close a tab with unsaved/conflicted edits. All history remains until explicitly
deleted and confirmed. Current graph selection is stored on the server.

Copy exports fixed `# BoxWeave` and `MCP: boxweave` context, the global prompt,
all card comments and all undirected relation comments. It excludes live resource
properties except stored Tenant context. Single-Tenant graphs add a Tenant line;
mixed graphs retain per-card Tenant labels and old graphs remain compatible.
Duplicate names receive card labels. Copy & new saves and copies
before switching; clipboard failure keeps the graph and opens selectable Markdown.
In HTTP/restricted browsers, manually copy and explicitly choose **complete & new
graph**. Existing graphs remain available. Never paste credentials into comments:
common JWT/private-key/credential patterns are rejected, but arbitrary secrets
cannot reliably be recognized from free text. Review Markdown before sharing it.
An agent without BoxWeave MCP must say it cannot verify server resources.

## Update, backup and recovery

By default, source assets use the installer's resolved Git commit; npm's tracked lock
and pip's exact SHA256 hash lock constrain dependencies. Production builds and
Caddy syntax validation finish before installed BoxWeave files are published.
`--source-dir /absolute/reviewed/checkout` explicitly stages this checkout's listed
source files instead of downloading them. It is not saved as a default for future
updates. The manifest sets `source.local_tree_staged=true` and records per-file
hashes; its resolved upstream commit is a reference, not a claim that locally
modified files equal that commit. Client enrollment still uses published pinned
client assets; local changes to client scripts need a matching published ref.
This option allows installing the Web change without committing or pushing it.

Web releases are immutable content-addressed directories under
`/opt/boxweave-web/releases`. Older releases are retained for manual recovery.
Web JSON, service units, static assets, lock files and source hashes are recorded
in installation provenance. Web keeps its own backups under
`/var/backups/boxweave/web-update-*` and a consistent SQLite backup before migration.
Database and Caddy CA directories are outside replaceable release trees.

MCP timers are resumed before independent Web publication. Web failure does not
stop MCP; it leaves `/etc/boxweave/update-in-progress` so overall installation
cannot be claimed ready. Inspect logs and rerun the revised installer. Publication
is atomic per file, not globally transactional. A failed web update can leave Web
stopped or partly updated; existing graph/credential data are retained. Do not
restore an old database over new writes: first stop Web, keep a private copy of
both files, inspect schema compatibility, and restore matching software/data.
Newer unsupported schema versions fail with a diagnostic rather than downgrade.
SQLite contains password hashes, sessions and prompts; keep backups private.

```bash
sudo systemctl status boxweave-web boxweave-web-caddy boxweave-web-token.timer
sudo journalctl -u boxweave-web -u boxweave-web-caddy -n 60 --no-pager
sudo boxweave web status
sudo boxweave doctor
```

The web identity cannot read MCP credential directories or the provider signing
secret. A separate root service mints a web-subject token with container/route
read/write scopes, two-hour expiry and hourly rotation. Only a recognized
atomic-capable Provider also receives network-policy read/write scopes through
the fixed loopback capability handshake; files are atomically
replaced at 0600 in a root-controlled directory. Old tokens expire within two
hours (rotation is not immediate revocation). Use provider token revocation via
an authorized host operator if immediate compromise response is required.
Provider scopes remain broader than individual Web actions. This is not a claim
of complete least privilege. Web does not control MCP availability.

Application enablement is separate from LXC `boot.autostart`; the latter requires
separate provider evidence. Dedicated-box/fake-provider tests cannot establish
real host installation, external LAN reachability or authenticated infrastructure
mutations. See [test evidence](WEB-TESTING.md).

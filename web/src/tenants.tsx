import { TenantAccessPanel } from "./access";
import { useEffect, useState } from "react";
export type TenantRef = {
  id: string;
  key?: string;
  name?: string;
  source?: string;
};
export type Tenant = TenantRef & {
  description: string;
  providerKey?: string;
  version: number;
  origin: string;
  boxCount: number;
  running: number;
  stopped: number;
  unknown: number;
  guardStatus: string;
  membershipStatus: string;
};
type Props = {
  api: (path: string, method?: string, data?: unknown) => Promise<any>;
  snapshot: any;
  overview: boolean;
  initial: string;
  refresh: () => Promise<void>;
  navigate: (id: string, page: string) => void;
  prompt: (row: any) => void;
  batch: (action: string, rows: any[]) => void;
  fail: (e: any) => void;
};
export function GuardSummary({ snapshot }: { snapshot: any }) {
  const guard = snapshot?.guard;
  if (!guard) return null;
  return (
    <section className="panel tenant-guard">
      <h3>Provider Guard evidence</h3>
      <div className="tenant-grid">
        {["core", "tenant"].map((k) => (
          <div key={k}>
            <strong>
              {k === "core" ? "Core Guard" : "Tenant Guard"}:{" "}
              {guard[k]?.status || "unknown"}
            </strong>
            <p>
              {guard[k]?.mode} · {guard[k]?.firewallDriver}
            </p>
            <small>Last pass: {guard[k]?.lastPass || "unavailable"}</small>
          </div>
        ))}
      </div>
      <p>
        Protected Boxes:{" "}
        {
          (snapshot.containers || []).filter(
            (b: any) => b.tenant?.guard?.status === "verified",
          ).length
        }{" "}
        · Unresolved: {guard.tenant?.unresolved?.join(", ") || "none reported"}{" "}
        · Unmanaged: {guard.tenant?.unmanaged?.join(", ") || "none reported"}
      </p>
      <p className="muted">
        Verified means fresh ENFORCE evidence and a matching attached NIC ACL.
        Bidirectional TCP connectivity has not been tested. Same-Tenant access
        still depends on running services and other controls. Gateway, proxy,
        published ports and outbound traffic follow separate paths. Tenant
        membership shares neither files nor users and does not establish a
        cross-host layer-2 network.
      </p>
    </section>
  );
}
export function TenantPanel(p: Props) {
  const [id, setId] = useState(p.initial),
    [query, setQuery] = useState(""),
    [status, setStatus] = useState(""),
    [tab, setTab] = useState("Overview"),
    [detail, setDetail] = useState<any>(null),
    [edit, setEdit] = useState<Tenant | null | undefined>(undefined),
    [name, setName] = useState(""),
    [description, setDescription] = useState(""),
    [selected, setSelected] = useState<string[]>([]);
  const tenants: Tenant[] = p.snapshot?.tenants || [];
  const tenant = tenants.find((t) => t.id === id);
  useEffect(() => {
    setId(p.initial);
    setTab("Overview");
  }, [p.initial]);
  useEffect(() => {
    setSelected([]);
  }, [id]);
  useEffect(() => {
    setDetail(null);
    if (id)
      p.api("tenants/" + encodeURIComponent(id) + "/network")
        .then(setDetail)
        .catch(p.fail);
  }, [id, p.snapshot]);
  const boxes = (p.snapshot?.containers || []).filter(
    (b: any) => b.tenant?.id === id,
  );
  function editing(t: Tenant | null) {
    setEdit(t);
    setName(t?.name || "");
    setDescription(t?.description || "");
  }
  async function save() {
    await p.api(
      "tenants" + (edit ? "/" + edit.id : ""),
      edit ? "PATCH" : "POST",
      { name, description, ...(edit ? { version: edit.version } : {}) },
    );
    setEdit(undefined);
    await p.refresh();
  }
  async function remove() {
    if (!tenant) return;
    const d = await p.api("tenants/" + tenant.id);
    if (d.deletionBlockers?.length) throw Error(d.deletionBlockers.join("; "));
    if (
      !window.confirm(
        "Delete only this empty Tenant’s local metadata? Saved resources are preserved.",
      )
    )
      return;
    await p.api("tenants/" + tenant.id, "DELETE", {
      confirm: true,
      version: tenant.version,
    });
    setId("");
    await p.refresh();
  }
  function promptTenant(t: Tenant) {
    p.prompt({
      resourceId: t.id,
      resourceType: "tenant",
      name: t.name,
      tenant: {
        id: t.id,
        ...(t.providerKey && t.boxCount > 0 ? { key: t.providerKey } : {}),
        name: t.name,
        source: !t.boxCount ? "local" : "resource-reference",
      },
    });
  }
  return (
    <>
      {p.overview && (
        <>
          <div className="stats">
            <section>
              <small>TENANTS</small>
              <strong>{tenants.length}</strong>
            </section>
            <section>
              <small>BOXES</small>
              <strong>{p.snapshot?.containers?.length || 0}</strong>
            </section>
            <section>
              <small>UNRESOLVED BOXES</small>
              <strong>
                {
                  (p.snapshot?.containers || []).filter(
                    (b: any) => !b.tenant?.id,
                  ).length
                }
              </strong>
            </section>
          </div>
          <GuardSummary snapshot={p.snapshot} />
        </>
      )}
      <div className="panel tenant-panel">
        <div className="toolbar">
          <input
            aria-label="Search Tenants"
            placeholder="Search Tenants…"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
          <select
            aria-label="Filter Tenant status"
            value={status}
            onChange={(e) => setStatus(e.target.value)}
          >
            <option value="">All Guard states</option>
            {[
              "verified",
              "empty",
              "unknown",
              "pending",
              "degraded",
              "unsupported",
            ].map((x) => (
              <option key={x}>{x}</option>
            ))}
          </select>
          <button className="primary" onClick={() => editing(null)}>
            + Create Tenant
          </button>
        </div>
        <p className="muted">
          Names and descriptions are local display metadata. Empty Tenants have
          no Provider network. Resource membership and ACL evidence come from
          Containarium.
        </p>
        <div className="table-scroll">
          <table>
            <thead>
              <tr>
                <th>TENANT / STABLE ID</th>
                <th>BOXES</th>
                <th>RUNNING / STOPPED / UNKNOWN</th>
                <th>GUARD</th>
                <th>DESCRIPTION</th>
                <th>CONTEXT</th>
              </tr>
            </thead>
            <tbody>
              {tenants
                .filter(
                  (t) =>
                    (t.name + " " + t.id + " " + t.description)
                      .toLowerCase()
                      .includes(query.toLowerCase()) &&
                    (!status || t.guardStatus === status),
                )
                .map((t) => (
                  <tr key={t.id}>
                    <td>
                      <button
                        className="text-button"
                        onClick={() => {
                          setId(t.id);
                          setTab("Overview");
                        }}
                      >
                        {t.name}
                      </button>
                      <small className="stable-id">{t.id}</small>
                    </td>
                    <td>{t.boxCount}</td>
                    <td>
                      {t.running} / {t.stopped} / {t.unknown}
                    </td>
                    <td>
                      <span className="badge">{t.guardStatus}</span>
                      <small>Membership: {t.membershipStatus}</small>
                    </td>
                    <td>{t.description || "—"}</td>
                    <td>
                      <button onClick={() => promptTenant(t)}>
                        + Prompt card
                      </button>
                    </td>
                  </tr>
                ))}
            </tbody>
          </table>
        </div>
      </div>
      {tenant && (
        <section className="panel tenant-panel">
          <div className="toolbar">
            <h2>Tenant: {tenant.name}</h2>
            <button onClick={() => editing(tenant)}>
              Edit display metadata
            </button>
            <button onClick={() => remove().catch(p.fail)}>
              Delete empty metadata
            </button>
          </div>
          <p className="stable-id">Stable ID: {tenant.id}</p>
          <p>
            Provider key (resolved or inferred):{" "}
            {tenant.providerKey || "None — local metadata only"}
          </p>
          <div className="tabs">
            {[
              "Overview",
              "Boxes",
              "Network",
              "Monitoring",
              "Prompt Context",
            ].map((x) => (
              <button
                key={x}
                className={tab === x ? "active" : ""}
                onClick={() => setTab(x)}
              >
                {x}
                {x === "Boxes" ? " (" + tenant.boxCount + ")" : ""}
              </button>
            ))}
          </div>
          {tab === "Overview" && (
            <>
              <p>{tenant.description || "No description"}</p>
              <p>
                Guard: {tenant.guardStatus}. Empty metadata does not create a
                shared network.
              </p>
              <button onClick={() => p.navigate(tenant.id, "Boxes")}>
                Open scoped Boxes
              </button>
            </>
          )}
          {tab === "Boxes" && (
            <>
              <div className="toolbar">
                {["start", "stop", "delete"].map((action) => (
                  <button
                    key={action}
                    disabled={!selected.length}
                    onClick={() =>
                      p.batch(
                        action,
                        boxes
                          .filter((b: any) => selected.includes(b.username))
                          .map((b: any) => ({ username: b.username })),
                      )
                    }
                  >
                    {action} selected
                  </button>
                ))}
                <button onClick={() => p.navigate(tenant.id, "Boxes")}>
                  Open Boxes / create / resize
                </button>
              </div>
              <table>
                <thead>
                  <tr>
                    <th>SELECT</th>
                    <th>BOX</th>
                    <th>STATE</th>
                    <th>IP</th>
                    <th>GUARD</th>
                    <th>CONTEXT</th>
                  </tr>
                </thead>
                <tbody>
                  {boxes.map((b: any) => (
                    <tr key={b.username}>
                      <td>
                        <input
                          type="checkbox"
                          aria-label={"Select Tenant Box " + b.username}
                          checked={selected.includes(b.username)}
                          onChange={(e) =>
                            setSelected((s) =>
                              e.target.checked
                                ? [...s, b.username]
                                : s.filter((x) => x !== b.username),
                            )
                          }
                        />
                      </td>
                      <td>{b.username}</td>
                      <td>{b.state}</td>
                      <td>{b.network?.ipAddress || "—"}</td>
                      <td>{b.tenant.guard.status}</td>
                      <td>
                        <button
                          onClick={() =>
                            p.prompt({ ...b, resourceType: "box" })
                          }
                        >
                          + Prompt card
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </>
          )}
          {tab === "Network" && (
            <>
              <GuardSummary snapshot={{ ...p.snapshot, containers: boxes }} />
              <h3>Internal Network — Box addresses</h3>
              {boxes.map((b: any) => (
                <p key={b.username}>
                  {b.username}: {b.network?.ipAddress || "unavailable"} ·{" "}
                  {b.tenant.guard.status}
                </p>
              ))}
              <TenantAccessPanel
                api={p.api}
                snapshot={p.snapshot}
                scope={tenant.id}
                readOnly
                prompt={p.prompt}
                fail={p.fail}
              />
              <button onClick={() => p.navigate(tenant.id, "Network")}>
                Manage Tenant access in Network
              </button>
              <h3>Associated routes (exact resource identity)</h3>
              <pre>
                {JSON.stringify(
                  {
                    routes: detail?.routes || [],
                    passthrough: detail?.passthrough || [],
                  },
                  null,
                  2,
                )}
              </pre>
              {!p.snapshot?.capabilities?.networkPolicyWrite && (
                <p>
                  {p.snapshot?.capabilities?.policyReason ||
                    "Policy writes are unavailable in this Provider contract."}{" "}
                  Tenant reassignment is unavailable in every contract.
                </p>
              )}
            </>
          )}
          {tab === "Monitoring" && (
            <>
              <p>
                Boxes: {tenant.boxCount} · Running: {tenant.running} · Stopped:{" "}
                {tenant.stopped} · Unknown: {tenant.unknown}
              </p>
              <p>
                Guard: {tenant.guardStatus}. Live CPU, memory and disk
                utilization are unavailable in the current CLI contract.
              </p>
              <button onClick={() => p.navigate(tenant.id, "Monitoring")}>
                Open scoped Monitoring
              </button>
            </>
          )}
          {tab === "Prompt Context" && (
            <>
              <p>
                Add a stable Tenant reference. A graph may include multiple
                Tenants; links remain undirected context associations.
              </p>
              <button onClick={() => promptTenant(tenant)}>
                + Tenant prompt card
              </button>
            </>
          )}
        </section>
      )}
      {edit !== undefined && (
        <div className="modal-backdrop">
          <section
            className="modal"
            role="dialog"
            aria-modal="true"
            aria-label="Tenant metadata"
          >
            <h2>{edit ? "Edit" : "Create"} Tenant metadata</h2>
            <label>
              Name
              <input value={name} onChange={(e) => setName(e.target.value)} />
            </label>
            <label>
              Description
              <textarea
                value={description}
                onChange={(e) => setDescription(e.target.value)}
              />
            </label>
            <p>
              This changes local metadata only. It does not create, merge or
              move a Provider network Tenant.
            </p>
            <button onClick={() => save().catch(p.fail)}>Save Tenant</button>
            <button onClick={() => setEdit(undefined)}>Cancel</button>
          </section>
        </div>
      )}
    </>
  );
}

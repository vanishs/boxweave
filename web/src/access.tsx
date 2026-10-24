import { useEffect, useState } from "react";
import type { Tenant } from "./tenants";
type Props = {
  api: (path: string, method?: string, data?: unknown) => Promise<any>;
  snapshot: any;
  scope?: string;
  readOnly?: boolean;
  prompt: (row: any) => void;
  fail: (error: any) => void;
  changed?: () => Promise<void>;
};
export function TenantAccessPanel(p: Props) {
  const [data, setData] = useState<any>(null),
    [source, setSource] = useState(""),
    [destination, setDestination] = useState(""),
    [direction, setDirection] = useState("forward"),
    [operation, setOperation] = useState("grant"),
    [plan, setPlan] = useState<any>(null),
    [notice, setNotice] = useState(""),
    [pending, setPending] = useState(false);
  const tenants: Tenant[] = p.snapshot?.tenants || [];
  async function refresh() {
    setData(
      await p.api("tenant-access?tenant=" + encodeURIComponent(p.scope || "")),
    );
  }
  useEffect(() => {
    refresh().catch(p.fail);
  }, [p.scope, p.snapshot]);
  useEffect(() => {
    if (p.scope && p.scope !== "unresolved") setSource(p.scope);
  }, [p.scope]);
  async function review() {
    setPending(true);
    try {
      setPlan(
        await p.api("tenant-access/plans", "POST", {
          source,
          destination,
          direction,
          operation,
        }),
      );
    } finally {
      setPending(false);
    }
  }
  async function confirm() {
    setPending(true);
    try {
      const receipt = await p.api(
        "tenant-access/plans/" + plan.id + "/confirm",
        "POST",
        { confirm: true },
      );
      const item = receipt.items[0];
      setNotice(
        item.status +
          (item.message
            ? ": " + item.message
            : " — configured policy and ACL evidence are separate from TCP connectivity"),
      );
      setPlan(null);
      await refresh();
      await p.changed?.();
    } finally {
      setPending(false);
    }
  }
  function addContext(relation: any) {
    const src = relation.source,
      dst = relation.destination;
    p.prompt({
      resourceId: "access:" + src.id + ":" + dst.id,
      resourceType: "tenant-access",
      name: src.name + " → " + dst.name,
      tenant: {
        id: dst.id,
        key: dst.providerKey,
        name: dst.name,
        source: "provider-policy",
      },
      access: {
        source: { id: src.id, key: src.providerKey, name: src.name },
        destination: { id: dst.id, key: dst.providerKey, name: dst.name },
        direction: "source-to-destination",
        policyStatus: relation.policyStatus,
        verificationStatus: "not_tested",
      },
    });
  }
  return (
    <section className="panel tenant-panel">
      <div className="toolbar">
        <h2>
          {p.readOnly
            ? "External Tenant Access"
            : "Tenant Connectivity / Tenant Access"}
        </h2>
        <button onClick={() => refresh().catch(p.fail)}>
          Refresh policies
        </button>
      </div>
      <p>
        {data?.scopeNotice ||
          "Tenant-wide source-to-destination ingress authorization. Egress, service listeners and other controls still apply."}
      </p>
      <p className="muted">
        Authorization records are not end-to-end connectivity results. No TCP
        connectivity test has been performed. Unlisted pairs have no explicit
        Tenant access grant; other network paths have separate controls.
      </p>
      {data?.policyReadState === "unknown" && (
        <p className="warning">
          Network Policy inventory is unavailable. Do not infer that no grants
          exist.
        </p>
      )}
      {notice && (
        <p
          className={
            /^(unknown|failed|conflict|degraded)/.test(notice)
              ? "error"
              : "warning"
          }
          role="status"
        >
          {notice}
        </p>
      )}
      {!p.readOnly && (
        <>
          <div className="form-grid">
            <label>
              Source Tenant
              <select
                aria-label="Source Tenant"
                value={source}
                onChange={(e) => setSource(e.target.value)}
              >
                <option value="">Select source</option>
                {tenants.map((t) => (
                  <option key={t.id} value={t.id}>
                    {t.name} · {t.providerKey || "no Provider membership"}
                  </option>
                ))}
              </select>
            </label>
            <label>
              Destination Tenant
              <select
                aria-label="Destination Tenant"
                value={destination}
                onChange={(e) => setDestination(e.target.value)}
              >
                <option value="">Select destination</option>
                {tenants.map((t) => (
                  <option key={t.id} value={t.id}>
                    {t.name} · {t.providerKey || "no Provider membership"}
                  </option>
                ))}
              </select>
            </label>
          </div>
          <div className="form-grid">
            <label>
              Access direction
              <select
                aria-label="Access direction"
                value={direction}
                onChange={(e) => setDirection(e.target.value)}
              >
                <option value="forward">Source → Destination</option>
                <option value="reverse">Destination → Source</option>
                <option value="both">Both directions ↔</option>
              </select>
            </label>
            <label>
              Operation
              <select
                aria-label="Access operation"
                value={operation}
                onChange={(e) => setOperation(e.target.value)}
              >
                <option value="grant">Grant ingress access</option>
                <option value="revoke">Revoke only this access</option>
              </select>
            </label>
          </div>
          {!data?.capabilities?.networkPolicyWrite && (
            <p className="warning">
              {data?.capabilities?.policyReason ||
                "Provider lacks the authenticated atomic Tenant access contract. Egress and deny policies will not be replaced as a workaround."}
            </p>
          )}
          <button
            className="primary"
            disabled={
              pending ||
              !source ||
              !destination ||
              source === destination ||
              !data?.capabilities?.networkPolicyWrite ||
              data?.policyReadState !== "available"
            }
            onClick={() => review().catch(p.fail)}
          >
            Review policy changes
          </button>
        </>
      )}
      <h3>Configured ingress permissions</h3>
      <div className="table-scroll">
        <table>
          <thead>
            <tr>
              <th>SOURCE → DESTINATION</th>
              <th>DIRECTION</th>
              <th>POLICY</th>
              <th>GUARD</th>
              <th>CONNECTIVITY</th>
              <th>CONTEXT</th>
            </tr>
          </thead>
          <tbody>
            {(data?.relations || []).map((r: any, i: number) => (
              <tr key={i}>
                <td>
                  {r.source.name} → {r.destination.name}
                  <small>
                    {r.source.providerKey} → {r.destination.providerKey}
                  </small>
                </td>
                <td>Source may initiate</td>
                <td>{r.policyStatus}</td>
                <td>{r.guardStatus}</td>
                <td>Not tested</td>
                <td>
                  <button
                    disabled={!r.source.id || !r.destination.id}
                    onClick={() => addContext(r)}
                  >
                    + Access prompt card
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {!data?.relations?.length && (
        <p>
          {data?.policyReadState === "available"
            ? "No explicit ingress grants in this scope."
            : "Authorization state unknown."}
        </p>
      )}
      <h3>Tenant access summary</h3>
      {tenants
        .filter((t) => !p.scope || p.scope === "unresolved" || t.id === p.scope)
        .map((t) => (
          <div key={t.id}>
            <strong>
              {t.name} · Guard {t.guardStatus}
            </strong>
            <p>
              Inbound permissions:{" "}
              {(data?.relations || [])
                .filter((r: any) => r.destination.id === t.id)
                .map((r: any) => r.source.name + " → " + t.name)
                .join(", ") ||
                (data?.policyReadState === "available"
                  ? "none configured"
                  : "unknown")}
              <br />
              Outbound tenant access:{" "}
              {(data?.relations || [])
                .filter((r: any) => r.source.id === t.id)
                .map((r: any) => t.name + " → " + r.destination.name)
                .join(", ") ||
                (data?.policyReadState === "available"
                  ? "none configured"
                  : "unknown")}
            </p>
          </div>
        ))}
      <p className="muted">
        Outbound tenant access means a destination's ingress policy allows this
        source; it does not prove that all outbound traffic is permitted.
      </p>
      <details>
        <summary>Current Network Policies (read only)</summary>
        <pre>{JSON.stringify(data?.policies || [], null, 2)}</pre>
      </details>
      <details>
        <summary>Recent policy operation receipts</summary>
        <pre>{JSON.stringify(data?.recentOperations || [], null, 2)}</pre>
      </details>
      {plan && (
        <div className="modal-backdrop">
          <section
            className="modal wide"
            role="dialog"
            aria-modal="true"
            aria-label="Review Tenant policy"
          >
            <h2>Review Tenant policy changes</h2>
            <p className="warning">{plan.scopeNotice}</p>
            <p>
              Only listed destination allow_from_tenants entries change.
              Existing Egress CIDR/Domain, Mode, Metadata, Preset and Deny Rules
              are preserved. A revision conflict aborts both directions.
            </p>
            {plan.preview.map((item: any, i: number) => (
              <section key={i}>
                <h3>
                  {item.source.name} → {item.destination.name}
                </h3>
                <p>Destination policy: {item.destination.providerKey}</p>
                <p>
                  Before: {item.before.join(", ") || "none"}
                  <br />
                  After: {item.after.join(", ") || "none"}
                </p>
                <details>
                  <summary>Preserved policy and observed revision</summary>
                  <pre>{JSON.stringify(item, null, 2)}</pre>
                </details>
              </section>
            ))}
            <button
              disabled={pending}
              className={
                plan.intent.operation === "revoke" ? "danger" : "primary"
              }
              onClick={() => confirm().catch(p.fail)}
            >
              Confirm Tenant policy {plan.intent.operation}
            </button>
            <button disabled={pending} onClick={() => setPlan(null)}>
              Cancel
            </button>
          </section>
        </div>
      )}
    </section>
  );
}

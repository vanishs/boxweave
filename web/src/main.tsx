import React, { useEffect, useRef, useState } from "react";
import { createRoot } from "react-dom/client";
import {
  ReactFlow,
  Background,
  Controls,
  Handle,
  Position,
  applyNodeChanges,
  applyEdgeChanges,
  type NodeProps,
  type Node,
  type Edge,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import "./style.css";
import {
  blank,
  payload,
  createItems,
  copyAndNext,
  type Graph,
  type Card,
} from "./model";
let csrf = "";
async function api(path: string, method = "GET", data?: unknown) {
  const r = await fetch("/api/" + path, {
    method,
    headers: { "Content-Type": "application/json", "X-CSRF-Token": csrf },
    body: data === undefined ? undefined : JSON.stringify(data),
  });
  const d = await r.json();
  if (!r.ok) throw Error(d.error || "Request failed");
  return d;
}
function PromptCard({
  data,
}: NodeProps<Node<{ title: string; comment: string; kind: string }>>) {
  return (
    <div className="prompt-card">
      <Handle type="target" position={Position.Left} />
      <small>{data.kind === "text" ? "NOTE" : "RESOURCE"}</small>
      <strong>{data.title || "Untitled card"}</strong>
      <p>{data.comment || "Add a comment…"}</p>
      <Handle type="source" position={Position.Right} />
    </div>
  );
}
const nodeTypes = { card: PromptCard };
type Row = Record<string, any>;
function App() {
  const [auth, setAuth] = useState<any>(null),
    [password, setPassword] = useState(""),
    [error, setError] = useState(""),
    [page, setPage] = useState("Boxes");
  const [rows, setRows] = useState<Row[]>([]),
    [selected, setSelected] = useState<string[]>([]),
    [query, setQuery] = useState(""),
    [filter, setFilter] = useState(""),
    [sort, setSort] = useState("name"),
    [kind, setKind] = useState("route");
  const [graphs, setGraphs] = useState<Graph[]>([]),
    [graph, setGraph] = useState<Graph>(blank()),
    [dirty, setDirty] = useState(false),
    [saving, setSaving] = useState(false),
    [conflict, setConflict] = useState(false);
  const graphRef = useRef(graph),
    saveRef = useRef<Promise<void> | null>(null),
    epoch = useRef(0);
  graphRef.current = graph;
  const [dialog, setDialog] = useState<string | null>(null),
    [detail, setDetail] = useState<Row | null>(null),
    [card, setCard] = useState({
      title: "",
      comment: "",
      type: "box",
      id: "",
      target: "",
    });
  const [cardTargets, setCardTargets] = useState<Row[]>([]);
  const [names, setNames] = useState(""),
    [mode, setMode] = useState("shared"),
    [shared, setShared] = useState({ cpu: "2", memory: "2GB", disk: "10GB" }),
    [image, setImage] = useState("images:ubuntu/24.04"),
    [individual, setIndividual] = useState<Row[]>([]);
  const [fields, setFields] = useState<string[]>([]),
    [plan, setPlan] = useState<any>(null),
    [jobs, setJobs] = useState<any[]>([]),
    [network, setNetwork] = useState({
      username: "",
      domain: "",
      targetPort: 8080,
      externalPort: 18080,
      protocol: "TCP",
      description: "",
    });
  const [editNode, setEditNode] = useState(""),
    [editEdge, setEditEdge] = useState(""),
    [md, setMd] = useState(""),
    [manual, setManual] = useState(false);
  const fail = (e: any) => setError(String(e.message || e));
  useEffect(() => {
    api("status")
      .then((d) => {
        setAuth(d);
        csrf = d.csrf || "";
      })
      .catch(fail);
  }, []);
  async function loadGraphs() {
    const ticket = epoch.current;
    const d = await api("graphs");
    setGraphs(d.graphs);
    if (!graphRef.current.id && ticket === epoch.current) {
      const g = d.graphs.find((g: Graph) => g.id === d.current);
      if (g) {
        setGraph(g);
        epoch.current++;
      }
    }
  }
  async function refresh() {
    setError("");
    setSelected([]);
    if (page === "Prompt Graph") return;
    const d = await api(
      page === "Boxes"
        ? "boxes"
        : page === "Monitoring"
          ? "metrics"
          : "network/" + kind,
    );
    setRows(d.containers || d.metrics || d.routes || []);
    if (d.unreachableBackends?.length)
      setError("Some backends are unreachable. Results are incomplete.");
  }
  useEffect(() => {
    if (auth?.authenticated) {
      loadGraphs().catch(fail);
      api("jobs")
        .then((d) => setJobs(d.jobs))
        .catch(fail);
    }
  }, [auth?.authenticated]);
  useEffect(() => {
    if (auth?.authenticated) refresh().catch(fail);
  }, [page, kind, auth?.authenticated]);
  useEffect(() => {
    if (!jobs.some((j) => !j.finished)) return;
    const t = setInterval(
      () =>
        api("jobs")
          .then((d) => setJobs(d.jobs))
          .catch(fail),
      2000,
    );
    return () => clearInterval(t);
  }, [jobs]);
  function change(next: Graph) {
    graphRef.current = next;
    setGraph(next);
    epoch.current++;
    setDirty(true);
  }
  async function save() {
    if (saveRef.current) await saveRef.current;
    for (let attempt = 0; attempt < 20; attempt++) {
      const ticket = epoch.current;
      await saveOnce();
      if (ticket === epoch.current) return;
    }
    throw Error("Graph is still changing. Pause editing and try again.");
  }
  async function saveOnce() {
    if (saveRef.current) {
      await saveRef.current;
      return saveOnce();
    }
    if (conflict) throw Error("Resolve the graph conflict first");
    const snapshot = graphRef.current,
      ticket = epoch.current;
    if (
      !snapshot.id &&
      !snapshot.name &&
      !snapshot.prompt &&
      !snapshot.nodes.length &&
      !snapshot.edges.length
    )
      return;
    const task = (async () => {
      setSaving(true);
      try {
        const result = await api(
          "graphs" + (snapshot.id ? "/" + snapshot.id : ""),
          snapshot.id ? "PUT" : "POST",
          snapshot.id
            ? { ...payload(snapshot), version: snapshot.version }
            : payload(snapshot),
        );
        if (ticket === epoch.current) {
          graphRef.current = result;
          setGraph(result);
          setDirty(false);
        } else {
          const updated = {
            ...graphRef.current,
            id: result.id,
            version: result.version,
          };
          graphRef.current = updated;
          setGraph(updated);
        }
        setGraphs((gs) => [result, ...gs.filter((g) => g.id !== result.id)]);
      } catch (e: any) {
        if (e.message.includes("conflict")) setConflict(true);
        throw e;
      } finally {
        setSaving(false);
      }
    })();
    saveRef.current = task;
    try {
      await task;
    } finally {
      saveRef.current = null;
    }
  }
  useEffect(() => {
    if (!dirty || conflict) return;
    const t = setTimeout(() => save().catch(fail), 650);
    return () => clearTimeout(t);
  }, [graph, dirty, conflict]);
  useEffect(() => {
    const leave = (e: BeforeUnloadEvent) => {
      if (dirty) {
        e.preventDefault();
        e.returnValue = "";
      }
    };
    window.addEventListener("beforeunload", leave);
    return () => window.removeEventListener("beforeunload", leave);
  }, [dirty]);
  async function switchGraph(id: string) {
    if (dirty || saveRef.current) await save();
    const g = id ? await api("graphs/" + id) : blank();
    if (id) await api("graphs/" + id + "/select", "POST", {});
    epoch.current++;
    graphRef.current = g;
    setGraph(g);
    setDirty(false);
    setEditNode("");
    setEditEdge("");
  }
  async function exportMarkdown() {
    await save();
    if (!graphRef.current.id) return "# BoxWeave\nMCP: boxweave\n";
    return (await api("graphs/" + graphRef.current.id + "/markdown")).markdown;
  }
  async function copy(next: boolean) {
    try {
      if (next)
        await copyAndNext(
          save,
          exportMarkdown,
          (s) => navigator.clipboard.writeText(s),
          () => {
            const draft = blank();
            epoch.current++;
            graphRef.current = draft;
            setGraph(draft);
            setDirty(false);
            setEditNode("");
            setEditEdge("");
          },
        );
      else {
        const s = await exportMarkdown();
        await navigator.clipboard.writeText(s);
      }
    } catch (e) {
      setMd(await exportMarkdown());
      setManual(next);
      setDialog("Markdown");
      setError(
        "Automatic copy unavailable. Select and copy the Markdown below.",
      );
    }
  }
  function key(r: Row) {
    return (
      r.username ||
      r.fullDomain ||
      r.domain ||
      (r.externalPort ? `${r.externalPort}/${r.protocol}` : r.name)
    );
  }
  function addPrompt(r: Row) {
    setCardTargets([r]);
    setCard({
      title: r.username || r.fullDomain || r.domain || r.name || key(r),
      comment: "",
      type: page === "Boxes" ? "box" : page === "Monitoring" ? "metric" : kind,
      id: key(r),
      target: graph.id,
    });
    setDialog("Add prompt card");
  }
  async function addCard() {
    await save();
    let target: Graph =
      card.target === "new"
        ? blank()
        : card.target && card.target !== graphRef.current.id
          ? await api("graphs/" + card.target)
          : graphRef.current;
    const added: Card[] = cardTargets.map((r, index) => ({
      id: crypto.randomUUID(),
      type: "resource",
      title:
        cardTargets.length === 1
          ? card.title
          : r.username || r.fullDomain || r.domain || r.name || key(r),
      comment: card.comment,
      resource: { type: card.type, id: key(r) },
      position: {
        x: 60 + ((target.nodes.length + index) % 3) * 260,
        y: 60 + Math.floor((target.nodes.length + index) / 3) * 220,
      },
    }));
    target = { ...target, nodes: [...target.nodes, ...added] };
    epoch.current++;
    graphRef.current = target;
    setGraph(target);
    setDirty(true);
    await saveOnce();
    await api("graphs/" + graphRef.current.id + "/select", "POST", {});
    setDialog(null);
  }
  async function makePlan(action: string, items: Row[]) {
    setPlan(await api("plans", "POST", { action, items }));
    setDialog("Confirm plan");
  }
  async function confirmPlan() {
    const j = await api("plans/" + plan.id + "/confirm", "POST", {
      confirm: true,
    });
    setJobs((js) => [j, ...js]);
    setDialog(null);
    setPlan(null);
  }
  function switchMode(m: string) {
    if (m === "individual") {
      const byName = new Map(individual.map((r) => [r.username, r]));
      setIndividual(
        createItems(names, shared, image).map(
          (i) => byName.get(i.username) || i,
        ),
      );
    } else {
      setNames(individual.map((i) => i.username).join("\n"));
    }
    setMode(m);
  }
  const filtered = rows
    .filter(
      (r) =>
        JSON.stringify(r).toLowerCase().includes(query.toLowerCase()) &&
        (!filter || String(r.state || r.protocol || "").includes(filter)),
    )
    .sort((a, b) =>
      String(a[sort] || key(a)).localeCompare(String(b[sort] || key(b))),
    );
  const chosen = rows.filter((r) => selected.includes(key(r)));
  const currentNode = graph.nodes.find((n) => n.id === editNode),
    currentEdge = graph.edges.find((e) => e.id === editEdge);
  function updateNode(p: Partial<Card>) {
    change({
      ...graphRef.current,
      nodes: graphRef.current.nodes.map((n) =>
        n.id === editNode ? { ...n, ...p } : n,
      ),
    });
  }
  if (!auth)
    return (
      <main className="login">
        <h1>BoxWeave</h1>
        <p>{error || "Connecting…"}</p>
      </main>
    );
  if (!auth.authenticated)
    return (
      <main className="login">
        <div className="brand">
          ▦ BoxWeave <span>CONSOLE</span>
        </div>
        <h1>
          Your machines.
          <br />A clearer view.
        </h1>
        {!auth.initialized ? (
          <>
            <p>Initialize your local administrator on the server:</p>
            <code>sudo boxweave web init-admin</code>
          </>
        ) : (
          <form
            onSubmit={(e) => {
              e.preventDefault();
              api("login", "POST", { password })
                .then((d) => {
                  csrf = d.csrf;
                  setPassword("");
                  setAuth({ ...auth, authenticated: true });
                  setError("");
                })
                .catch(fail);
            }}
          >
            <label>
              Administrator password
              <input
                type="password"
                autoComplete="current-password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
              />
            </label>
            <button className="primary">Sign in →</button>
          </form>
        )}
        {!auth.https && (
          <p className="warning">
            HTTP mode: credentials and session traffic are not encrypted.
          </p>
        )}
        {error && (
          <p role="alert" className="error">
            {error}
          </p>
        )}
      </main>
    );
  return (
    <div className="shell">
      <aside>
        <div className="brand">▦ BoxWeave</div>
        <small className="eyebrow">PERSONAL INFRASTRUCTURE</small>
        <nav>
          {["Boxes", "Network", "Monitoring", "Prompt Graph"].map((p, i) => (
            <button
              className={page === p ? "active" : ""}
              key={p}
              onClick={() => {
                setPage(p);
                setQuery("");
                setFilter("");
              }}
            >
              <span>{["▣", "⇄", "◴", "⌘"][i]}</span>
              {p}
            </button>
          ))}
        </nav>
        <div className="sidebar-bottom">
          <span className="dot" /> Local administrator
          <button
            onClick={() =>
              api("logout", "POST", {})
                .then(() => location.reload())
                .catch(fail)
            }
          >
            Sign out
          </button>
        </div>
      </aside>
      <main>
        <header>
          <div>
            <small className="eyebrow">
              YOUR SERVER / {page.toUpperCase()}
            </small>
            <h1>{page}</h1>
            <p>
              {page === "Prompt Graph"
                ? "Arrange context. Connect ideas. Bring your own agent."
                : "A focused view of your BoxWeave infrastructure."}
            </p>
          </div>
          <div className="header-actions">
            {page === "Prompt Graph" ? (
              <>
                <button onClick={() => copy(false).catch(fail)}>
                  Copy Markdown
                </button>
                <button
                  className="primary"
                  onClick={() => copy(true).catch(fail)}
                >
                  Copy & new graph ↗
                </button>
              </>
            ) : (
              <>
                <button onClick={() => refresh().catch(fail)}>↻ Refresh</button>
                {page !== "Monitoring" && (
                  <button
                    className="primary"
                    onClick={() =>
                      setDialog(
                        page === "Boxes" ? "Create boxes" : "Create route",
                      )
                    }
                  >
                    + {page === "Boxes" ? "Create boxes" : "Create route"}
                  </button>
                )}
              </>
            )}
          </div>
        </header>
        {error && (
          <div className="error" role="alert">
            {error}
            <button onClick={() => setError("")}>Dismiss</button>
          </div>
        )}
        {!auth.https && (
          <div className="warning">
            HTTP mode · Transport encryption is disabled.
          </div>
        )}
        {page !== "Prompt Graph" ? (
          <>
            <div className="stats">
              <section>
                <small>
                  {page === "Monitoring"
                    ? "RESOURCES REPORTING"
                    : "TOTAL " + page.toUpperCase()}
                </small>
                <strong>{rows.length}</strong>
              </section>
              <section>
                <small>SELECTED</small>
                <strong>{selected.length}</strong>
              </section>
              <section>
                <small>CONTEXT CARDS</small>
                <strong>{graph.nodes.length}</strong>
                <span>in {graph.name || "current graph"}</span>
              </section>
            </div>
            <div className="panel">
              <div className="toolbar">
                <input
                  aria-label="Search resources"
                  placeholder="Search resources…"
                  value={query}
                  onChange={(e) => setQuery(e.target.value)}
                />
                {page === "Network" && (
                  <select
                    value={kind}
                    onChange={(e) => setKind(e.target.value)}
                  >
                    <option value="route">Proxy routes</option>
                    <option value="passthrough">TCP / UDP passthrough</option>
                  </select>
                )}
                <select
                  aria-label="Filter"
                  value={filter}
                  onChange={(e) => setFilter(e.target.value)}
                >
                  <option value="">All states / protocols</option>
                  {[
                    ...new Set(
                      rows.map((r) => r.state || r.protocol).filter(Boolean),
                    ),
                  ].map((s) => (
                    <option key={s}>{s}</option>
                  ))}
                </select>
                <select
                  aria-label="Sort"
                  value={sort}
                  onChange={(e) => setSort(e.target.value)}
                >
                  <option value="name">Name</option>
                  <option value="state">State</option>
                </select>
              </div>
              {chosen.length > 0 && (
                <div className="bulk">
                  <strong>{chosen.length} selected</strong>
                  {page === "Boxes" ? (
                    <>
                      {["start", "stop"].map((a) => (
                        <button
                          key={a}
                          onClick={() =>
                            makePlan(
                              a,
                              chosen.map((r) => ({ username: r.username })),
                            ).catch(fail)
                          }
                        >
                          {a}
                        </button>
                      ))}
                      <button
                        onClick={() => {
                          setFields([]);
                          setDialog("Resize boxes");
                        }}
                      >
                        Modify resources
                      </button>
                      <button
                        className="danger"
                        onClick={() =>
                          makePlan(
                            "delete",
                            chosen.map((r) => ({ username: r.username })),
                          ).catch(fail)
                        }
                      >
                        Delete
                      </button>
                    </>
                  ) : page === "Network" ? (
                    <button
                      className="danger"
                      onClick={() =>
                        makePlan(
                          kind + "-delete",
                          chosen.map((r) =>
                            kind === "route"
                              ? { domain: r.fullDomain || r.domain }
                              : {
                                  externalPort: r.externalPort,
                                  protocol: String(r.protocol).replace(
                                    "ROUTE_PROTOCOL_",
                                    "",
                                  ),
                                },
                          ),
                        ).catch(fail)
                      }
                    >
                      Delete selected
                    </button>
                  ) : (
                    <button
                      onClick={() => {
                        addPrompt(chosen[0]);
                        setCardTargets(chosen);
                      }}
                    >
                      Add prompt card
                    </button>
                  )}
                </div>
              )}
              <div className="table-scroll">
                <table>
                  <thead>
                    <tr>
                      <th>
                        <input
                          aria-label="Select all visible"
                          type="checkbox"
                          checked={
                            !!filtered.length &&
                            filtered.every((r) => selected.includes(key(r)))
                          }
                          onChange={(e) =>
                            setSelected(
                              e.target.checked ? filtered.map(key) : [],
                            )
                          }
                        />
                      </th>
                      <th>{page === "Network" ? "ROUTE" : "RESOURCE"}</th>
                      <th>
                        {page === "Monitoring"
                          ? "RESOURCE STATE"
                          : "STATE / PROTOCOL"}
                      </th>
                      <th>
                        {page === "Boxes"
                          ? "CPU / MEMORY / DISK"
                          : page === "Monitoring"
                            ? "CPU / MEMORY / DISK"
                            : "TARGET"}
                      </th>
                      <th>CONTEXT</th>
                    </tr>
                  </thead>
                  <tbody>
                    {filtered.map((r) => (
                      <tr key={key(r)}>
                        <td>
                          <input
                            aria-label={"Select " + key(r)}
                            type="checkbox"
                            checked={selected.includes(key(r))}
                            onChange={(e) =>
                              setSelected((s) =>
                                e.target.checked
                                  ? [...s, key(r)]
                                  : s.filter((x) => x !== key(r)),
                              )
                            }
                          />
                        </td>
                        <td>
                          <button
                            className="text-button"
                            onClick={() => {
                              if (page === "Boxes")
                                api("boxes/" + r.username)
                                  .then((d) => {
                                    setDetail(d);
                                    setDialog("Resource details");
                                  })
                                  .catch(fail);
                              else {
                                setDetail(r);
                                setDialog("Resource details");
                              }
                            }}
                          >
                            {key(r)}
                          </button>
                          {r.network?.ipAddress && (
                            <small>{r.network.ipAddress}</small>
                          )}
                        </td>
                        <td>
                          {page === "Monitoring" ? (
                            String(r.state || "unknown").replace(
                              "CONTAINER_STATE_",
                              "",
                            )
                          ) : (
                            <span className="badge">
                              {String(r.state || r.protocol || "unknown")
                                .replace("CONTAINER_STATE_", "")
                                .replace("ROUTE_PROTOCOL_", "")}
                            </span>
                          )}
                        </td>
                        <td>
                          {page === "Boxes"
                            ? `${r.resources?.cpu || "—"} / ${r.resources?.memory || "—"} / ${r.resources?.disk || "—"}`
                            : page === "Monitoring"
                              ? `${r.resources?.cpu || "—"} / ${r.resources?.memory || "—"} / ${r.resources?.disk || "—"}`
                              : `${r.targetIp || r.containerIp || "—"}:${r.targetPort || r.port || "—"}`}
                        </td>
                        <td>
                          <button onClick={() => addPrompt(r)}>
                            + Prompt card
                          </button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
                {!filtered.length && (
                  <div className="empty">
                    <strong>No resources to show</strong>
                    <p>Create a resource or change your filters.</p>
                  </div>
                )}
              </div>
            </div>
            {page === "Network" && (
              <p className="muted">
                Only verified create, list and delete operations are available.
                Passthrough provides neither TLS nor source filtering. Confirm
                exposure scope before creating a route.
              </p>
            )}
            {jobs.length > 0 && (
              <section className="panel receipts">
                <h2>Operation receipts</h2>
                {jobs.slice(0, 10).map((j) => (
                  <details key={j.id} open={!j.finished}>
                    <summary>
                      {j.action} · {j.items.length} targets ·{" "}
                      {j.finished
                        ? "completed; inspect individual results"
                        : "in progress"}
                    </summary>
                    {j.items.map((i: any, index: number) => (
                      <div className="receipt" key={index}>
                        <code>
                          {i.target.username ||
                            i.target.domain ||
                            i.target.externalPort}
                        </code>
                        <strong>{i.status}</strong>
                        <span>
                          {i.observedState} {i.message}
                        </span>
                      </div>
                    ))}
                    {j.action === "create" && j.finished && (
                      <button
                        onClick={() =>
                          api("jobs/" + j.id + "/reconcile", "POST", {})
                            .then((r) =>
                              setJobs((js) =>
                                js.map((x) => (x.id === r.id ? r : x)),
                              ),
                            )
                            .catch(fail)
                        }
                      >
                        Query actual creation state
                      </button>
                    )}
                  </details>
                ))}
              </section>
            )}
          </>
        ) : (
          <div className="graph-layout">
            <section className="panel graph-list">
              <div className="toolbar">
                <h2>Saved graphs</h2>
                <button onClick={() => switchGraph("").catch(fail)}>
                  + New
                </button>
              </div>
              <button
                className={!graph.id ? "active" : ""}
                onClick={() => switchGraph("").catch(fail)}
              >
                New unsaved graph
              </button>
              {graphs.map((g) => (
                <button
                  className={g.id === graph.id ? "active" : ""}
                  key={g.id}
                  onClick={() => switchGraph(g.id).catch(fail)}
                >
                  {g.name || "Untitled graph"}
                  <small>
                    {g.nodes.length} cards · revision {g.version}
                  </small>
                </button>
              ))}
              <p className="muted">
                All graphs are retained. Nothing runs automatically.
              </p>
            </section>
            <section className="panel graph-main">
              <div className="graph-heading">
                <input
                  aria-label="Graph name"
                  placeholder="Untitled graph"
                  value={graph.name}
                  onChange={(e) =>
                    change({ ...graphRef.current, name: e.target.value })
                  }
                />
                <small>
                  {conflict
                    ? "Conflict · local edits retained"
                    : saving
                      ? "Saving…"
                      : dirty
                        ? "Unsaved"
                        : "Saved"}{" "}
                  · {graph.nodes.length} cards
                </small>
                {graph.id && (
                  <button
                    className="danger"
                    onClick={() => setDialog("Delete graph")}
                  >
                    Delete graph
                  </button>
                )}
              </div>
              {conflict && (
                <div className="warning">
                  Another editor saved this graph. Copy your local text before
                  reloading.
                  <button
                    onClick={() => {
                      api("graphs/" + graph.id)
                        .then((g) => {
                          epoch.current++;
                          setConflict(false);
                          setDirty(false);
                          setGraph(g);
                          graphRef.current = g;
                        })
                        .catch(fail);
                    }}
                  >
                    Discard local edits & reload
                  </button>
                  <button
                    onClick={() => {
                      setConflict(false);
                      const g = {
                        ...graph,
                        id: "",
                        version: 0,
                        name: graph.name + " (recovered)",
                      };
                      change(g);
                    }}
                  >
                    Save local edits as a new graph
                  </button>
                </div>
              )}
              <label className="prompt-label">
                YOUR PROMPT
                <textarea
                  placeholder="What would you like your agent to do?"
                  value={graph.prompt}
                  onChange={(e) =>
                    change({ ...graphRef.current, prompt: e.target.value })
                  }
                />
              </label>
              <div className="canvas-tools">
                <strong>
                  ▦ BoxWeave <span className="muted">Root scope</span>
                </strong>
                <button
                  onClick={() => {
                    const n: Card = {
                      id: crypto.randomUUID(),
                      type: "text",
                      title: "New note",
                      comment: "",
                      position: {
                        x: 60 + (graphRef.current.nodes.length % 3) * 260,
                        y:
                          60 +
                          Math.floor(graphRef.current.nodes.length / 3) * 220,
                      },
                    };
                    change({ ...graph, nodes: [...graph.nodes, n] });
                    setEditNode(n.id);
                    setEditEdge("");
                  }}
                >
                  + Text card
                </button>
                <small>Connections are undirected</small>
              </div>
              <div className="canvas-root">
                <ReactFlow<Node, Edge>
                  nodes={graph.nodes.map((n) => ({
                    id: n.id,
                    type: "card",
                    position: n.position,
                    data: { title: n.title, comment: n.comment, kind: n.type },
                  }))}
                  edges={graph.edges.map((e) => ({
                    ...e,
                    label: e.title,
                    markerEnd: undefined,
                    markerStart: undefined,
                  }))}
                  nodeTypes={nodeTypes}
                  onNodesChange={(changes) => {
                    const graph = graphRef.current;
                    const ns = applyNodeChanges(
                      changes,
                      graph.nodes.map((n) => ({
                        id: n.id,
                        position: n.position,
                        data: {},
                      })),
                    );
                    if (
                      changes.some(
                        (c) => c.type === "position" || c.type === "remove",
                      )
                    ) {
                      const ids = new Set(ns.map((n) => n.id));
                      change({
                        ...graph,
                        nodes: graph.nodes
                          .filter((n) => ids.has(n.id))
                          .map((n) => ({
                            ...n,
                            position: ns.find((x) => x.id === n.id)!.position,
                          })),
                        edges: graph.edges.filter(
                          (e) => ids.has(e.source) && ids.has(e.target),
                        ),
                      });
                    }
                  }}
                  onEdgesChange={(changes) => {
                    const graph = graphRef.current;
                    if (changes.some((c) => c.type === "remove")) {
                      const es = applyEdgeChanges(
                        changes,
                        graph.edges as Edge[],
                      );
                      change({
                        ...graph,
                        edges: graph.edges.filter((e) =>
                          es.some((x) => x.id === e.id),
                        ),
                      });
                    }
                  }}
                  onConnect={(c) => {
                    if (c.source && c.target && c.source !== c.target) {
                      const id = crypto.randomUUID();
                      change({
                        ...graph,
                        edges: [
                          ...graph.edges,
                          {
                            id,
                            source: c.source,
                            target: c.target,
                            title: "",
                            comment: "",
                          },
                        ],
                      });
                      setEditEdge(id);
                      setEditNode("");
                    }
                  }}
                  onNodeClick={(_, n) => {
                    setEditNode(n.id);
                    setEditEdge("");
                  }}
                  onEdgeClick={(_, e) => {
                    setEditEdge(e.id);
                    setEditNode("");
                  }}
                  fitView
                  fitViewOptions={{ maxZoom: 1, padding: 0.2 }}
                  maxZoom={1.5}
                >
                  <Background gap={22} />
                  <Controls />
                </ReactFlow>
              </div>
              <p className="muted">
                BoxWeave is the fixed context container. Card positions and
                connections imply no execution order.
              </p>
            </section>
            <section className="panel inspector">
              <h2>
                {currentNode
                  ? "Edit card"
                  : currentEdge
                    ? "Edit connection"
                    : "Card details"}
              </h2>
              {currentNode ? (
                <>
                  <small>
                    {currentNode.type} · {currentNode.resource?.id}
                  </small>
                  <label>
                    Title
                    <input
                      value={currentNode.title}
                      onChange={(e) => updateNode({ title: e.target.value })}
                    />
                  </label>
                  <label>
                    Comment
                    <textarea
                      value={currentNode.comment}
                      onChange={(e) => updateNode({ comment: e.target.value })}
                    />
                  </label>
                  <button
                    onClick={() => {
                      change({
                        ...graph,
                        nodes: [
                          ...graph.nodes,
                          {
                            ...currentNode,
                            id: crypto.randomUUID(),
                            position: {
                              x: currentNode.position.x + 260,
                              y: currentNode.position.y + 40,
                            },
                          },
                        ],
                      });
                    }}
                  >
                    Duplicate
                  </button>
                  <button
                    className="danger"
                    onClick={() => {
                      change({
                        ...graph,
                        nodes: graph.nodes.filter((n) => n.id !== editNode),
                        edges: graph.edges.filter(
                          (e) => e.source !== editNode && e.target !== editNode,
                        ),
                      });
                      setEditNode("");
                    }}
                  >
                    Delete card
                  </button>
                </>
              ) : currentEdge ? (
                <>
                  <label>
                    Title
                    <input
                      value={currentEdge.title}
                      onChange={(e) =>
                        change({
                          ...graph,
                          edges: graph.edges.map((x) =>
                            x.id === editEdge
                              ? { ...x, title: e.target.value }
                              : x,
                          ),
                        })
                      }
                    />
                  </label>
                  <label>
                    Comment
                    <textarea
                      value={currentEdge.comment}
                      onChange={(e) =>
                        change({
                          ...graph,
                          edges: graph.edges.map((x) =>
                            x.id === editEdge
                              ? { ...x, comment: e.target.value }
                              : x,
                          ),
                        })
                      }
                    />
                  </label>
                  <button
                    className="danger"
                    onClick={() => {
                      change({
                        ...graph,
                        edges: graph.edges.filter((e) => e.id !== editEdge),
                      });
                      setEditEdge("");
                    }}
                  >
                    Delete connection
                  </button>
                </>
              ) : (
                <p className="muted">
                  Select a card or a connection to edit it. Add resource cards
                  from Boxes, Network or Monitoring.
                </p>
              )}
            </section>
          </div>
        )}
      </main>
      {dialog && (
        <div className="modal-backdrop">
          <section
            className={"modal " + (dialog === "Create boxes" ? "wide" : "")}
            role="dialog"
            aria-modal="true"
            aria-label={dialog}
          >
            <div className="modal-title">
              <h2>{dialog}</h2>
              <button aria-label="Close dialog" onClick={() => setDialog(null)}>
                ×
              </button>
            </div>
            {dialog === "Create boxes" && (
              <>
                <div className="tabs">
                  <button
                    className={mode === "shared" ? "active" : ""}
                    onClick={() => switchMode("shared")}
                  >
                    Unified configuration
                  </button>
                  <button
                    className={mode === "individual" ? "active" : ""}
                    onClick={() => switchMode("individual")}
                  >
                    Individual configuration
                  </button>
                </div>
                {mode === "shared" ? (
                  <>
                    <label>
                      Box names (one per line)
                      <textarea
                        value={names}
                        onChange={(e) => setNames(e.target.value)}
                        placeholder={"api-dev\nworker-dev"}
                      />
                    </label>
                    <div className="form-grid">
                      {Object.keys(shared).map((k) => (
                        <label key={k}>
                          {k.toUpperCase()}
                          <input
                            value={(shared as any)[k]}
                            onChange={(e) =>
                              setShared({ ...shared, [k]: e.target.value })
                            }
                          />
                        </label>
                      ))}
                    </div>
                    <label>
                      Image
                      <input
                        value={image}
                        onChange={(e) => setImage(e.target.value)}
                      />
                    </label>
                  </>
                ) : (
                  <>
                    <div className="table-scroll">
                      <table>
                        <thead>
                          <tr>
                            {["Name", "CPU", "Memory", "Disk", "Image", ""].map(
                              (x) => (
                                <th key={x}>{x}</th>
                              ),
                            )}
                          </tr>
                        </thead>
                        <tbody>
                          {individual.map((r, i) => (
                            <tr key={i}>
                              {[
                                "username",
                                "cpu",
                                "memory",
                                "disk",
                                "image",
                              ].map((k) => (
                                <td key={k}>
                                  <input
                                    aria-label={k + " row " + (i + 1)}
                                    value={
                                      k in r.resources ? r.resources[k] : r[k]
                                    }
                                    onChange={(e) =>
                                      setIndividual((rs) =>
                                        rs.map((x, j) =>
                                          i !== j
                                            ? x
                                            : k in x.resources
                                              ? {
                                                  ...x,
                                                  resources: {
                                                    ...x.resources,
                                                    [k]: e.target.value,
                                                  },
                                                }
                                              : { ...x, [k]: e.target.value },
                                        ),
                                      )
                                    }
                                  />
                                </td>
                              ))}
                              <td>
                                <button
                                  onClick={() =>
                                    setIndividual((rs) =>
                                      rs.filter((_, j) => i !== j),
                                    )
                                  }
                                >
                                  ×
                                </button>
                              </td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                    <button
                      onClick={() =>
                        setIndividual((rs) => [
                          ...rs,
                          { username: "", resources: { ...shared }, image },
                        ])
                      }
                    >
                      + Add row
                    </button>
                  </>
                )}
                <p className="muted">
                  Up to 50 boxes. The server checks names and validates all
                  parameters before creating a confirmation plan. The installed
                  managed SSH public key is always supplied.
                </p>
                <button
                  className="primary"
                  onClick={() =>
                    makePlan(
                      "create",
                      mode === "shared"
                        ? createItems(names, shared, image)
                        : individual,
                    ).catch(fail)
                  }
                >
                  Review creation plan →
                </button>
              </>
            )}
            {dialog === "Resize boxes" && (
              <>
                <p>
                  Only checked fields will change on {chosen.length} boxes. Disk
                  can only grow.
                </p>
                {Object.keys(shared).map((k) => (
                  <label key={k}>
                    <span>
                      <input
                        type="checkbox"
                        checked={fields.includes(k)}
                        onChange={(e) =>
                          setFields((fs) =>
                            e.target.checked
                              ? [...fs, k]
                              : fs.filter((x) => x !== k),
                          )
                        }
                      />
                      {k}
                    </span>
                    <input
                      disabled={!fields.includes(k)}
                      value={(shared as any)[k]}
                      onChange={(e) =>
                        setShared({ ...shared, [k]: e.target.value })
                      }
                    />
                  </label>
                ))}
                <button
                  className="primary"
                  onClick={() =>
                    makePlan(
                      "resize",
                      chosen.map((r) => ({
                        username: r.username,
                        resources: Object.fromEntries(
                          fields.map((k) => [k, (shared as any)[k]]),
                        ),
                      })),
                    ).catch(fail)
                  }
                >
                  Review changes
                </button>
              </>
            )}
            {dialog === "Confirm plan" && plan && (
              <>
                <p>
                  Confirm {plan.action} for exactly these targets.{" "}
                  {plan.action.includes("delete")
                    ? "Deletion is permanent."
                    : ""}
                </p>
                <pre>{JSON.stringify(plan.items, null, 2)}</pre>
                {plan.action.includes("add") && (
                  <p className="warning">
                    This creates a network exposure. Verify authentication,
                    source restrictions and intended clients. No LAN-only
                    enforcement is implied.
                  </p>
                )}
                <button
                  className={
                    plan.action.includes("delete") ? "danger" : "primary"
                  }
                  onClick={() => confirmPlan().catch(fail)}
                >
                  Confirm {plan.action}
                </button>
              </>
            )}
            {dialog === "Create route" && (
              <>
                <p>
                  {kind === "route"
                    ? "Proxy route"
                    : "Raw TCP / UDP passthrough"}
                  . The target IP is resolved from the box on the server.
                </p>
                {[
                  "username",
                  ...(kind === "route"
                    ? ["domain"]
                    : ["externalPort", "protocol"]),
                  "targetPort",
                  "description",
                ].map((k) => (
                  <label key={k}>
                    {k}
                    <input
                      value={(network as any)[k]}
                      onChange={(e) =>
                        setNetwork({
                          ...network,
                          [k]: k.endsWith("Port")
                            ? Number(e.target.value)
                            : e.target.value,
                        })
                      }
                    />
                  </label>
                ))}
                <button
                  className="primary"
                  onClick={() => {
                    const {
                      username,
                      targetPort,
                      description,
                      domain,
                      externalPort,
                      protocol,
                    } = network;
                    makePlan(kind + "-add", [
                      kind === "route"
                        ? { username, targetPort, description, domain }
                        : {
                            username,
                            targetPort,
                            description,
                            externalPort,
                            protocol,
                          },
                    ]).catch(fail);
                  }}
                >
                  Review route plan
                </button>
              </>
            )}
            {dialog === "Add prompt card" && (
              <>
                {cardTargets.length > 1 && (
                  <p>
                    Adding {cardTargets.length} independent cards. The comment
                    applies to each card; edit them separately in Prompt Graph.
                  </p>
                )}
                <label>
                  Title
                  <input
                    value={card.title}
                    onChange={(e) =>
                      setCard({ ...card, title: e.target.value })
                    }
                  />
                </label>
                <small>
                  {card.type} · {card.id}
                </small>
                <label>
                  Comment
                  <textarea
                    value={card.comment}
                    onChange={(e) =>
                      setCard({ ...card, comment: e.target.value })
                    }
                  />
                </label>
                <label>
                  Target graph
                  <select
                    value={card.target}
                    onChange={(e) =>
                      setCard({ ...card, target: e.target.value })
                    }
                  >
                    <option value="">
                      {graph.name || "Current / new untitled graph"}
                    </option>
                    {graphs.map((g) => (
                      <option key={g.id} value={g.id}>
                        {g.name || "Untitled graph"} · {g.id.slice(0, 8)}
                      </option>
                    ))}
                    <option value="new">Create new untitled graph</option>
                  </select>
                </label>
                <button
                  className="primary"
                  onClick={() => addCard().catch(fail)}
                >
                  Add card & save
                </button>
              </>
            )}
            {dialog === "Resource details" && (
              <pre>{JSON.stringify(detail, null, 2)}</pre>
            )}
            {dialog === "Delete graph" && (
              <>
                <p>
                  Permanently delete “{graph.name || "Untitled graph"}” and all
                  its cards and connections?
                </p>
                <button
                  className="danger"
                  onClick={() => {
                    api("graphs/" + graph.id, "DELETE", {
                      version: graph.version,
                      confirm: true,
                    })
                      .then(() => {
                        setGraphs((gs) => gs.filter((g) => g.id !== graph.id));
                        epoch.current++;
                        setGraph(blank());
                        graphRef.current = blank();
                        setDirty(false);
                        setDialog(null);
                      })
                      .catch(fail);
                  }}
                >
                  Confirm deletion
                </button>
              </>
            )}
            {dialog === "Markdown" && (
              <>
                <textarea
                  className="markdown"
                  readOnly
                  value={md}
                  onFocus={(e) => e.target.select()}
                />
                <p>
                  Use your own connected agent. If BoxWeave MCP is unavailable,
                  the agent must say so.
                </p>
                {manual && (
                  <button
                    className="primary"
                    onClick={() => {
                      switchGraph("")
                        .then(() => setDialog(null))
                        .catch(fail);
                    }}
                  >
                    I have copied it · complete & new graph
                  </button>
                )}
              </>
            )}
            {error && (
              <p className="error" role="alert">
                {error}
              </p>
            )}
          </section>
        </div>
      )}
    </div>
  );
}
createRoot(document.getElementById("root")!).render(<App />);

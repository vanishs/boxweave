export type Card = {
  id: string;
  type: "text" | "resource";
  title: string;
  comment: string;
  resource?: {
    type: string;
    id: string;
    tenant?: { id: string; key?: string; name?: string; source?: string };
  } | null;
  metadata?: Record<string, string>;
  position: { x: number; y: number };
};
export type Link = {
  id: string;
  source: string;
  target: string;
  title: string;
  comment: string;
};
export type Graph = {
  id: string;
  version: number;
  name: string;
  prompt: string;
  nodes: Card[];
  edges: Link[];
};
export const blank = (): Graph => ({
  id: "",
  version: 0,
  name: "",
  prompt: "",
  nodes: [],
  edges: [],
});
export const payload = ({ name, prompt, nodes, edges }: Graph) => ({
  name,
  prompt,
  nodes,
  edges,
});
export async function copyAndNext(
  save: () => Promise<void>,
  get: () => Promise<string>,
  copy: (s: string) => Promise<void>,
  next: () => void,
) {
  await save();
  const text = await get();
  await copy(text);
  next();
}
export function createItems(
  names: string,
  shared: { cpu: string; memory: string; disk: string },
  image: string,
  tenantId = "",
) {
  return names
    .split(/[\s,]+/)
    .filter(Boolean)
    .map((username) => ({
      username,
      resources: { ...shared },
      image,
      tenantId,
    }));
}

// Expire verification in an open tab even when a Provider refresh fails.
export function evidenceFresh(stamp: string, now = Date.now()) {
  const age = (now - Date.parse(stamp)) / 1000;
  return Number.isFinite(age) && age >= -30 && age <= 180;
}
export function expireBoxEvidence(row: any, now = Date.now()) {
  if (
    row.tenant?.guard?.status !== "verified" ||
    evidenceFresh(row.tenant.guard.lastPass, now)
  )
    return row;
  return {
    ...row,
    tenant: {
      ...row.tenant,
      confirmed: false,
      guard: {
        ...row.tenant.guard,
        status: "unknown",
        reason: "Guard evidence expired; refresh Provider state",
      },
    },
  };
}
export function expireEvidence(snapshot: any, now = Date.now()) {
  if (!snapshot) return snapshot;
  const guard = { ...snapshot.guard };
  for (const key of ["core", "tenant"]) {
    if (guard[key]?.fresh && !evidenceFresh(guard[key].lastPass, now))
      guard[key] = { ...guard[key], fresh: false, status: "unknown" };
  }
  return {
    ...snapshot,
    guard,
    containers: snapshot.containers.map((r: any) => expireBoxEvidence(r, now)),
    tenants: snapshot.tenants.map((t: any) =>
      t.guardStatus === "verified" && !guard.tenant?.fresh
        ? { ...t, guardStatus: "unknown", membershipStatus: "inferred" }
        : t,
    ),
  };
}

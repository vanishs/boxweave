export type Card = {
  id: string;
  type: "text" | "resource";
  title: string;
  comment: string;
  resource?: { type: string; id: string } | null;
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
) {
  return names
    .split(/[\s,]+/)
    .filter(Boolean)
    .map((username) => ({ username, resources: { ...shared }, image }));
}

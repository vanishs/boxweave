"""Validated, unordered prompt cards and stable minimal Markdown."""

import re

SECRET = re.compile(
    r"-----BEGIN (?:[A-Z ]*PRIVATE KEY)-----|\beyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+|\b(?:password|token|secret)\s*[:=]\s*\S+",
    re.I,
)


def text(value, limit=20000):
    if not isinstance(value, str) or len(value) > limit or "\x00" in value:
        raise ValueError("Invalid text or text too long")
    if SECRET.search(value):
        raise ValueError("Remove credentials from prompt text")
    return value


def validate(data):
    if not isinstance(data, dict) or set(data) - {"name", "prompt", "nodes", "edges"}:
        raise ValueError("Invalid graph")
    result = {
        "name": text(data.get("name", ""), 200),
        "prompt": text(data.get("prompt", "")),
        "nodes": [],
        "edges": [],
    }
    nodes = data.get("nodes", [])
    edges = data.get("edges", [])
    if (
        not isinstance(nodes, list)
        or not isinstance(edges, list)
        or len(nodes) > 500
        or len(edges) > 2000
    ):
        raise ValueError("Graph too large")
    ids = set()
    for n in nodes:
        if not isinstance(n, dict) or set(n) - {
            "id",
            "type",
            "title",
            "comment",
            "resource",
            "position",
            "metadata",
        }:
            raise ValueError("Invalid node")
        nid = text(n.get("id"), 100)
        if not re.fullmatch(r"[a-zA-Z0-9_-]{1,100}", nid) or nid in ids:
            raise ValueError("Invalid or duplicate node ID")
        ids.add(nid)
        if n.get("type") not in ("resource", "text"):
            raise ValueError("Invalid node type")
        pos = n.get("position", {"x": 0, "y": 0})
        if set(pos) != {"x", "y"} or any(
            type(v) not in (float, int) or not -100000 <= v <= 100000
            for v in pos.values()
        ):
            raise ValueError("Invalid position")
        ref = n.get("resource")
        if ref is not None:
            if (
                not isinstance(ref, dict)
                or set(ref) != {"type", "id"}
                or ref["type"] not in ("box", "route", "passthrough", "metric")
            ):
                raise ValueError("Invalid reference")
            ref = {k: text(v, 200) for k, v in ref.items()}
        metadata = n.get("metadata", {})
        if not isinstance(metadata, dict) or len(metadata) > 20:
            raise ValueError("Invalid metadata")
        metadata = {text(k, 100): text(v, 1000) for k, v in metadata.items()}
        result["nodes"].append(
            dict(
                metadata=metadata,
                id=nid,
                type=n["type"],
                title=text(n.get("title", ""), 500),
                comment=text(n.get("comment", "")),
                resource=ref,
                position=pos,
            )
        )
    edgeids = set()
    for e in edges:
        if not isinstance(e, dict) or set(e) - {
            "id",
            "source",
            "target",
            "title",
            "comment",
        }:
            raise ValueError("Invalid edge fields")
        eid = text(e.get("id"), 100)
        if not re.fullmatch(r"[a-zA-Z0-9_-]{1,100}", eid):
            raise ValueError("Invalid edge ID")
        if (
            eid in edgeids
            or e.get("source") not in ids
            or e.get("target") not in ids
            or e["source"] == e["target"]
        ):
            raise ValueError("Invalid edge")
        edgeids.add(eid)
        result["edges"].append(
            dict(
                id=eid,
                source=e["source"],
                target=e["target"],
                title=text(e.get("title", ""), 500),
                comment=text(e.get("comment", "")),
            )
        )
    return result


def escape(value):
    return re.sub(r"([\\`*_{}\[\]<>#!|])", r"\\\1", value).replace("\n", " ")


def markdown(graph):
    g = validate({k: graph[k] for k in ("name", "prompt", "nodes", "edges")})
    nodes = sorted(g["nodes"], key=lambda n: n["id"])
    names = [n["title"] for n in nodes]
    labels = {
        n["id"]: escape(n["title"] or "Untitled")
        + (
            f' [card {n["id"][:8] if sum(x["id"][:8]==n["id"][:8] for x in nodes)==1 else n["id"]}]'
            if names.count(n["title"]) > 1
            else ""
        )
        for i, n in enumerate(nodes)
    }
    lines = ["# BoxWeave", "MCP: boxweave", ""]
    if g["prompt"]:
        lines += [g["prompt"], ""]
    for n in nodes:
        lines += ["- " + labels[n["id"]]]
        if n["comment"]:
            lines += ["  - " + line for line in n["comment"].split("\n")]
        lines += [""]
    if g["edges"]:
        lines += ["Relations (undirected):"]
        for e in sorted(g["edges"], key=lambda e: e["id"]):
            a, b = sorted((e["source"], e["target"]))
            lines += [
                f"- {labels[a]} -- {labels[b]}"
                + (": " + escape(e["title"]) if e["title"] else "")
            ]
            if e["comment"]:
                lines += ["  " + line for line in e["comment"].split("\n")]
    return "\n".join(lines).rstrip() + "\n"

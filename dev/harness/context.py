"""Discover canonical assets from the index and skill metadata."""
import re
from pathlib import Path

from .common import HarnessError, repo_path

LINK = re.compile(r"\[[^\]]+\]\(([^)\s]+)\)")
TOKEN = re.compile(r"[a-z0-9]+")
DOMAINS = {
    "internal/install": "installer maintenance install",
    "internal/diagnostic": "diagnostic recovery maintenance",
    "internal/recovery": "recovery operations diagnostic",
    "internal/peerscope": "peer network operations",
    "internal/networklimits": "peer network",
    "internal/native": "native nesting network operations",
    "internal/state": "operations checkpoint",
    "internal/control": "mcp operations security",
    "dev": "development skill workflow checkpoint",
    "scripts": "tool testing verification",
}


def assets(root):
    items = {}
    for line in repo_path(root, "dev/INDEX.md").read_text(encoding="utf-8").splitlines():
        if not line.startswith("|"):
            continue
        cells = [cell.strip() for cell in line.strip("|").split("|")]
        if len(cells) != 4 or not re.fullmatch(r"[a-z][a-z0-9-]*(\.[a-z0-9-]+)+", cells[0]):
            continue
        asset_id, summary, links, state = cells
        if asset_id in items:
            raise HarnessError(f"Duplicate asset ID: {asset_id}")
        paths = []
        for href in LINK.findall(links):
            if "://" in href or href.startswith("#"):
                continue
            path = repo_path(root, "dev/" + href.split("#")[0])
            if not path.is_file():
                raise HarnessError(f"Missing asset: {asset_id}: {href}")
            paths.append(path.relative_to(root.resolve()).as_posix())
        if not paths:
            raise HarnessError(f"No local reference: {asset_id}")
        items[asset_id] = {"id": asset_id, "summary": summary, "paths": paths, "status": state}
    if not items:
        raise HarnessError("No assets in dev/INDEX.md")
    for skill in sorted((root / "dev/skills").glob("*/SKILL.md")):
        if not skill.resolve().is_relative_to(root.resolve()):
            raise HarnessError("Skill path leaves repository")
        match = re.match(r"---\nname: ([a-z0-9-]+)\ndescription: ([^\n]+)\n---\n",
                         skill.read_text(encoding="utf-8"))
        if not match or match[1] != skill.parent.name:
            raise HarnessError(f"Invalid skill header: {skill.parent.name}")
        asset_id = "skill." + match[1]
        path = skill.relative_to(root).as_posix()
        if asset_id in items:
            if path not in items[asset_id]["paths"]:
                raise HarnessError(f"Skill index mismatch: {asset_id}")
            items[asset_id]["description"] = match[2]
        else:
            items[asset_id] = {"id": asset_id, "summary": match[2], "paths": [path],
                               "status": "Candidate", "description": match[2]}
    return list(items.values())


def register(parser):
    parser.add_argument("--path", action="append", default=[])
    parser.add_argument("--query", default="")
    parser.add_argument("--limit", type=int, default=8)
    parser.add_argument("--all", action="store_true")


def run(args, root):
    if not 1 <= args.limit <= 50:
        raise HarnessError("limit must be in 1..50")
    items = assets(root)
    words = set(TOKEN.findall(args.query.lower()))
    local = set()
    for path in args.path:
        candidate = repo_path(root, path)
        rel = candidate.relative_to(root.resolve()).as_posix()
        words.update(TOKEN.findall(rel.lower()))
        for prefix, domain in DOMAINS.items():
            if rel == prefix or rel.startswith(prefix + "/"):
                words.update(TOKEN.findall(domain))
        directory = candidate if candidate.is_dir() else candidate.parent
        while directory != root.resolve():
            guide = directory / "AGENTS.md"
            if guide.is_file() and guide.resolve().is_relative_to(root.resolve()):
                local.add(guide.relative_to(root.resolve()).as_posix())
            directory = directory.parent
    # Follow the indexed targets linked from matching module guides only.
    # Do not recursively load the whole repository or follow external URLs.
    related = set()
    for guide in local:
        for href in LINK.findall((root / guide).read_text(encoding="utf-8")):
            if "://" in href or href.startswith("#"):
                continue
            target = repo_path(root, str(Path(guide).parent / href.split("#")[0]))
            related.add(target.relative_to(root.resolve()).as_posix())
    ranked = []
    for item in items:
        if item["status"].lower() in ("stale", "deprecated", "rejected"):
            continue
        search = " ".join([item["id"], item["summary"], item.get("description", ""), *item["paths"]])
        score = len(words.intersection(TOKEN.findall(search.lower())))
        score += 10 * len(local.intersection(item["paths"]))
        score += 5 * len(related.intersection(item["paths"]))
        if score and item["id"] != "rules.entry":
            ranked.append((score, item))
    ranked.sort(key=lambda row: (-row[0], row[1]["id"]))
    return {
        "status": "OBSERVED", "required": ["AGENTS.md", "dev/README.md", "dev/INDEX.md"],
        "local_guides": sorted(local),
        "assets": items if args.all else [item for _, item in ranked[:args.limit]],
        "note": "Recommendations only; confirm canonical contracts and applicability.",
    }, 0

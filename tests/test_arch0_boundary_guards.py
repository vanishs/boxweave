"""ARCH-0 regression: import direction and standalone helper distribution parity.

These checks deliberately keep the root installer independent. They compare
source declarations without importing privileged install helpers or loading
untrusted install configuration.
"""
import ast
import pathlib
import unittest

ROOT = pathlib.Path(__file__).resolve().parents[1]
BW = ROOT / "bwmini"


def module_source(name):
    return ast.parse((BW / (name + ".py")).read_text(encoding="utf-8"))


def local_imports(module):
    result = set()
    for node in ast.walk(module_source(module)):
        if isinstance(node, ast.ImportFrom):
            if node.level == 1:
                if node.module:
                    result.add(node.module.split(".")[0])
                else:
                    result.update(a.name for a in node.names)
            elif node.module and node.module.startswith("bwmini."):
                result.add(node.module.split(".")[1])
        if isinstance(node, ast.Import):
            for alias in node.names:
                if alias.name.startswith("bwmini."):
                    result.add(alias.name.split(".")[1])
    return result


def literals(path, *, assignment=None, loop_member=None):
    tree = ast.parse(path.read_text(encoding="utf-8"))
    candidates = []
    for node in ast.walk(tree):
        if (assignment and isinstance(node, (ast.Assign, ast.AnnAssign))):
            targets = node.targets if isinstance(node, ast.Assign) else [node.target]
            if any(isinstance(t, ast.Name) and t.id == assignment for t in targets):
                candidates.append(ast.literal_eval(node.value))
        if (loop_member and isinstance(node, (ast.For, ast.AsyncFor))
                and isinstance(node.target, ast.Name) and node.target.id == loop_member):
            try:
                values = ast.literal_eval(node.iter)
            except (ValueError, TypeError):
                continue
            if isinstance(values, tuple) and "mcp_gate.py" in values:
                candidates.append(values)
    if len(candidates) != 1:
        raise AssertionError(f"expected exactly one inventory: {path} {assignment} {loop_member}: {len(candidates)}")
    return tuple(candidates[0])


class BoundaryTests(unittest.TestCase):
    def test_no_static_package_import_cycles(self):
        names = {p.stem for p in BW.glob("*.py")}
        edges = {name: local_imports(name) & names - {name} for name in names}
        active = []
        visited = set()

        def visit(name):
            if name in active:
                self.fail("circular module dependency: " + " -> ".join(active[active.index(name):] + [name]))
            if name in visited:
                return
            active.append(name)
            for child in sorted(edges[name]):
                visit(child)
            active.pop()
            visited.add(name)

        for name in sorted(names):
            visit(name)

    def test_cli_facade_preserves_existing_imports(self):
        from bwmini import __main__, app_runtime
        self.assertIs(__main__.read_config, app_runtime.read_config)
        self.assertIsNot(__main__.load, app_runtime.load)
        self.assertNotIn("__main__", local_imports("app_runtime"))
        self.assertNotIn("__main__", local_imports("deploy_check"))
        self.assertIn("app_runtime", local_imports("deploy_check"))

    def test_root_helper_assets_match_source_release_and_installed_copies(self):
        installer = BW / "online.py"
        publisher = ROOT / "scripts" / "build_release_manifest.py"
        released = set(literals(installer, assignment="RELEASE_FILES"))
        source_copies = set(literals(publisher, loop_member="name"))
        packaged = set(literals(publisher, assignment="names"))
        installed = set(literals(installer, loop_member="name"))
        self.assertEqual(released, packaged)
        self.assertEqual(source_copies | {"boxweave.zip"}, released)
        self.assertEqual(installed, released - {"boxweave.zip", "bootstrap.py", "distribution.py"})
        for filename in released - {"boxweave.zip"}:
            self.assertTrue((BW / filename).is_file(), filename)

        # The deployed gateway must have all standalone sibling imports in
        # the root-owned installed helper set, not just in the application wheel.
        for name in local_imports("mcp_gate"):
            if name not in {"__init__"}:
                self.assertIn(name + ".py", installed)

    def test_no_direct_app_store_import_inside_standalone_gateway(self):
        for name in ("mcp_gate", "coordination_bridge", "coordination_runtime",
                     "operation_coordination", "gateway_tasks"):
            self.assertFalse({"store", "service", "web", "remote_auth"} & local_imports(name),
                             name)


if __name__ == "__main__":
    unittest.main()

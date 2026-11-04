"""Upstream version policy, release trust and no-retry regression.

Synthetic tests only: no daemon, secrets, host upgrades, or downloads.
"""
import json
import tempfile
import unittest
from pathlib import Path
from types import SimpleNamespace
from unittest.mock import patch

from bwmini import upstream_version as u


class UpstreamPolicyTests(unittest.TestCase):
    def test_baseline_is_explicit_and_minimum_only(self):
        self.assertEqual(u.BASELINE, "0.128.0")
        self.assertEqual(u.release(), ("0.128.0", u.PINNED))
        for v in ("0.128.0", "v0.129.0", "1.2.3"):
            self.assertGreaterEqual(u.version_tuple(v), u.version_tuple(u.BASELINE))
        for v in ("v0.127.9", "latest", "v0.129.0-rc1", "../../1.0", "0.129"):
            with self.subTest(v=v), self.assertRaises(u.UpgradeFailure):
                u.release(v)

    def test_latest_resolves_published_verifiable_assets(self):
        version = "0.130.1"
        metadata = {
            "tag_name": "v" + version, "draft": False, "prerelease": False,
            "assets": [{
                "name": n + "-linux-amd64",
                "browser_download_url": u.DOWNLOAD + version + "/" + n + "-linux-amd64",
                "digest": "sha256:" + ("%064x" % (i + 1)),
            } for i, n in enumerate(u.ASSETS)]
        }
        with patch.object(u, "_get_json", return_value=metadata) as get:
            selected, digests = u.release(latest=True)
        self.assertEqual(selected, version)
        self.assertEqual(len(digests), 3)
        get.assert_called_once_with(u.API + "/latest")

    def test_missing_or_forged_digest_refused(self):
        metadata = {"tag_name": "v0.130.0", "draft": False, "prerelease": False,
                    "assets": [{"name": name + "-linux-amd64", "digest": None,
                                "browser_download_url": u.DOWNLOAD + "0.130.0/" + name + "-linux-amd64"}
                               for name in u.ASSETS]}
        with patch.object(u, "_get_json", return_value=metadata), \
             self.assertRaisesRegex(u.UpgradeFailure, "UPSTREAM_ASSET_DIGEST_UNAVAILABLE"):
            u.release("0.130.0")
        metadata["draft"] = True
        with patch.object(u, "_get_json", return_value=metadata), \
             self.assertRaisesRegex(u.UpgradeFailure, "UPSTREAM_RELEASE_NOT_PUBLISHED"):
            u.release("0.130.0")

    def test_native_upgrade_timeout_never_reposts(self):
        with tempfile.TemporaryDirectory() as d:
            calls = []
            def request(url, token, method, path, body=None, timeout=12):
                calls.append((method, path, body))
                raise TimeoutError()
            with patch.object(u, "STATE", Path(d)), \
                 patch.object(u, "_token", return_value="private-not-printed"), \
                 patch.object(u, "_api", side_effect=request), \
                 self.assertRaisesRegex(u.UpgradeFailure, "OUTCOME_UNKNOWN_NO_RETRY"):
                u.native_upgrade("/usr/local/bin/containariumd",
                                 "/etc/containarium/jwt.secret",
                                 "http://127.0.0.1:8080", "0.128.0")
            self.assertEqual(calls, [("POST", "/v1/backends/upgrade",
                                       {"github_tag": "v0.128.0"})])

    def test_synthetic_unchanged_newer_daemon_with_sufficient_mcp(self):
        with patch.object(u.os, "geteuid", return_value=0), \
             patch.object(u, "installed_version", return_value="0.200.0"), \
             patch.object(u, "mcp_version", return_value="0.130.0"), \
             patch.object(u.Path, "exists", return_value=True), \
             patch.object(u, "trusted", return_value=True), \
             patch.object(u, "download_assets") as download:
            status = u.ensure({}, cli="/usr/local/bin/containariumd",
                              url="http://127.0.0.1:8080",
                              secret="/etc/containarium/jwt.secret")
            self.assertEqual(status["status"], "kept_compatible")
            download.assert_not_called()

    def test_stdio_mcp_initialize_is_only_version_probe(self):
        message = {"jsonrpc": "2.0", "id": 1, "result": {
            "serverInfo": {"name": "containarium-mcp-server", "version": "0.131.2"},
            "protocolVersion": "2024-11-05"}}
        with tempfile.TemporaryDirectory() as d:
            with patch.object(u, "STATE", Path(d)), \
                 patch.object(u, "trusted", return_value=True), \
                 patch.object(u, "_token", return_value="private-not-printed"), \
                 patch.object(u.subprocess, "run", return_value=SimpleNamespace(
                     returncode=0, stdout=json.dumps(message) + "\n", stderr=""
                 )) as run:
                result = u.mcp_version("/usr/local/bin/mcp-server",
                                       "/usr/local/bin/containariumd",
                                       "http://127.0.0.1:8080",
                                       "/etc/containarium/jwt.secret")
                self.assertEqual(result, "0.131.2")
                args, kwargs = run.call_args
                self.assertEqual(args[0], ["/usr/local/bin/mcp-server"])
                self.assertTrue(kwargs["input"].endswith("\n"))
                self.assertIn('"method": "initialize"', kwargs["input"])
                self.assertEqual(kwargs["env"]["CONTAINARIUM_SERVER_URL"],
                                 "http://127.0.0.1:8080")
                self.assertNotIn("private-not-printed", str(kwargs["env"]))

    def test_downgrade_refused_before_network_or_write(self):
        with patch.object(u.os, "geteuid", return_value=0), \
             patch.object(u, "installed_version", return_value="0.140.0"), \
             patch.object(u, "download_assets") as download, \
             self.assertRaisesRegex(u.UpgradeFailure, "UPSTREAM_DOWNGRADE_REFUSED"):
            u.ensure({}, cli="/usr/local/bin/containariumd",
                     url="http://127.0.0.1:8080",
                     secret="/etc/containarium/jwt.secret",
                     target="0.128.0")
        download.assert_not_called()

    def test_old_external_daemon_uses_one_host_upgrade_and_updates_clients(self):
        with tempfile.TemporaryDirectory() as d:
            calls = []
            def installed(path):
                return "0.120.1" if str(path).endswith("containariumd") else "0.120.1"
            with patch.object(u.os, "geteuid", return_value=0), \
                 patch.object(u, "STATE", Path(d)), \
                 patch.object(u, "installed_version", side_effect=installed), \
                 patch.object(u, "mcp_version", return_value="0.120.1"), \
                 patch.object(u.Path, "exists", return_value=True), \
                 patch.object(u, "trusted", return_value=True), \
                 patch.object(u, "download_assets") as download, \
                 patch.object(u, "external_host_upgrade", return_value={
                     "status": "running_verified", "version": "0.128.0", "upgrade_id": "one"
                 }) as host_upgrade, \
                 patch.object(u, "refresh_clients") as clients:
                state = {}
                result = u.ensure(state, cli="/usr/local/bin/containariumd",
                                  url="http://127.0.0.1:8080",
                                  secret="/etc/containarium/jwt.secret")
                self.assertEqual(result["upgrade_id"], "one")
                self.assertEqual(result["baseline"], "0.128.0")
                self.assertEqual(state["upstream_version_observed"], "0.128.0")
                download.assert_called_once()
                host_upgrade.assert_called_once()
                args = host_upgrade.call_args.args
                self.assertTrue(Path(args[0]).name.startswith("upstream-stage-"))
                self.assertEqual(args[1:], ("/usr/local/bin/containariumd",
                                            "/etc/containarium/jwt.secret",
                                            "http://127.0.0.1:8080",
                                            "0.120.1", "0.128.0"))
                clients.assert_called_once()

    def test_native_upgrade_failed_job_is_reported_not_timed_out(self):
        with tempfile.TemporaryDirectory() as d:
            calls = []
            def request(url, token, method, path, body=None, timeout=12):
                calls.append((method, path))
                if method == "POST":
                    return {"upgradeId": "upg-12345"}
                if path == "/v1/system/info":
                    return {"info": {"daemonVersion": "0.120.1"}}
                if path == "/v1/upgrades/upg-12345":
                    return {"status": "failed", "error": "private upstream error"}
                raise AssertionError((method, path))
            with patch.object(u, "STATE", Path(d)), \
                 patch.object(u, "_token", return_value="private-not-printed"), \
                 patch.object(u, "_api", side_effect=request), \
                 patch.object(u.time, "sleep"), \
                 self.assertRaisesRegex(u.UpgradeFailure, "UPSTREAM_UPGRADE_JOB_FAILED"):
                u.native_upgrade("/usr/local/bin/containariumd",
                                 "/etc/containarium/jwt.secret",
                                 "http://127.0.0.1:8080", "0.128.0")
            self.assertEqual(calls, [
                ("POST", "/v1/backends/upgrade"),
                ("GET", "/v1/system/info"),
                ("GET", "/v1/upgrades/upg-12345")])
            receipt = json.loads((Path(d) / "upstream-original-upgrade.json").read_text())
            self.assertEqual(receipt["upgrade_id"], "upg-12345")
            self.assertNotIn("private", json.dumps(receipt))

    def test_external_host_upgrade_refuses_unknown_service_before_write(self):
        fake = SimpleNamespace(returncode=0, stdout=(
            "LoadState=loaded\nActiveState=active\nProtectSystem=strict\n"
            "ExecStart={ path=/usr/bin/unrelated ; argv[]=/usr/bin/unrelated }\n"))
        with patch.object(u, "trusted", return_value=True), \
             patch.object(u.subprocess, "run", return_value=fake), \
             patch.object(u, "_atomic_binary") as swap, \
             patch.object(u, "_token") as token:
            with self.assertRaisesRegex(u.UpgradeFailure, "UPSTREAM_EXTERNAL_UNIT_REVIEW_REQUIRED"):
                u.external_host_upgrade(Path("/stage"), "/usr/local/bin/containariumd",
                                        "/etc/containarium/jwt.secret", "http://127.0.0.1:8080",
                                        "0.120.1", "0.128.0")
            swap.assert_not_called()
            token.assert_not_called()

    def test_native_upgrade_denial_is_definite_without_replay(self):
        from urllib.error import HTTPError
        with tempfile.TemporaryDirectory() as d:
            calls = []
            def refuse(url, token, method, path, body=None, timeout=12):
                calls.append(method)
                raise HTTPError(url, 403, "Forbidden", {}, None)
            with patch.object(u, "STATE", Path(d)), \
                 patch.object(u, "_token", return_value="private-not-printed"), \
                 patch.object(u, "_api", side_effect=refuse), \
                 self.assertRaisesRegex(u.UpgradeFailure, "REJECTED_HTTP_403"):
                u.native_upgrade("/usr/local/bin/containariumd",
                                 "/etc/containarium/jwt.secret",
                                 "http://127.0.0.1:8080", "0.128.0")
            self.assertEqual(calls, ["POST"])

    def test_unprivileged_status_and_upgrade_refused(self):
        with patch.object(u.os, "geteuid", return_value=1000):
            self.assertEqual(u.main(["status"]), 2)
            self.assertEqual(u.main(["update", "--latest"]), 2)


if __name__ == "__main__":
    unittest.main()

"""Web deployment settings, unit isolation and staging contract tests."""

import importlib.util
import json
from pathlib import Path
import subprocess
import socket
import sys
import tempfile
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[1]
spec = importlib.util.spec_from_file_location(
    "web_install", ROOT / "server/web_install.py"
)
web = importlib.util.module_from_spec(spec)
spec.loader.exec_module(web)


class WebInstallTests(unittest.TestCase):
    def test_default_https_and_precedence(self):
        d = web.settings(host="192.0.2.2")
        self.assertEqual(d["mode"], "https")
        self.assertEqual(d["port"], 18443)
        saved = {**d, "mode": "http", "port": 19443}
        self.assertEqual(web.settings(old=saved)["mode"], "http")
        self.assertEqual(
            web.settings(mode="https", port=20443, old=saved)["port"], 20443
        )

    def test_invalid_ports_and_hosts(self):
        for kw in [
            {"port": 80},
            {"port": 18444},
            {"host": "0.0.0.0"},
            {"host": "8.8.8.8"},
            {"host": "evil.test"},
            {"mode": "ftp"},
        ]:
            with self.assertRaises(ValueError):
                web.settings(**{"host": "192.0.2.2", **kw})

    def test_dedicated_caddy_no_admin_no_redirect_fallback(self):
        d = web.settings(host="192.0.2.2")
        s = web.caddyfile(d)
        self.assertIn("tls internal", s)
        self.assertIn("admin off", s)
        self.assertIn("bind 192.0.2.2", s)
        d["mode"] = "http"
        d["origin"] = "http://192.0.2.2:18443"
        self.assertNotIn("tls internal", web.caddyfile(d))

    def test_units_keep_credentials_and_mcp_independent(self):
        d = web.settings(host="192.0.2.2")
        d["release"] = "/opt/boxweave-web/releases/test"
        u = web.units(d)
        self.assertIn("User=boxweave-web\n", u["boxweave-web.service"])
        self.assertIn("ProtectHome=true", u["boxweave-web.service"])
        self.assertIn(
            "InaccessiblePaths=-/etc/containarium -/etc/boxweave/config.json",
            u["boxweave-web.service"],
        )
        self.assertNotIn("Requires=", u["boxweave-web.service"])
        self.assertNotIn("boxweave-mcp", str(u))
        self.assertIn("OnUnitInactiveSec=1h", u["boxweave-web-token.timer"])

    def test_atomic_refuses_symlinks(self):
        with tempfile.TemporaryDirectory() as temp:
            p = Path(temp) / "file"
            web.atomic(p, b"first", 0o600)
            web.atomic(p, b"second", 0o600)
            self.assertEqual(p.read_bytes(), b"second")
            self.assertEqual(p.stat().st_mode & 0o777, 0o600)
            link = Path(temp) / "link"
            link.symlink_to(p)
            with self.assertRaises(RuntimeError):
                web.atomic(link, b"bad")

    def test_installer_web_is_mandatory_and_mcp_resumes_before_web_publish(self):
        s = (ROOT / "install.sh").read_text()
        self.assertIn('web_install.py" prepare', s)
        self.assertIn('web_install.py" publish', s)
        self.assertLess(
            s.index('web_install.py" prepare'), s.index("publish_components\n")
        )
        start = s.index("INSTALL_PHASE='mandatory independent web installation'")
        self.assertLess(s.index("systemctl restart boxweave-token.timer"), start)
        self.assertIn("TIMERS_PAUSED=0", s[start - 30 : start])
        self.assertNotIn("--skip-web", s)


    def test_apt_retries_only_on_lock_contention(self):
        blocked = subprocess.CompletedProcess(
            ["apt-get"], 100, "", "E: Could not get lock /var/lib/apt/lists/lock"
        )
        success = subprocess.CompletedProcess(["apt-get"], 0, "ok\n", "")
        with patch.object(web.subprocess, "run", side_effect=[blocked, success]) as runner:
            with patch.object(web.time, "sleep") as sleeper:
                web.run_apt(["update"], timeout=180)
        self.assertEqual(runner.call_count, 2)
        self.assertEqual(
            runner.call_args.args[0][:3],
            ["apt-get", "-o", "DPkg::Lock::Timeout=300"],
        )
        sleeper.assert_called_once()

    def test_apt_does_not_retry_other_package_errors(self):
        failure = subprocess.CompletedProcess(
            ["apt-get"], 100, "", "E: Unable to locate package missing"
        )
        with patch.object(web.subprocess, "run", return_value=failure) as runner:
            with patch.object(web.time, "sleep") as sleeper:
                with self.assertRaises(subprocess.CalledProcessError):
                    web.run_apt(["install", "missing"], timeout=60)
        runner.assert_called_once()
        sleeper.assert_not_called()

    def test_prepare_uses_bounded_apt_wait_for_missing_prerequisites(self):
        class StopAfterApt(Exception):
            pass

        with patch.object(web.shutil, "which", return_value=None), patch.object(
            web.subprocess, "run", return_value=subprocess.CompletedProcess([], 1)
        ), patch.object(web, "run_apt") as apt, patch.object(
            web, "run", side_effect=StopAfterApt
        ):
            with self.assertRaises(StopAfterApt):
                web.prepare(Path("/unused"), {})
        self.assertEqual(apt.call_args_list[0].args, (["update"],))
        self.assertEqual(apt.call_args_list[0].kwargs, {"timeout": 180})
        self.assertEqual(apt.call_args_list[1].kwargs, {"timeout": 660})

if __name__ == "__main__":
    unittest.main()

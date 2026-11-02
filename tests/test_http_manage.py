"""HTTP MCP quick-start management fixtures: no live host or ports."""
import json
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

from bwmini import http_manage
from bwmini.distribution import encoded


class HttpManageTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.prefix = Path(self.temp.name)
        self.db = self.prefix / "state.sqlite"
        self.config = {
            "mode": "mcp", "host": "127.0.0.1", "port": 18640,
            "origin": "http://127.0.0.1:18640", "secure_cookie": False,
            "database": str(self.db), "launcher": ["/usr/bin/ssh", "fixture"]
        }
        (self.prefix / "config.json").write_bytes(encoded(self.config))
        (self.prefix / "runtime.json").write_bytes(
            encoded(self.config | {"remote_mcp": {"enabled": True, "port": 18844,
                                                 "allowed_origins": []}}))
        self.name = http_manage.unit_name(self.prefix)
        (self.prefix / "installed.json").write_bytes(encoded({"units": [self.name]}))
        (self.prefix / ".install.lock").touch()

    def tearDown(self):
        self.temp.cleanup()

    def test_local_default_status_does_not_report_external_verified(self):
        with patch.object(http_manage.urllib.request, "build_opener",
                          side_effect=OSError("fixture connection refused")):
            result = http_manage.observation(self.prefix)
        self.assertFalse(result["local_discovery"])
        self.assertFalse(result["external_verified"])
        self.assertEqual(result["local_url"], "http://127.0.0.1:18844/mcp")

    def test_configure_public_url_updates_both_private_configs_and_restart(self):
        with patch.object(http_manage, "user_control") as control, \
             patch.object(http_manage, "observation", return_value={"ok": True}):
            http_manage.configure(self.prefix, "https://public.example.test")
        control.assert_called_once_with(self.prefix, "restart")
        for filename in ("config.json", "runtime.json"):
            data = json.loads((self.prefix / filename).read_text())
            self.assertEqual(data["remote_mcp"]["public_url"],
                             "https://public.example.test")
            self.assertEqual(data["remote_mcp"]["port"], 18844)

    def test_failed_restart_rolls_back_both_config_files(self):
        before = [(self.prefix / name).read_bytes() for name in
                  ("config.json", "runtime.json")]
        with patch.object(http_manage, "user_control",
                          side_effect=ValueError("restart failed")):
            with self.assertRaises(ValueError):
                http_manage.configure(self.prefix, "https://public.example.test")
        self.assertEqual(before, [(self.prefix / name).read_bytes() for name in
                                  ("config.json", "runtime.json")])

    def test_insecure_public_url_and_update_marker_refused(self):
        with self.assertRaises(ValueError):
            http_manage.configure(self.prefix, "http://public.example.test")
        (self.prefix / "update-in-progress.json").touch()
        with self.assertRaisesRegex(ValueError, "INSTALL_IN_PROGRESS"):
            http_manage.configure(self.prefix, "https://public.example.test")

    def test_disable_is_explicit_and_not_overridden(self):
        config = self.config | {"remote_mcp": {"enabled": False}}
        (self.prefix / "config.json").write_bytes(encoded(config))
        with self.assertRaisesRegex(ValueError, "REMOTE_EXPLICITLY_DISABLED"):
            http_manage.configure(self.prefix, "https://public.example.test")

    def test_managed_service_membership_required(self):
        (self.prefix / "installed.json").write_bytes(encoded({"units": []}))
        with self.assertRaisesRegex(ValueError, "REMOTE_SERVICE_NOT_MANAGED"):
            http_manage.user_control(self.prefix, "restart")


if __name__ == "__main__":
    unittest.main()

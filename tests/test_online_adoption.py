"""Existing Provider detection and read-only acceptance contract.

These tests never start a Provider, mutate Incus, or read a real signing secret.
"""
import json
from pathlib import Path
from types import SimpleNamespace
from unittest.mock import patch
import unittest

from bwmini import online


class ExistingProviderTests(unittest.TestCase):
    def tools(self, present):
        return patch.object(online.shutil, "which",
                            side_effect=lambda name: "/usr/local/bin/" + name if name in present else None)

    def test_incus_only_is_not_mistaken_for_containarium(self):
        with self.tools({"incus"}):
            self.assertIsNone(online.existing_provider_options())
            with self.assertRaisesRegex(online.Failure, "PROVIDER_OPTIONS_REQUIRE_EXISTING_CONTAINARIUM"):
                online.existing_provider_options(provider_url="http://127.0.0.1:8080")

    def test_complete_provider_keeps_installed_identity(self):
        with self.tools({"incus", "containarium", "containariumd"}):
            provider = online.existing_provider_options()
            self.assertEqual(provider, {
                "url": "http://127.0.0.1:8080",
                "secret": "/etc/containarium/jwt.secret",
                "cli": "/usr/local/bin/containariumd"})
            self.assertEqual(online.existing_provider_options(
                provider_url="http://127.0.0.1:18080",
                provider_secret_file="/etc/approved-provider.jwt",
                provider_cli="/opt/provider/containariumd"), {
                    "url": "http://127.0.0.1:18080",
                    "secret": "/etc/approved-provider.jwt",
                    "cli": "/opt/provider/containariumd"})

    def test_explicit_cli_outside_path_is_accepted_for_discovery(self):
        with self.tools({"incus", "containarium"}):
            selected = online.existing_provider_options(provider_cli="/opt/approved/containariumd")
            self.assertEqual(selected["cli"], "/opt/approved/containariumd")

    def test_incomplete_provider_is_never_installed_over(self):
        for binaries in ({"containarium"}, {"incus", "containarium"},
                         {"incus", "containariumd"}):
            with self.subTest(binaries=binaries), self.tools(binaries):
                with self.assertRaisesRegex(online.Failure, "PARTIAL_EXISTING_PROVIDER"):
                    online.existing_provider_options()

    def test_provider_endpoint_must_be_exact_loopback(self):
        with self.tools({"incus", "containarium", "containariumd"}):
            for url in ("http://localhost:8080", "http://192.168.1.1:8080",
                        "https://127.0.0.1:8080", "http://127.0.0.1:8080/admin",
                        "http://127.0.0.1:8080?secret=abc"):
                with self.subTest(url=url), self.assertRaises(online.Failure):
                    online.existing_provider_options(provider_url=url)

    def test_provider_probe_is_read_only_and_requires_storage(self):
        provider = {"url": "http://127.0.0.1:8080",
                    "secret": "/etc/containarium/jwt.secret",
                    "cli": "/usr/local/bin/containariumd"}
        readonly = SimpleNamespace(st_uid=0, st_mode=0o100600)
        response = SimpleNamespace(status=200)
        response.__enter__ = None
        class HTTPResponse:
            status = 200
            def __enter__(self): return self
            def __exit__(self, *args): return False
        with patch.object(Path, "is_symlink", return_value=False), \
             patch.object(Path, "is_file", return_value=True), \
             patch.object(Path, "stat", return_value=readonly), \
             patch.object(online.os, "access", return_value=True), \
             self.tools({"incus", "containarium", "containariumd"}), \
             patch.object(online.subprocess, "run", return_value=SimpleNamespace(
                 returncode=0, stdout=json.dumps([{"name": "default", "driver": "dir"}]))) as execute, \
             patch.object(online.urllib.request, "urlopen", return_value=HTTPResponse()) as http, \
             patch.object(online, "atomic") as write, \
             patch.object(online, "run") as managed_run:
            self.assertEqual(online.verify_existing_provider(provider), provider)
            execute.assert_called_once()
            self.assertEqual(execute.call_args.args[0],
                             ["incus", "--force-local", "storage", "list", "--format", "json"])
            self.assertEqual(http.call_args.args[0], "http://127.0.0.1:8080/v1/containers")
            write.assert_not_called()
            managed_run.assert_not_called()

    def test_uninitialized_incus_refused_without_writing(self):
        provider = {"url": "http://127.0.0.1:8080",
                    "secret": "/etc/containarium/jwt.secret",
                    "cli": "/usr/local/bin/containariumd"}
        readonly = SimpleNamespace(st_uid=0, st_mode=0o100600)
        with patch.object(Path, "is_symlink", return_value=False), \
             patch.object(Path, "is_file", return_value=True), \
             patch.object(Path, "stat", return_value=readonly), \
             patch.object(online.os, "access", return_value=True), \
             self.tools({"incus", "containarium", "containariumd"}), \
             patch.object(online.subprocess, "run", return_value=SimpleNamespace(
                 returncode=0, stdout="[]")), \
             patch.object(online, "atomic") as write:
            with self.assertRaisesRegex(online.Failure, "EXISTING_INCUS_STORAGE_NOT_READY"):
                online.verify_existing_provider(provider)
            write.assert_not_called()


if __name__ == "__main__":
    unittest.main()

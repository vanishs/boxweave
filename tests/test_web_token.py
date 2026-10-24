"""Independent token scope upgrade is capability-gated and never follows redirects."""

import base64
import importlib.util
import io
import json
import os
from pathlib import Path
import subprocess
import tempfile
import time
from types import SimpleNamespace
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[1]
spec = importlib.util.spec_from_file_location("web_token", ROOT / "server/web_token.py")
issuer = importlib.util.module_from_spec(spec)
spec.loader.exec_module(issuer)


def synthetic(scopes):
    enc = (
        lambda obj: base64.urlsafe_b64encode(json.dumps(obj).encode())
        .decode()
        .rstrip("=")
    )
    return (
        enc({"alg": "none"})
        + "."
        + enc(
            {
                "username": "boxweave-web",
                "roles": ["admin"],
                "scopes": sorted(scopes),
                "exp": time.time() + 7200,
            }
        )
        + ".synthetic-fixture"
    )


class WebTokenTests(unittest.TestCase):
    def test_old_provider_keeps_old_scopes_and_new_contract_adds_only_policy_scopes(
        self,
    ):
        for available in (False, True):
            with self.subTest(
                available=available
            ), tempfile.TemporaryDirectory() as temp:
                root = Path(temp)
                auth = root / "auth"
                auth.mkdir()
                config = root / "config.json"
                config.write_text(
                    json.dumps(
                        {
                            "containarium_daemon_binary": "/fixture/provider",
                            "jwt_secret_file": "/fixture/not-read",
                            "api_url": "http://127.0.0.1:8080",
                        }
                    )
                )

                def redirected(value):
                    return (
                        config
                        if str(value) == "/etc/boxweave/config.json"
                        else (
                            auth / "provider.jwt"
                            if str(value) == "/var/lib/boxweave-web-auth/provider.jwt"
                            else Path(value)
                        )
                    )

                def run(args, **kwargs):
                    scopes = set(args[args.index("--scopes") + 1].split(","))
                    return subprocess.CompletedProcess(args, 0, synthetic(scopes), "")

                with patch.object(issuer.os, "geteuid", return_value=0), patch.object(
                    issuer, "Path", side_effect=redirected
                ), patch.object(
                    issuer.pwd,
                    "getpwnam",
                    return_value=SimpleNamespace(
                        pw_uid=os.getuid(), pw_gid=os.getgid()
                    ),
                ), patch.object(
                    issuer.os, "chown"
                ), patch.object(
                    issuer.subprocess, "run", side_effect=run
                ) as mint, patch.object(
                    issuer, "network_contract_available", return_value=available
                ), patch(
                    "sys.stdout", new=io.StringIO()
                ):
                    issuer.main()
                self.assertEqual(mint.call_count, 2 if available else 1)
                expected = issuer.BASE_SCOPES | (
                    issuer.NETWORK_SCOPES if available else set()
                )
                last = mint.call_args.args[0]
                self.assertEqual(
                    set(last[last.index("--scopes") + 1].split(",")), expected
                )
                self.assertEqual((auth / "provider.jwt").stat().st_mode & 0o777, 0o600)

    def test_capability_probe_is_fixed_loopback_only_and_bounded(self):
        with patch.object(issuer, "build_opener") as opener:
            for url in (
                "http://192.0.2.2:8080",
                "https://external.test",
                "http://user:pass@127.0.0.1:8080",
                "http://127.0.0.1:8080/other",
            ):
                self.assertFalse(issuer.network_contract_available(url, "synthetic"))
            opener.assert_not_called()
        response = SimpleNamespace(
            read=lambda n: json.dumps(
                {
                    "contract": "boxweave.tenant-network/v1",
                    "atomicTenantAccess": True,
                    "backendScope": "local-lxc",
                }
            ).encode()
        )

        class Context:
            def __enter__(self):
                return response

            def __exit__(self, *args):
                pass

        with patch.object(issuer, "build_opener") as opener:
            opener.return_value.open.return_value = Context()
            self.assertTrue(
                issuer.network_contract_available("http://127.0.0.1:8080", "synthetic")
            )
            request = opener.return_value.open.call_args.args[0]
            self.assertEqual(
                request.full_url,
                "http://127.0.0.1:8080/v1/network-tenants/capabilities",
            )
        self.assertIsNone(
            issuer.NoRedirect().redirect_request(None, None, None, None, None, None)
        )

    def test_unrecognized_or_malformed_contract_never_expands_authority(self):
        class Response:
            def __init__(self, value):
                self.value = value

            def __enter__(self):
                return self

            def __exit__(self, *args):
                pass

            def read(self, n):
                return json.dumps(self.value).encode()

        for data in (
            [],
            {},
            {
                "contract": "unrecognized",
                "atomicTenantAccess": True,
                "backendScope": "local-lxc",
            },
        ):
            with patch.object(issuer, "build_opener") as opener:
                opener.return_value.open.return_value = Response(data)
                self.assertFalse(
                    issuer.network_contract_available(
                        "http://127.0.0.1:8080", "synthetic"
                    )
                )


if __name__ == "__main__":
    unittest.main()

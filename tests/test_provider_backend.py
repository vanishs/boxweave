"""Authenticated local placement metadata; no live infrastructure."""

from pathlib import Path
import sys

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "server"))
import unittest
from unittest.mock import Mock
from webui.provider import Provider, ProviderError


class BackendIdentityTests(unittest.TestCase):
    def test_local_backend_requires_unique_provider_metadata(self):
        for backends, expected in [
            ([{"id": "dev", "type": "local"}], ["dev"]),
            ([{"id": "dev", "type": "tunnel"}], []),
            ([{"id": "dev", "type": "local"}, {"id": "dev", "type": "tunnel"}], []),
            ([{"id": "dev", "type": "future"}], []),
            ({"id": "dev", "type": "local"}, []),
            ([None], []),
            ([{"id": None, "type": "local"}], []),
        ]:
            p = object.__new__(Provider)
            p.request = Mock(
                side_effect=[
                    {"containers": [{"name": "a-container", "backendId": "dev"}]},
                    {"backends": backends},
                ]
            )
            self.assertEqual(p.boxes()["localBackendIds"], expected)
            self.assertEqual(p.request.call_args.args, ("GET", "/v1/backends"))
        p = object.__new__(Provider)
        p.request = Mock(side_effect=[{"containers": []}, ProviderError(403)])
        self.assertEqual(p.boxes()["localBackendIds"], [])

"""Ephemeral UI test server; no real provider or credentials."""

import os
from pathlib import Path
import sys
import tempfile

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "server"))
from webui.app import create_app, HASHER
from waitress import serve
from test_web import FakeProvider

root = Path(__file__).resolve().parents[1]
temp = tempfile.TemporaryDirectory(prefix="boxweave-browser-")
port = int(os.environ.get("BOXWEAVE_FIXTURE_PORT", "18555"))
native = os.environ.get("BOXWEAVE_NATIVE_FIXTURE") == "1"
if native:
    from test_tenant_network import NativeProvider

    provider = NativeProvider()
else:
    provider = FakeProvider()
provider.data = {
    n: {
        "username": n,
        "name": n + "-container",
        "state": "CONTAINER_STATE_RUNNING",
        "resources": {"cpu": "2", "memory": "2GB", "disk": "10GB"},
        "network": {"ipAddress": "10.0.0.2"},
    }
    for n in ("alpha", "beta")
}
if native:
    provider.data = {
        n: {
            "username": n,
            "name": n + "-container",
            "state": "CONTAINER_STATE_RUNNING",
            "network": {"ipAddress": ip},
            "resources": {"cpu": "2", "memory": "2GB", "disk": "10GB"},
            "networkTenant": key,
            "networkTenantSource": "native",
        }
        for n, key, ip in [
            ("dev-existing", "allinone", "10.0.0.2"),
            ("test-existing", "testing", "10.0.0.3"),
        ]
    }
    provider.policies["testing"] = {
        "tenant": "testing",
        "mode": "NETWORK_POLICY_MODE_ENFORCE",
        "egressCidrs": ["10.0.0.0/8"],
        "egressDomains": ["example.com"],
        "allowMetadata": False,
        "egressPreset": "EGRESS_PRESET_ALLOW_LIST_ONLY",
        "denyRules": [
            {"cidr": "192.0.2.0/24", "note": "preserve this independent deny"}
        ],
    }
app = create_app(
    {
        "database": temp.name + "/data.db",
        "origin": f"http://127.0.0.1:{port}",
        "static": str(root / "web/dist"),
    },
    provider,
)
with app.extensions["store"].db() as db:
    db.execute(
        "INSERT INTO admin VALUES (1,?)", (HASHER.hash("browser test password only"),)
    )
serve(app, host="127.0.0.1", port=port, threads=4)

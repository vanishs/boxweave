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
app = create_app(
    {
        "database": temp.name + "/data.db",
        "origin": "http://127.0.0.1:18555",
        "static": str(root / "web/dist"),
    },
    provider,
)
with app.extensions["store"].db() as db:
    db.execute(
        "INSERT INTO admin VALUES (1,?)", (HASHER.hash("browser test password only"),)
    )
serve(app, host="127.0.0.1", port=18555, threads=4)

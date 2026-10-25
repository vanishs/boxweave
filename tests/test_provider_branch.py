"""Optional cross-repository REST contract test. No real Incus/ACL/TCP testing."""

import json
import os
from pathlib import Path
import selectors
import subprocess
import tempfile
import unittest

import test_web as web
from webui.provider import Provider
from webui.tenants import tenant_id

ROOT = Path(__file__).resolve().parents[1]


@unittest.skipUnless(
    os.environ.get("BOXWEAVE_PROVIDER_SOURCE"),
    "requires explicit pinned Provider source",
)
class ProviderBranchContractTests(unittest.TestCase):
    req = web.WebTests.req
    login = web.WebTests.login

    @classmethod
    def setUpClass(cls):
        web.WebTests.setUpClass.__func__(cls)
        cls.source = Path(os.environ["BOXWEAVE_PROVIDER_SOURCE"]).resolve()
        profile = json.loads(
            (ROOT / "patches/containarium/native-network-tenant.json").read_text()
        )
        head = subprocess.check_output(
            ["git", "rev-parse", "HEAD"], cwd=cls.source, text=True
        ).strip()
        if head != profile["commit"]:
            raise RuntimeError("Provider source does not match reviewed profile")
        if subprocess.check_output(["git", "status", "--porcelain"], cwd=cls.source):
            raise RuntimeError("Provider fixture requires a clean dedicated checkout")
        cls.temp_build = tempfile.TemporaryDirectory()
        cls.addClassCleanup(cls.temp_build.cleanup)
        helper = cls.source / "internal/server/zz_boxweave_contract_fixture_test.go"
        if helper.exists():
            raise RuntimeError("Refusing to overwrite an existing fixture")
        try:
            helper.write_bytes(
                (ROOT / "tests/provider_contract_fixture.go").read_bytes()
            )
            subprocess.run(["gofmt", "-w", str(helper)], check=True, timeout=30)
            cls.binary = str(Path(cls.temp_build.name) / "provider-contract")
            subprocess.run(
                ["go", "test", "-p", "2", "-c", "-o", cls.binary, "./internal/server"],
                cwd=cls.source,
                check=True,
                timeout=600,
            )
        finally:
            helper.unlink(missing_ok=True)

    def setUp(self):
        web.WebTests.setUp(self)
        self.app.extensions["batch_pool"].shutdown(True)
        self.fixture = subprocess.Popen(
            [
                self.binary,
                "-test.run=^TestBoxWeaveContractFixture$",
                "-test.timeout=60s",
            ],
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
            env={**os.environ, "BOXWEAVE_CONTRACT_FIXTURE": "1"},
        )
        self.addCleanup(self.stop_fixture)
        selector = selectors.DefaultSelector()
        selector.register(self.fixture.stdout, selectors.EVENT_READ)
        try:
            if not selector.select(20):
                raise RuntimeError("Provider fixture readiness timeout")
            origin = self.fixture.stdout.readline().strip()
        finally:
            selector.close()
        if not origin.startswith("http://127.0.0.1:"):
            raise RuntimeError("Unexpected fixture listener: " + origin)
        token = Path(self.temp.name) / "contract-token"
        token.write_text("synthetic-contract-only")
        token.chmod(0o600)
        self.provider = Provider(
            {
                "api_url": origin,
                "token_file": str(token),
                "public_key": "ssh-ed25519 AAAA fixture",
            }
        )
        self.app = web.create_app(self.cfg, self.provider)
        self.app.testing = True
        self.addCleanup(self.app.extensions["batch_pool"].shutdown, True)
        self.c = self.app.test_client()
        self.store = self.app.extensions["store"]
        self.login()

    def stop_fixture(self):
        if self.fixture.poll() is None:
            self.fixture.stdin.close()
            self.fixture.wait(timeout=10)
        stderr = self.fixture.stderr.read()
        self.fixture.stdout.close()
        self.fixture.stderr.close()
        self.assertEqual(self.fixture.returncode, 0, stderr)

    def plan(self, direction="forward", operation="grant"):
        response = self.req(
            "tenant-access/plans",
            "POST",
            {
                "source": tenant_id("allinone"),
                "destination": tenant_id("testing"),
                "direction": direction,
                "operation": operation,
            },
        )
        self.assertEqual(response.status_code, 201, response.json)
        return response.json

    def confirm(self, plan):
        response = self.req(
            "tenant-access/plans/" + plan["id"] + "/confirm", "POST", {"confirm": True}
        )
        self.assertEqual(response.status_code, 200, response.json)
        return response.json

    def test_actual_generated_rest_revisions_states_and_policy_preservation(self):
        snapshot = self.req("tenants").json
        self.assertTrue(snapshot["capabilities"]["sharedTenantCreate"], snapshot)
        self.assertTrue(snapshot["capabilities"]["networkPolicyWrite"], snapshot)
        testing = next(t for t in snapshot["tenants"] if t["providerKey"] == "testing")
        self.assertEqual(testing["membershipStatus"], "confirmed")
        self.assertEqual(testing["guardStatus"], "verified")
        original = self.provider.network_policy("testing")
        self.assertEqual(
            original["policy"]["egressMode"], "NETWORK_POLICY_MODE_LOG_ONLY"
        )
        plan = self.plan("both")
        receipt = self.confirm(plan)
        self.assertEqual(receipt["items"][0]["status"], "configured", receipt)
        self.assertEqual(receipt["result"]["applicationState"], "pending")
        self.assertFalse(receipt["connectivityTested"])
        observed = self.provider.network_policy("testing")
        self.assertNotEqual(original["revision"], observed["revision"])
        self.assertEqual(
            observed["policy"], {**original["policy"], "allowFromTenants": ["allinone"]}
        )
        self.assertEqual(
            self.provider.network_policy("allinone")["policy"]["allowFromTenants"],
            ["testing"],
        )
        stale = self.plan()
        current = self.provider.network_policy("testing")
        self.provider.patch_tenant_access(
            [
                {
                    "destination": "testing",
                    "addSources": ["fixture-other"],
                    "expectedRevision": current["revision"],
                }
            ]
        )
        self.assertEqual(self.confirm(stale)["items"][0]["status"], "conflict")
        self.assertEqual(
            self.confirm(self.plan("both", "revoke"))["items"][0]["status"],
            "configured",
        )
        self.assertEqual(
            self.provider.network_policy("testing")["policy"]["allowFromTenants"],
            ["fixture-other"],
        )

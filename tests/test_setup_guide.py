"""Phase 2 R-03 public read-only setup guide contract."""
import io
import json
import unittest
from unittest.mock import patch

from bwmini import host_cli

PUBLIC = {"schema_version": 1, "phase":"ready", "version":"0.30.0",
          "application_user":"boxweave", "app_prefix":"/home/boxweave/boxweave",
          "web_port":18640, "host":"10.0.0.5", "port":22}

class SetupGuideTests(unittest.TestCase):
    def test_http_default_does_not_claim_external_connection(self):
        with patch.object(host_cli,"public_state",return_value=PUBLIC), \
             patch("sys.stdout",new_callable=io.StringIO) as out:
            self.assertEqual(host_cli.main(["setup","--format","json"]),0)
        result=json.loads(out.getvalue())
        self.assertEqual(result["mode"],"https")
        self.assertEqual(result["external_reachability"],"not_verified")
        self.assertEqual(result["authenticated_mcp"],"not_verified")
        self.assertNotIn("https://",json.dumps(result))

    def test_ssh_exact_published_server(self):
        with patch.object(host_cli,"public_state",return_value=PUBLIC), \
             patch("sys.stdout",new_callable=io.StringIO) as out:
            self.assertEqual(host_cli.main(["setup","--mode","ssh","--format","json"]),0)
        result=json.loads(out.getvalue())
        self.assertEqual(result["ssh_host"],"10.0.0.5")
        self.assertEqual(result["ssh_port"],22)
        self.assertEqual(result["next_command"],"boxweave client")

    def test_local_check_does_not_claim_authenticated_mcp(self):
        with patch.object(host_cli,"public_state",return_value=PUBLIC), \
             patch.object(host_cli,"web_observation",return_value={"reachable":True,"authenticated":False}) as reader, \
             patch("sys.stdout",new_callable=io.StringIO) as out:
            self.assertEqual(host_cli.main(["setup","--mode","local","--check","--format","json"]),0)
        result=json.loads(out.getvalue())
        self.assertEqual(result["local_web_probe"]["authenticated"],False)
        self.assertEqual(result["authenticated_mcp"],"not_verified")
        reader.assert_called_once_with(18640)

    def test_safe_mode_never_executes_host_command(self):
        with patch.object(host_cli,"public_state",return_value=PUBLIC), \
             patch.object(host_cli,"observe") as execute, \
             patch("sys.stdout",new_callable=io.StringIO):
            self.assertEqual(host_cli.main(["setup","--mode","ssh","--check"]),0)
        execute.assert_not_called()

    def test_source_only_host_helper_isolation(self):
        import subprocess,sys
        output=subprocess.run([sys.executable,"-I","bwmini/host_cli.py","setup","--help"],
                              capture_output=True,text=True,timeout=10)
        self.assertEqual(output.returncode,0,output.stderr)
        self.assertIn("--mode",output.stdout)

if __name__ == "__main__":
    unittest.main()

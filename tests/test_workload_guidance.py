"""Live-executed, isolated managed-workload guidance fixture tests (no SSH)."""
import base64
import os
from pathlib import Path
import re
import shutil
import subprocess
import sys
import tempfile
import unittest

from bwmini import client_setup
from test_client_setup import ready_state


def extract_function():
    source = client_setup.POWERSHELL
    match = re.search(r"(?ms)^function Merge-BoxWeaveGuidance\(.*?^\}\n", source)
    if not match:
        raise AssertionError("Generated PowerShell guidance merge missing")
    return match.group(0)


@unittest.skipUnless(shutil.which("pwsh"), "PowerShell Core not installed")
class PowerShellGuidanceTests(unittest.TestCase):
    def test_existing_instructions_preserved_replaced_and_idempotent(self):
        code = r"""
$ErrorActionPreference = 'Stop'
$old = "# unrelated instructions" + "`r`n"
$block = "<!-- BEGIN BOXWEAVE MANAGED -->" + "`n" + "ssh-ed25519 NEW" + "`n" + "<!-- END BOXWEAVE MANAGED -->"
$next = Merge-BoxWeaveGuidance $old $block
if (-not $next.Contains("# unrelated instructions")) { throw "OTHER_CONTENT_REMOVED" }
if (-not $next.Contains("ssh-ed25519 NEW")) { throw "BLOCK_MISSING" }
if ((Merge-BoxWeaveGuidance $next $block) -cne $next) { throw "NOT_IDEMPOTENT" }
$changed = $block.Replace("NEW","NEWER")
$replaced = Merge-BoxWeaveGuidance $next $changed
if (-not $replaced.Contains("ssh-ed25519 NEWER")) { throw "KEY_NOT_UPDATED" }
if (-not $replaced.Contains("# unrelated instructions")) { throw "OTHER_CONTENT_REMOVED" }
try {
  $null = Merge-BoxWeaveGuidance "<!-- BEGIN BOXWEAVE MANAGED -->" $block
  throw "MISSING_END_ACCEPTED"
} catch {
  if ($_.Exception.Message -eq "MISSING_END_ACCEPTED") { throw }
}
try {
  $null = Merge-BoxWeaveGuidance ($next+$next) $block
  throw "DOUBLE_BLOCK_ACCEPTED"
} catch {
  if ($_.Exception.Message -eq "DOUBLE_BLOCK_ACCEPTED") { throw }
}
""".replace("`",chr(96))
        with tempfile.TemporaryDirectory() as tmp:
            script = Path(tmp)/"guidance.ps1"
            script.write_text(extract_function()+"\n"+code,encoding="utf-8")
            result=subprocess.run(["pwsh","-NoProfile","-File",str(script)],
                                  capture_output=True,text=True,timeout=30)
            self.assertEqual(result.returncode,0,result.stdout+result.stderr)

    def test_public_metadata_is_rendered_without_secrets(self):
        text=client_setup.render(ready_state(),"example-admin")
        self.assertIn("for this server's connect tool.",text.lower())
        self.assertIn("ssh_keys",text)
        self.assertNotIn("__MANAGED_PUBLIC_KEY__",text)
        self.assertNotIn("-----BEGIN OPENSSH PRIVATE KEY-----",text)


class WslGuidancePythonTests(unittest.TestCase):
    def script(self):
        match = re.search(
            r"(?s)\$installWslGuidance = @'\n(.*?)\n'@\n", client_setup.POWERSHELL,
        )
        if not match:
            raise AssertionError("WSL guidance Python source missing")
        return match.group(1)

    def execute(self, root, payload):
        env = dict(os.environ, HOME=str(root))
        return subprocess.run(
            [sys.executable, "-c", self.script(),
             base64.b64encode(payload.encode()).decode()],
            env=env, capture_output=True,text=True,timeout=30,
        )

    def test_wsl_guidance_merges_and_respects_existing_override(self):
        with tempfile.TemporaryDirectory() as tmp:
            root=Path(tmp);codex=root/".codex";codex.mkdir()
            regular=codex/"AGENTS.md"
            override=codex/"AGENTS.override.md"
            regular.write_text("Unrelated project rules\n")
            override.write_text("Override rules\n")
            marker="<!-- BEGIN BOXWEAVE MANAGED -->\n"
            policy1=marker+"ssh-ed25519 KEY1\n<!-- END BOXWEAVE MANAGED -->"
            policy2=marker+"ssh-ed25519 KEY2\n<!-- END BOXWEAVE MANAGED -->"
            one=self.execute(root,policy1)
            self.assertEqual(one.returncode,0,one.stderr)
            self.assertEqual(regular.read_text(),"Unrelated project rules\n")
            self.assertIn("ssh-ed25519 KEY1",override.read_text())
            two=self.execute(root,policy2)
            self.assertEqual(two.returncode,0,two.stderr)
            self.assertIn("Override rules",override.read_text())
            self.assertIn("ssh-ed25519 KEY2",override.read_text())
            self.assertNotIn("ssh-ed25519 KEY1",override.read_text())
            self.assertEqual(self.execute(root,policy2).returncode,0)
            self.assertEqual(len(list(codex.glob("AGENTS.override.md.boxweave-*.bak"))),2)

    def test_wsl_guidance_symlink_and_marker_fail_closed(self):
        with tempfile.TemporaryDirectory() as tmp:
            root=Path(tmp);codex=root/".codex";codex.mkdir()
            policy="<!-- BEGIN BOXWEAVE MANAGED -->\nkey\n<!-- END BOXWEAVE MANAGED -->"
            (codex/"AGENTS.md").write_text("<!-- BEGIN BOXWEAVE MANAGED -->")
            res=self.execute(root,policy)
            self.assertNotEqual(res.returncode,0)
            (codex/"AGENTS.md").unlink()
            (codex/"AGENTS.md").symlink_to(root/"elsewhere")
            res=self.execute(root,policy)
            self.assertNotEqual(res.returncode,0)
            self.assertFalse((root/"elsewhere").exists())


if __name__=="__main__":
    unittest.main()

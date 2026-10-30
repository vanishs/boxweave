"""Execute the generated PowerShell TOML merger in isolation (no WSL/SSH/host writes)."""
import json
from pathlib import Path
import re
import shutil
import subprocess
import tempfile
import tomllib
import unittest

from bwmini import client_setup


def merger_source():
    source = client_setup.POWERSHELL
    match = re.search(
        r"(?ms)^function Merge-BoxWeaveToml\(.*?^\}\n(?=\$wslString)",
        source,
    )
    if not match:
        raise AssertionError("PowerShell merger function missing")
    return match.group(0)


@unittest.skipUnless(shutil.which("pwsh"), "PowerShell Core not installed")
class GeneratedPowerShellMergeTests(unittest.TestCase):

    def test_ssh_destination_one_argument_in_real_powershell(self):
        """Regression: user@ plus host in array literal became two argv entries."""
        source = client_setup.POWERSHELL
        self.assertEqual(source.count("$sshTarget = 'boxweave@' + $ServerHost"), 1)
        snippet = "$sshTarget = " + source.split("$sshTarget = ", 1)[1].split("$homeDir = ", 1)[0]
        code = r'''
$ErrorActionPreference = 'Stop'
$Distro = 'Ubuntu'
$ServerHost = '192.168.2.233'
$ServerPort = 22
$RemoteMcp = '/usr/bin/python3 /usr/local/lib/boxweave/default/mcp_gate.py'
$identity = @{ key = '/home/test/.config/boxweave/id_ed25519'; known_hosts = '/home/test/.config/boxweave/known_hosts' }
''' + snippet + r'''
ConvertTo-Json -InputObject @($sshArgs) -Compress
'''
        with tempfile.TemporaryDirectory() as temp:
            script = Path(temp) / "ssh-args.ps1"
            script.write_text(code, encoding="utf-8")
            process = subprocess.run(
                ["pwsh", "-NoProfile", "-File", str(script)],
                capture_output=True, text=True, timeout=30,
            )
            self.assertEqual(process.returncode, 0, process.stderr)
            argv = json.loads(process.stdout)
            self.assertEqual(argv[-2], "boxweave@192.168.2.233")
            self.assertEqual(argv[-1], "/usr/bin/python3 /usr/local/lib/boxweave/default/mcp_gate.py")
            self.assertEqual(argv.count("boxweave@192.168.2.233"), 1)
            self.assertNotIn("boxweave@", argv)
            self.assertNotIn("192.168.2.233", argv)

    def test_nested_boxweave_tables_preserved_and_replace_explicit(self):
        original = '''# Codex desktop config
[mcp_servers.boxweave]
command = "legacy-ssh"
args = ["old"]
enabled = true
# keep custom comment

[mcp_servers.boxweave.env]
PRIVATE_OPTION = "keep"
[mcp_servers.boxweave.http_headers]
X_Client = "keep"
[mcp_servers.another]
command = "unrelated"
'''
        code = r'''
$original = @'
__ORIGINAL__
'@
try {
    $null = Merge-BoxWeaveToml $original '"wsl.exe"' '["--distribution","Ubuntu"]' $false
    throw 'NONREPLACEMENT_MUST_FAIL'
} catch {
    if ($_.Exception.Message -notlike '*-Replace*') { throw }
}
$result = Merge-BoxWeaveToml $original '"wsl.exe"' '["--distribution","Ubuntu"]' $true
$again = Merge-BoxWeaveToml $result '"wsl.exe"' '["--distribution","Ubuntu"]' $false
if ($result -cne $again) { throw 'IDEMPOTENCE_FAILED' }
[IO.File]::WriteAllText($OutFile, $result, [Text.UTF8Encoding]::new($false))
'''.replace("__ORIGINAL__", original.rstrip("\n"))
        with tempfile.TemporaryDirectory() as temp:
            tmp = Path(temp)
            script = tmp / "merge.ps1"
            result = tmp / "result.toml"
            script.write_text("param([string]$OutFile)\n"+merger_source()+"\n"+code)
            p = subprocess.run(["pwsh","-NoProfile","-File",str(script),str(result)],
                               capture_output=True,text=True,timeout=30)
            self.assertEqual(p.returncode,0,f"{p.stdout}\n{p.stderr}")
            written = result.read_text(encoding="utf-8")
            before = tomllib.loads(original)
            after = tomllib.loads(written)
            self.assertEqual(after["mcp_servers"]["boxweave"]["command"], "wsl.exe")
            self.assertEqual(after["mcp_servers"]["boxweave"]["args"], ["--distribution", "Ubuntu"])
            self.assertTrue(after["mcp_servers"]["boxweave"]["enabled"])
            self.assertEqual(after["mcp_servers"]["boxweave"]["env"], before["mcp_servers"]["boxweave"]["env"])
            self.assertEqual(after["mcp_servers"]["boxweave"]["http_headers"],
                             before["mcp_servers"]["boxweave"]["http_headers"])
            self.assertEqual(after["mcp_servers"]["another"], before["mcp_servers"]["another"])
            self.assertIn("# keep custom comment",written)

    def test_nested_only_and_multiline_transport_fail_closed(self):
        code = r'''
$nestedOnly = "[mcp_servers.boxweave.env]" + "%BT%n" + "A = 'keep'"
$multiline = "[mcp_servers.boxweave]" + "%BT%n" + "args = [" + "%BT%n" + '"old",' + "%BT%n" + "]"
foreach ($bad in @($nestedOnly, $multiline)) {
    try {
        $null = Merge-BoxWeaveToml $bad '"wsl.exe"' '["Ubuntu"]' $true
        throw 'UNSAFE_MERGE_ACCEPTED'
    } catch {
        if ($_.Exception.Message -eq 'UNSAFE_MERGE_ACCEPTED') { throw }
    }
}
'''.replace("%BT%",chr(96))
        with tempfile.TemporaryDirectory() as temp:
            script = Path(temp)/"failclosed.ps1"
            script.write_text(merger_source()+"\n"+code)
            p = subprocess.run(["pwsh","-NoProfile","-File",str(script)],
                               capture_output=True,text=True,timeout=30)
            self.assertEqual(p.returncode,0,f"{p.stdout}\n{p.stderr}")


if __name__ == "__main__":
    unittest.main()

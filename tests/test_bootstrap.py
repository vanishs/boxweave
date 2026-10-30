"""Non-root bootstrap trust-boundary tests; never create users/services or tokens."""
import hashlib
import os
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch
from bwmini import bootstrap as b

class BootstrapTests(unittest.TestCase):
    def test_endpoint_rejects_public_credentials_and_ambiguous_urls(self):
        self.assertEqual(b.endpoint('http://127.0.0.1:18080'), 'http://127.0.0.1:18080')
        for value in ('https://example.com:443', 'http://localhost:8080', 'http://127.0.0.1',
                      'http://user:pass@127.0.0.1:8080', 'http://127.0.0.1:8080/',
                      'http://127.0.0.1:8080?token=x', 'http://127.0.0.1:bad'):
            with self.subTest(value=value), self.assertRaises(b.Refused): b.endpoint(value)

    def test_reviewed_snapshots_bytes_and_rejects_tamper_and_links(self):
        with tempfile.TemporaryDirectory() as folder:
            p=Path(folder)/'artifact'; p.write_bytes(b'reviewed-public-payload')
            sha=hashlib.sha256(p.read_bytes()).hexdigest()
            self.assertEqual(b.reviewed(p,sha), b'reviewed-public-payload')
            p.write_bytes(b'changed')
            with self.assertRaisesRegex(b.Refused,'DIGEST'): b.reviewed(p,sha)
            link=Path(folder)/'link'; link.symlink_to(p)
            with self.assertRaisesRegex(b.Refused,'SYMLINK'): b.reviewed(link,sha)
            hard=Path(folder)/'hard'; os.link(p,hard)
            with self.assertRaisesRegex(b.Refused,'REGULAR'): b.reviewed(p,sha)

    def test_artifact_requires_independent_complete_digest(self):
        for digest in ('', 'abc', 'A'*64):
            with self.assertRaisesRegex(b.Refused,'SHA256'): b.reviewed('/does-not-exist',digest)

    def test_paths_reject_traversal_and_shell_control_characters(self):
        for path in ('relative', '/tmp/../etc/config', '/tmp/bad\nname'):
            with self.assertRaises(b.Refused): b.safe_path(path)

    def test_launcher_uses_file_reference_and_clears_inherited_credentials(self):
        text=b.launcher_text(Path('/usr/local/lib/boxweave/test/mcp'),
                             Path('/usr/local/lib/boxweave/test/application.token'),
                             'http://127.0.0.1:18080','/home/test').decode()
        self.assertIn('/usr/bin/env -i ',text)
        self.assertIn('CONTAINARIUM_MCP_ALLOW_INSECURE=true',text)
        self.assertIn('CONTAINARIUM_JWT_TOKEN_FILE=',text)
        self.assertNotIn('CONTAINARIUM_JWT_TOKEN=',text)
        self.assertNotIn('cat ',text)
        self.assertNotIn('secrets:read',b.SCOPES)
        self.assertNotIn('*',b.SCOPES)

    def test_nonroot_apply_and_rotation_never_launch_host_commands(self):
        from types import SimpleNamespace
        with patch.object(b.os,'geteuid',return_value=1000), patch.object(b.subprocess,'run') as run:
            with self.assertRaisesRegex(b.Refused,'ROOT_REQUIRED'): b.apply(SimpleNamespace(),{})
            with self.assertRaisesRegex(b.Refused,'ROOT_REQUIRED'): b.rotate('/unread')
            run.assert_not_called()

    def test_public_directory_remains_traversable_under_private_umask(self):
        import stat
        with tempfile.TemporaryDirectory() as folder:
            path=Path(folder)/'launcher'
            old=os.umask(0o077)
            try: b.make_directory(path,0o755)
            finally: os.umask(old)
            self.assertEqual(stat.S_IMODE(path.stat().st_mode),0o755)

    def test_atomic_preserves_previous_file_if_publication_fails(self):
        with tempfile.TemporaryDirectory() as folder:
            p=Path(folder)/'public'; p.write_bytes(b'previous')
            with patch.object(b.os,'replace',side_effect=OSError('injected')):
                with self.assertRaises(OSError): b.atomic(p,b'candidate')
            self.assertEqual(p.read_bytes(),b'previous')
            self.assertEqual([x.name for x in Path(folder).iterdir()],['public'])

if __name__=='__main__': unittest.main()

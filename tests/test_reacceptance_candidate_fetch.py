"""Original wrapper bytes cannot be substituted or unpack unsafe members."""
import hashlib
from pathlib import Path
import tempfile
import unittest
import zipfile
from unittest.mock import patch

from tools.reacceptance import candidate_fetch as subject
from tools.reacceptance.engine import AcceptanceFault


class CandidateFetchTests(unittest.TestCase):
    def test_wrong_wrapper_digest_does_not_create_output(self):
        with tempfile.TemporaryDirectory() as d:
            root = Path(d)
            p = root / 'original.zip'
            p.write_bytes(b'substitute')
            with self.assertRaisesRegex(AcceptanceFault, 'DIGEST_MISMATCH'):
                subject.unpack(p, 'sha256:' + 'a'*64, root/'out')
            self.assertFalse((root/'out').exists())

    def test_unexpected_and_symlink_members_refused_before_extraction(self):
        with tempfile.TemporaryDirectory() as d:
            root = Path(d)
            for unsafe in ('../escape', 'online.py'):
                p = root / (str(len(unsafe)) + '.zip')
                with zipfile.ZipFile(p, 'w') as z:
                    if unsafe == '../escape':
                        z.writestr(unsafe, b'bad')
                    else:
                        for name in subject.ASSETS:
                            info = zipfile.ZipInfo(name)
                            info.external_attr = (0o120777 if name == unsafe else 0o100600) << 16
                            z.writestr(info, b'bad')
                digest = 'sha256:' + hashlib.sha256(p.read_bytes()).hexdigest()
                with self.assertRaises(AcceptanceFault):
                    subject.unpack(p, digest, root/'out')
                self.assertFalse((root/'out').exists())

    def test_exact_safe_wrapper_preserves_bytes(self):
        with tempfile.TemporaryDirectory() as d:
            root=Path(d); p=root/'candidate.zip'
            with zipfile.ZipFile(p,'w') as z:
                for name in subject.ASSETS:
                    z.writestr(name,name.encode())
            digest='sha256:'+hashlib.sha256(p.read_bytes()).hexdigest()
            with patch.object(subject,'asset_index',return_value={'checked':True}):
                self.assertEqual(subject.unpack(p,digest,root/'out'),{'checked':True})
            self.assertEqual((root/'out'/'online.py').read_bytes(),b'online.py')

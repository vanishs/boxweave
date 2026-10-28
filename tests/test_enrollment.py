"""Non-root public-key-only enrollment, confined to a temporary home."""
import base64
import fcntl
from concurrent.futures import ThreadPoolExecutor
from pathlib import Path
import os
import pwd
import socket
import struct
import tempfile
import unittest
from unittest.mock import patch
from scripts.authorize_native_key import authorize, Rejected

def key(byte):
    blob=struct.pack(">I",11)+b"ssh-ed25519"+struct.pack(">I",32)+bytes([byte])*32
    return "ssh-ed25519 "+base64.b64encode(blob).decode()+" fixture"

class EnrollmentTests(unittest.TestCase):
    def setUp(self):
        self.tmp=tempfile.TemporaryDirectory()
        self.home=Path(self.tmp.name)
        self.user=pwd.getpwuid(os.geteuid()).pw_name
        self.host=socket.gethostname()
        self.folder=self.home/".ssh"
        self.folder.mkdir(mode=0o700)
        self.file=self.folder/"authorized_keys"
        self.file.write_text(key(1)+"\n")
        self.file.chmod(0o600)
    def tearDown(self):self.tmp.cleanup()
    def run_key(self, public=None):
        return authorize(self.home,public or key(2),self.user,self.host)
    def test_preservation_backup_restriction_and_idempotence(self):
        old=self.file.read_bytes()
        result=self.run_key()
        self.assertTrue(result["changed"])
        self.assertEqual(Path(result["backup"]).read_bytes(),old)
        data=self.file.read_bytes()
        self.assertTrue(data.startswith(old))
        self.assertIn(b'restrict,command="/usr/local/bin/boxweave-mcp" ssh-ed25519 ',data)
        self.assertEqual(self.file.stat().st_mode & 0o777,0o600)
        self.assertFalse(self.run_key()["changed"])
        self.assertEqual(self.file.read_bytes(),data)
    def test_conflicting_unrestricted_key_is_unchanged(self):
        self.file.write_text(key(2)+"\n")
        before=self.file.read_bytes()
        with self.assertRaisesRegex(Rejected,"KEY_CONFLICT"):self.run_key()
        self.assertEqual(self.file.read_bytes(),before)
    def test_identity_malformed_private_text_and_unsafe_paths(self):
        before=self.file.read_bytes()
        with self.assertRaisesRegex(Rejected,"IDENTITY_MISMATCH"):
            authorize(self.home,key(2),self.user,"wrong-host")
        for value in ["invalid","-----BEGIN OPENSSH PRIVATE KEY-----",key(2)+"\n"+key(3)]:
            with self.subTest(value=value), self.assertRaisesRegex(Rejected,"PUBLIC_KEY_INVALID"):
                self.run_key(value)
        self.file.chmod(0o666)
        with self.assertRaisesRegex(Rejected,"PATH_NOT_PRIVATE"):self.run_key()
        self.file.chmod(0o600)
        self.file.unlink()
        self.file.symlink_to(self.home/"missing")
        with self.assertRaisesRegex(Rejected,"UNSAFE_PATH"):self.run_key()
        self.assertEqual(before,key(1).encode()+b"\n")
    def test_concurrent_same_key_is_not_duplicated(self):
        with ThreadPoolExecutor(max_workers=2) as pool:
            results=list(pool.map(lambda _:self.run_key(),range(2)))
        self.assertEqual(sum(item["changed"] for item in results),1)
        self.assertEqual(self.file.read_text().count("boxweave-v2-native"),1)
    def test_publication_failure_keeps_original(self):
        before=self.file.read_bytes()
        with patch("scripts.authorize_native_key.os.replace",side_effect=OSError("synthetic")):
            with self.assertRaises(OSError):self.run_key()
        self.assertEqual(self.file.read_bytes(),before)
        self.assertFalse(list(self.folder.glob(".boxweave-public-*")))

    def test_busy_lock_fails_before_key_publication(self):
        lock=os.open(self.folder/".boxweave-native.lock",os.O_CREAT|os.O_RDWR,0o600)
        before=self.file.read_bytes()
        try:
            fcntl.flock(lock,fcntl.LOCK_EX|fcntl.LOCK_NB)
            with patch("scripts.authorize_native_key.time.monotonic",side_effect=[0,11]):
                with self.assertRaisesRegex(Rejected,"ENROLLMENT_BUSY"):self.run_key()
        finally:
            os.close(lock)
        self.assertEqual(self.file.read_bytes(),before)

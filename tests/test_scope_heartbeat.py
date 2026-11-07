"""Installed root scope heartbeat and bounded parallel SSH trust scanner."""
import json
import os
from pathlib import Path
import tempfile
import threading
import time
import unittest
from types import SimpleNamespace
from unittest.mock import patch

from bwmini import scope_heartbeat, hostkeys
from bwmini.coordination_bridge import published_scope
from bwmini.operation_coordination import CoordinationFault


class ScopeHeartbeatTests(unittest.TestCase):
    def setUp(self):
        temp=tempfile.TemporaryDirectory()
        self.addCleanup(temp.cleanup)
        self.root=Path(temp.name)
        self.public=self.root/"public"
        self.public.mkdir(mode=0o755)
        self.state=self.root/"state.json"
        self.state.write_text(json.dumps({"hostname":"dev",
            "managed_files":{"/managed":"sha256"}}))
        self.state.chmod(0o600)
        self.target=self.public/"scope.json"

    def fake_root_lstat(self,path):
        info=os.stat(path,follow_symlinks=False)
        if path in (self.state,self.public,self.target):
            return SimpleNamespace(st_uid=0,st_mode=info.st_mode,
                st_nlink=info.st_nlink,st_size=info.st_size,
                st_ino=info.st_ino,st_dev=info.st_dev)
        return info

    def test_live_scope_independent_of_stale_ssh_inventory(self):
        commands=[]
        def inspect(argv,**kw):
            commands.append((argv,kw))
            return SimpleNamespace(returncode=0,stdout='{"api_version":"1.0"}')
        (self.public/"targets.json").write_text(
            json.dumps({"observed_at":1,"targets":{}}))
        with (patch.object(scope_heartbeat.os,"geteuid",return_value=0),
              patch.object(scope_heartbeat,"SCOPE",self.target),
              patch.object(scope_heartbeat,"atomic",
                           side_effect=lambda path,data,*a:path.write_bytes(data)),
              patch.object(Path,"lstat",self.fake_root_lstat)):
            response=scope_heartbeat.publish(
                state=self.state,directory=self.public,
                now=lambda:100.,inspect=inspect,hostname=lambda:"dev")
            self.assertEqual(published_scope(self.target,now=lambda:110.),
                             response["scope"])
            with self.assertRaisesRegex(CoordinationFault,
                                       "COORDINATION_SCOPE_UNVERIFIED"):
                published_scope(self.target,now=lambda:151.)
        self.assertEqual(commands[0][0],
            ["/usr/bin/incus","--force-local","query","/1.0"])
        self.assertLessEqual(commands[0][1]["timeout"],5)

    def test_native_unavailable_never_publishes_fake_freshness(self):
        def offline(*a,**kw):
            return SimpleNamespace(returncode=1,stdout="")
        with (patch.object(scope_heartbeat.os,"geteuid",return_value=0),
              patch.object(scope_heartbeat,"SCOPE",self.target),
              patch.object(Path,"lstat",self.fake_root_lstat)):
            with self.assertRaisesRegex(ValueError,
                                       "SCOPE_NATIVE_PROVIDER_UNVERIFIED"):
                scope_heartbeat.publish(
                    state=self.state,directory=self.public,
                    inspect=offline,hostname=lambda:"dev")
        self.assertFalse(self.target.exists())

    def test_scan_parallel_reads_three_keys_once_per_box(self):
        fixtures=[{
            "name":f"scan-{i}-container","status":"Running",
            "created_at":"2026-11-07T00:00:00Z",
            "config":{"user.containarium.label.os-type":"ubuntu_2404",
                      "volatile.uuid":f"uuid-{i}"},
            "state":{"network":{"eth0":{"addresses":[{
                "family":"inet","scope":"global",
                "address":f"10.1.0.{i+10}"}]}}}
        } for i in range(16)]
        lock=threading.Lock()
        active=[0];peak=[0];calls=[]
        def command(argv):
            if "list" in argv:
                return json.dumps(fixtures)
            if "exec" not in argv:
                raise AssertionError("Unexpected native command")
            with lock:
                active[0]+=1;peak[0]=max(peak[0],active[0]);calls.append(argv)
            time.sleep(.01)
            with lock:
                active[0]-=1
            return ("ssh-ed25519 KEYED\n"
                    "ecdsa-sha2-nistp256 KEYEC\n"
                    "ssh-rsa KEYRSA\n")
        home=self.root/"home"
        home.mkdir()
        with (patch.object(hostkeys.os,"geteuid",return_value=0),
              patch.object(hostkeys,"PUBLIC",self.public),
              patch.object(hostkeys,"HOME",home),
              patch.object(hostkeys,"command",side_effect=command),
              patch.object(hostkeys,"atomic",
                           side_effect=lambda path,data,*a:path.write_bytes(data)),
              patch.object(hostkeys.pwd,"getpwnam",return_value=SimpleNamespace(
                  pw_uid=os.getuid(),pw_gid=os.getgid()))):
            hostkeys.reconcile()
        self.assertGreater(peak[0],1)
        self.assertLessEqual(peak[0],8)
        self.assertEqual(len(calls),16)
        entries=json.loads((self.public/"targets.json").read_text())
        self.assertEqual(len(entries["targets"]),16)
        self.assertEqual(
            len((home/".ssh"/"known_hosts").read_text().splitlines()),2+3*16)


if __name__=="__main__":
    unittest.main()

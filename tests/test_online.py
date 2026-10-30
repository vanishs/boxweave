"""Non-root contract checks for streamed release, SSH trust and client enrollment."""
import base64,hashlib,json,os,struct,tempfile,time,unittest
from pathlib import Path
from unittest.mock import patch
from bwmini import online,hostkeys,mcp_gate
from scripts.build_release_manifest import build

class StreamedInstallTests(unittest.TestCase):
    def key(self):
        blob=struct.pack(">I",11)+b"ssh-ed25519"+struct.pack(">I",32)+b"x"*32
        return "ssh-ed25519 "+base64.b64encode(blob).decode()+" test-public"

    def test_public_keys_require_full_valid_ed25519_not_fingerprints(self):
        self.assertTrue(online.public_key(self.key()).startswith("ssh-ed25519 "))
        for value in ("SHA256:fingerprint","ssh-ed25519 AAAA","not-a-key"):
            with self.subTest(value=value),self.assertRaises((online.Failure,ValueError)):online.public_key(value)

    def test_create_gate_never_allows_implicit_private_key_return(self):
        mcp_gate.explicit_keys({"ssh_keys":[self.key()]})
        for args in ({},{"ssh_keys":[]},{"ssh_keys":["SHA256:fingerprint"]},None):
            with self.subTest(args=args),self.assertRaises(ValueError):mcp_gate.explicit_keys(args)

    def test_connect_gate_requires_fresh_exact_incarnation_and_ip(self):
        target={"name":"work-container","created":"123","user":"work","ip":"10.203.0.2","keys":[self.key()]}
        registry={"observed_at":time.time(),"targets":{"work":target}}
        observed={"name":"work-container","createdAt":"123","username":"work","network":{"ipAddress":"10.203.0.2"},"state":"CONTAINER_STATE_RUNNING"}
        mcp_gate.trusted({"box":"work"},registry,observed)
        for change in ({"createdAt":"124"},{"network":{"ipAddress":"10.203.0.3"}},{"state":"CONTAINER_STATE_STOPPED"}):
            with self.subTest(change=change),self.assertRaises(ValueError):mcp_gate.trusted({"box":"work"},registry,observed|change)
        for args in ({"box":"other"},{"box":"work","host":"example.com"},{"box":"work","user":"root"}):
            with self.subTest(args=args),self.assertRaises(ValueError):mcp_gate.trusted(args,registry,observed)
        with self.assertRaises(ValueError):mcp_gate.trusted({"box":"work"},registry|{"observed_at":time.time()-60},observed)

    def test_hostkey_inventory_excludes_core_and_unmanaged_instances(self):
        normal={"name":"work-container","status":"Running","config":{"user.containarium.label.os-type":"ubuntu_2404"}}
        self.assertTrue(hostkeys.eligible(normal))
        for change in ({"name":"containarium-core-postgres"},{"config":{}},{"status":"Stopped"},{"name":"other"}):
            self.assertFalse(hostkeys.eligible(normal|change))

    def test_release_manifest_covers_every_root_helper_and_exact_archive(self):
        with tempfile.TemporaryDirectory() as folder:
            root=Path(folder)/"root";out=Path(folder)/"out";(root/"bwmini").mkdir(parents=True);out.mkdir()
            (root/"VERSION").write_text("0.15.0\n");(root/"install.sh").write_text("fixture")
            for name in online.RELEASE_FILES:
                if name=="boxweave.zip":(out/name).write_bytes(b"reviewed-app")
                else:(root/"bwmini"/name).write_text("public-"+name)
            manifest=build(root,out,"a"*40)
            self.assertEqual(set(manifest["files"]),set(online.RELEASE_FILES))
            self.assertEqual(online.manifest_read(out/"release.json","0.15.0"),manifest)
            manifest["files"]["unexpected.py"]="0"*64;(out/"release.json").write_text(json.dumps(manifest))
            with self.assertRaises(online.Failure):online.manifest_read(out/"release.json","0.15.0")

    def test_download_never_publishes_unverified_bytes(self):
        with tempfile.TemporaryDirectory() as folder:
            path=Path(folder)/"artifact"
            with self.assertRaises(online.Failure):online.download("http://example.com/file",path,"0"*64)
            with self.assertRaises(online.Failure):online.download("https://example.com/file",path,"bad")
            path.write_bytes(b"changed")
            with self.assertRaises(online.Failure):online.download("https://example.com/file",path,"0"*64)
            self.assertEqual(path.read_bytes(),b"changed")

    def test_unmanaged_postgres_is_refused_before_any_mutation(self):
        with tempfile.TemporaryDirectory() as folder:
            base=Path(folder);etc=base/'etc';data=base/'data';etc.mkdir();data.mkdir()
            (etc/'foreign-cluster').mkdir()
            with patch.object(online,'PG_ETC',etc),patch.object(online,'PG_DATA',data),patch.object(online,'atomic') as write,patch.object(online,'run') as run:
                with self.assertRaisesRegex(online.Failure,'FOREIGN_POSTGRES_CONFIGURATION'):online.prepare_postgres({},time.monotonic()+600)
                write.assert_not_called();run.assert_not_called()

    def test_nonroot_install_stops_before_creating_host_state(self):
        with patch.object(online.os,"geteuid",return_value=1000),patch.object(online,"directory") as mutate:
            with self.assertRaisesRegex(online.Failure,"ROOT_REQUIRED"):online.install(Path("/unused"),"0.15.0")
            mutate.assert_not_called()

if __name__=="__main__":unittest.main()

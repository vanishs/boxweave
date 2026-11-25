"""Offline transfer planner and assembly regression tests; no MCP effects."""
import base64
import copy
import json
import sys
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "dev"))
import transfer_plan as tr

BOX = "12345678-1234-1234-1234-123456789abc"


class TransferPlanTests(unittest.TestCase):
    def setUp(self):
        temp = tempfile.TemporaryDirectory()
        self.addCleanup(temp.cleanup)
        self.root = Path(temp.name)
        self.source = self.root / "source.bin"
        self.data = bytes((i*13 + i//40960) % 256 for i in range(81920))
        self.source.write_bytes(self.data)
        self.target = self.root / "target.bin"
        self.bundle = self.root / "bundle"
        self.result = tr.plan(self.source, self.bundle, BOX, str(self.target), "fixture/work", "attempt-1")
        self.manifest = tr.strict_load((self.bundle/"manifest.json").read_bytes())
        self.journal = tr.strict_load((self.bundle/"intent.json").read_bytes())
        self.rows = tr.actions(self.manifest)

    def stage(self, count=None):
        for row in self.rows[:-1][:count]:
            args = tr.strict_load((self.bundle/(row["action"]+"-request.json")).read_bytes())
            self.assertEqual(args["box_id"], BOX)
            self.assertEqual(args["mode"], "create")
            self.assertEqual(args["idempotency_key"],row["idempotency_key"])
            data = base64.b64decode(args["content_base64"],validate=True)
            self.assertEqual(tr.digest(data),row["sha256"])
            tr.exclusive(args["path"],data)

    def pages(self,row,data):
        result=[]
        for offset in range(0,len(data),65536):
            page=data[offset:offset+65536]
            result.append({"box_id":BOX,"path":row["path"],"size":len(data),"offset":offset,
                           "data_base64":base64.b64encode(page).decode(),
                           "page_sha256":tr.digest(page),"ended":offset+len(page)==len(data),
                           "whole_file":len(data)<=65536})
        return result

    def record(self,idx,data):
        row=self.rows[idx]
        self.journal["records"][row["action"]].update(operation_id=f"{idx+1:032x}",state="SUCCEEDED",
            pages=self.pages(row,data),exit_code=0 if row["action"]=="assemble" else None)

    def test_plan_is_stable_and_exclusive(self):
        manifest,chunks=tr.make_manifest(self.data,BOX,str(self.target),"fixture/work","attempt-1",40960)
        self.assertEqual(manifest,self.manifest)
        self.assertEqual(len(chunks),2)
        self.assertEqual(len({r["idempotency_key"] for r in self.rows}),4)
        self.assertLess((self.bundle/"part-0000-request.json").stat().st_size,65536)
        self.assertEqual(tr.assess(self.manifest,self.journal)["next_action"],"part-0000")
        with self.assertRaises(FileExistsError):
            tr.plan(self.source,self.bundle,BOX,str(self.target),"fixture/work","attempt-1")
        self.assertEqual(tr.strict_load((self.bundle/"intent.json").read_bytes()),self.journal)

    def test_assembly_integrity_and_independent_paged_readback(self):
        self.stage()
        for i,row in enumerate(self.rows[:-1]):
            self.record(i,Path(row["path"]).read_bytes())
        result=tr.assess(self.manifest,self.journal)
        self.assertEqual((result["status"],result["next_action"]),("READY","assemble"))
        got=tr.assemble(self.rows[-2]["path"])
        self.assertEqual(got["status"],"ASSEMBLED_READBACK_REQUIRED")
        self.assertEqual(self.target.read_bytes(),self.data)
        self.assertTrue(all(Path(row["path"]).exists() for row in self.rows))
        self.record(3,self.target.read_bytes())
        self.assertEqual(tr.assess(self.manifest,self.journal)["status"],"VERIFIED_READBACK")
        self.journal["records"]["assemble"]["pages"].reverse()
        self.assertEqual(tr.assess(self.manifest,self.journal)["status"],"RECONCILE")

    def test_resume_verified_part_and_unknown_receipt(self):
        self.stage(1)
        self.record(0,Path(self.rows[0]["path"]).read_bytes())
        result=tr.assess(self.manifest,self.journal)
        self.assertEqual(result["next_action"],"part-0001")
        self.assertEqual(result["actions"][0]["status"],"VERIFIED")
        record=self.journal["records"]["part-0001"]
        record["state"]="UNKNOWN"
        self.assertEqual(tr.assess(self.manifest,self.journal)["status"],"RECONCILE")
        record["state"]="SUCCEEDED"
        record["operation_id"]="a"*32
        self.assertEqual(tr.assess(self.manifest,self.journal)["status"],"READBACK_REQUIRED")
        record["state"]="FAILED"
        self.assertEqual(tr.assess(self.manifest,self.journal)["status"],"RECONCILE")
        self.journal["records"].pop("part-0001")
        self.assertEqual(tr.assess(self.manifest,self.journal)["status"],"RECONCILE")

    def test_missing_reordered_and_corrupt_parts(self):
        self.stage()
        first,second=map(lambda r:Path(r["path"]),self.rows[:2])
        original=first.read_bytes()
        first.unlink()
        with self.assertRaises(FileNotFoundError):
            tr.assemble(self.rows[-2]["path"])
        self.assertFalse(self.target.exists())
        first.write_bytes(second.read_bytes())
        with self.assertRaises(tr.TransferError):
            tr.assemble(self.rows[-2]["path"])
        self.assertFalse(self.target.exists())
        first.write_bytes(original[:-1])
        with self.assertRaises(tr.TransferError):
            tr.assemble(self.rows[-2]["path"])
        self.assertFalse(self.target.exists())
        first.write_bytes(original)
        tr.assemble(self.rows[-2]["path"])
        self.assertEqual(self.target.read_bytes(),self.data)

    def test_collision_and_symlink_refuse(self):
        self.stage()
        self.target.write_bytes(b"existing")
        with self.assertRaises(tr.TransferError):
            tr.assemble(self.rows[-2]["path"])
        self.assertEqual(self.target.read_bytes(),b"existing")
        self.target.unlink()
        self.target.symlink_to(self.source)
        with self.assertRaises(tr.TransferError):
            tr.assemble(self.rows[-2]["path"])
        self.assertEqual(self.source.read_bytes(),self.data)
        self.target.unlink()
        first=Path(self.rows[0]["path"])
        first.unlink()
        first.symlink_to(self.source)
        with self.assertRaises(OSError):
            tr.assemble(self.rows[-2]["path"])
        self.assertFalse(self.target.exists())

    def test_publication_failure_leaves_target_absent(self):
        self.stage()
        with patch("transfer_plan.os.link",side_effect=OSError("unsupported")):
            with self.assertRaises(OSError):
                tr.assemble(self.rows[-2]["path"])
        self.assertFalse(self.target.exists())
        self.assertTrue(all(Path(row["path"]).exists() for row in self.rows[:-1]))

    def test_manifest_rejects_mutation_paths_and_duplicate_keys(self):
        for invalid in (dict(self.manifest,target="/tmp/../other"),
                        dict(self.manifest,size=True),
                        dict(self.manifest,sha256="x"*64),
                        dict(self.manifest,parts=list(reversed(self.manifest["parts"]))),
                        dict(self.manifest,transfer_id="0"*32)):
            with self.subTest(invalid=invalid),self.assertRaises(tr.TransferError):
                tr.validate_manifest(invalid)
        with self.assertRaises(tr.TransferError):
            tr.strict_load(b'{"a":1,"a":2}')
        with self.assertRaises(tr.TransferError):
            tr.make_manifest(b"",BOX,str(self.target),"work","ref",40960)

    def test_receipt_and_page_identity_not_inferred(self):
        self.stage()
        for i,row in enumerate(self.rows[:-1]):
            self.record(i,Path(row["path"]).read_bytes())
        record=self.journal["records"]["part-0000"]
        record["pages"][0]["page_sha256"]="0"*64
        self.assertEqual(tr.assess(self.manifest,self.journal)["status"],"RECONCILE")
        self.record(0,Path(self.rows[0]["path"]).read_bytes())
        record["pages"][0]["box_id"]="wrong"
        self.assertEqual(tr.assess(self.manifest,self.journal)["status"],"RECONCILE")
        self.record(0,Path(self.rows[0]["path"]).read_bytes())
        self.journal["records"]["part-0001"]["operation_id"]=record["operation_id"]
        with self.assertRaises(tr.TransferError):
            tr.assess(self.manifest,self.journal)
        self.record(1,Path(self.rows[1]["path"]).read_bytes())
        self.record(3,self.data)
        self.journal["records"]["assemble"]["exit_code"]=1
        self.assertEqual(tr.assess(self.manifest,self.journal)["status"],"RECONCILE")

    def test_assembly_cannot_be_inferred_from_final_bytes(self):
        self.record(3,self.data)
        self.assertEqual(tr.assess(self.manifest,self.journal)["status"],"RECONCILE")


if __name__=="__main__":
    unittest.main()

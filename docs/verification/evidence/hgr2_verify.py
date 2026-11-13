#!/usr/bin/env python3
"""Separate-Box read-only verification of exact original Operation/native pre+post snapshots."""
import json,hashlib,sys
from pathlib import Path
folder=Path(__file__).resolve().parent
filenames=("hgr2-precleanup.json","hgr2-postcleanup.json")
expected_hashes=("d46e0d60d10381d92758a7df4f610d0ce033a56b4f3011fab3dd6b5fb2a8c548","e64023b9f825b67877cd18afe0ce6986500d56fb67c4d38badc8a19be0c4f2a1")
data=[]
for fname,expected in zip(filenames,expected_hashes):
    src=(folder/fname).read_bytes()
    digest=hashlib.sha256(src).hexdigest()
    assert digest==expected, (fname,"hash mismatch",digest)
    content=json.loads(src)
    assert content.get("credentials_included") is False
    assert content["candidate_binary_sha256"]=="021127cf723bfbe96c3668965b1e60dcc82856e5e3d41e09e84982568b5c1e85"
    assert content["candidate_service_active"]=="active" and content["primary_service_active"]=="active"
    assert content["original_db_counts"][0]["state"]=="SUCCEEDED"
    assert len(content["original_db_counts"])==1
    data.append(content)
pre,post=data
assert pre["snapshot_kind"]=="hostgo-r2-before-cleanup"
assert post["snapshot_kind"]=="hostgo-r2-after-cleanup"
foreign={"name":"bw-s25-l1a","uuid":"21767890-6d46-4867-889d-0069c1e4ac0d","state":"RUNNING"}
test={"name":"hgcdx-r2-a","uuid":"589a900e-3fb2-4a22-b02c-7de6fff5521f"}
assert pre["native_project_inventory"]==[foreign,{**test,"state":"RUNNING"}],pre["native_project_inventory"]
assert post["native_project_inventory"]==[foreign],post["native_project_inventory"]
before=pre["original_test_operations"]
after=post["original_test_operations"]
assert len(before)==2 and len(after)==4
expected=[
("box_create","efaff82fec24ec6543ea29d50a2c2305","hostgo-round2-20261112"),
("box_start","bbffaae107fa5767d4c6bedfd8099b53","hostgo-round2-fault-20261112"),
("box_stop","3b23f63bedfc3715cc2d4c1bded3f2e7","hostgo-round2-clean-20261112"),
("box_delete","25479b7bb0eef4beccced8992a08f2ef","hostgo-round2-clean-20261112")]
ids=[]
for op,(action,opid,work) in zip(after,expected):
    assert (op["action"],op["id"],op["work_ref"]) == (action,opid,work),op
    assert op["state"]=="SUCCEEDED" and not op["error_class"],op
    assert op["resource_key"]=="box:hgcdx-r2-a" and op["box_uuid"]==test["uuid"] and op["incarnation"]==test["uuid"]
    assert op["native_operation_id"] and op["agent_ref"]=="codex-cli-01672",op
    events=post["original_test_events"][opid]
    assert events and events[-1]["state"]=="SUCCEEDED" and events[0]["state"]=="ADMITTED",(opid,events)
    ids.append(opid)
assert before == after[:2]
assert pre["original_test_events"][ids[0]]==post["original_test_events"][ids[0]]
assert pre["original_test_events"][ids[1]]==post["original_test_events"][ids[1]]
assert pre["original_db_counts"]==[{"n":40,"state":"SUCCEEDED"}]
assert post["original_db_counts"]==[{"n":42,"state":"SUCCEEDED"}]
assert len(pre["owned_test_boxes"])==len(post["owned_test_boxes"])==1
for owned in (pre["owned_test_boxes"][0],post["owned_test_boxes"][0]):
    assert owned["name"]==test["name"] and owned["uuid"]==test["uuid"] and owned["project"]=="user-1002" and owned["incarnation"]==test["uuid"],owned
assert post["owned_test_boxes"][0]["state"]=="DELETED"
report={
    "verifier_box":"boxweave-test-ser2-container",
    "result":"PASS_SCOPED_SNAPSHOT_CROSSCHECK",
    "scope":"Four original Codex MCP Create/Start/Stop/Delete and exact native before/after, no UNKNOWN, foreign UUID unchanged",
    "pre_sha256":expected_hashes[0],
    "post_sha256":expected_hashes[1],
    "verified_original_operation_ids":ids,
    "verified_owned_uuid":test["uuid"],
    "foreign_unchanged_uuid":foreign["uuid"],
    "native_after_only_foreign":True,
    "original_operation_states_all_succeeded":True,
    "unknown_count_before_and_after":0,
    "independent_limit":"Verifier analyzed exported read-only snapshots, not an authenticated direct independent Incus socket session; this does NOT prove full in-flight crash/ACK recovery or complete BH gate signoff.",
    "overall_round2_pass":False
}
(folder/"hgr2-independent-review.json").write_text(json.dumps(report,sort_keys=True,indent=2)+"\n")
print("PASS: independent snapshot crosscheck 4 original operations, foreign project Box unchanged, no UNKNOWN")
print("review_sha256",hashlib.sha256((folder/"hgr2-independent-review.json").read_bytes()).hexdigest())

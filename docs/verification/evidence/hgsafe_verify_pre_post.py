import json,hashlib,collections,datetime
from pathlib import Path
d=Path(__file__).parent
files={}
for stage in ("predelete","postdelete"):
 p=d/("hgsafe-independent-"+stage+".json")
 raw=p.read_bytes()
 files[stage]={"sha256":hashlib.sha256(raw).hexdigest(),"bytes":len(raw)}
 files[stage]["data"]=json.loads(raw)
p=files["predelete"]["data"];q=files["postdelete"]["data"]
sha="021127cf723bfbe96c3668965b1e60dcc82856e5e3d41e09e84982568b5c1e85"
commit="0cf474fbd212c92702591ec85f75e7bea58547ee"
assert p["source_commit_expected"]==q["source_commit"]==commit
assert p["candidate_sha_expected"]==sha
assert all(p[layer+"_candidate_sha"]==q["candidate_sha256_"+layer]==sha for layer in ("l0","l1"))
assert p["foreign_l1"]==[{"name":"bw-s25-l1a","status":"Running","uuid":"21767890-6d46-4867-889d-0069c1e4ac0d"}]
assert q["l0_native"]==[["bw-s25-l1a","Running","21767890-6d46-4867-889d-0069c1e4ac0d"]]
assert q["l1_native"]==[]
assert all(q[layer+"_main_service"]==q[layer+"_candidate_service"]=="active" for layer in ("l0","l1"))
sign={}
for layer in ("l0","l1"):
  names={"hgsafe-"+layer+"-1","hgsafe-"+layer+"-2"}
  native=p[layer+"_native"]
  assert set(native)==names
  ownpre={x["name"]:x for x in p[layer+"_ledger"]["owned"]}
  ownpost={x["name"]:x for x in q[layer+"_ledger"]["owned"]}
  assert set(ownpre)==set(ownpost)==names
  uuids=set()
  for name in names:
    assert native[name]["uuid"]==ownpre[name]["uuid"]==ownpost[name]["uuid"]
    assert native[name]["status"] in ("Running","Ready")
    assert ownpre[name]["state"]=="OWNED" and ownpost[name]["state"]=="DELETED"
    n=native[name]["nic"]
    assert n["security.acls"] and n["security.acls.default.egress.action"]=="reject" and n["security.acls.default.ingress.action"]=="reject"
    assert all(n[x]=="true" for x in ("security.ipv4_filtering","security.ipv6_filtering","security.mac_filtering"))
    uuids.add(ownpre[name]["uuid"])
  orig={x["id"]:x for x in p[layer+"_ledger"]["operations"] if x["box_uuid"] in uuids}
  later={x["id"]:x for x in q[layer+"_ledger"]["ops"]}
  assert len(orig)==12 and len(later)==16 and set(orig).issubset(later)
  assert collections.Counter(x["action"] for x in orig.values())=={"box_create":2,"box_update":4,"box_exec":6}
  assert collections.Counter(x["action"] for x in later.values())=={"box_create":2,"box_update":4,"box_exec":6,"box_stop":2,"box_delete":2}
  for id,x in orig.items():
    assert all(x[k]==later[id][k] for k in ("action","state","box_uuid","native_operation_id"))
  assert all(x["state"]=="SUCCEEDED" and x["native_operation_id"] for x in later.values())
  exits=[p[layer+"_ledger"]["exec_exits"][x["id"]] for x in orig.values() if x["action"]=="box_exec"]
  assert collections.Counter(exits)=={28:5,0:1}
  assert q[layer+"_ledger"]["unknown_count"]==0
  s=p[layer+"_spoof_udp"]
  for family in ("4","6"):
    for phase in ("BEFORE","AFTER"):
      assert "HSWITNESS_"+layer.upper()+"_"+layer.upper()+"_VALID"+family+"_"+phase+" FROM" in s
  assert not any("HSWITNESS_"+layer.upper()+"_"+layer.upper()+"_FORGED"+x+" FROM" in s for x in ("4","6","_MAC"))
  for i in (1,2):
    receiver=p[layer+"_peer_"+str(i)]
    assert "HS_PEER_"+layer.upper()+"G"+str(i)+"_READY" in receiver
    assert "HS_PEER_"+layer.upper()+"G"+str(i)+"_RECEIVED_" not in receiver
    for ip in ("V4","V6"):
      assert "HSWITNESS_"+layer.upper()+"P_HOST_"+layer.upper()+"G"+str(i)+"_"+ip+" FROM" in p[layer+"_peer_host"]
  sign[layer]={"original_ops_before":12,"original_ops_after":16,"owned_guest_count":2,
      "native_post_deleted":True,"spoof_v4_v6_mac_rejected":True,"sibling_bidir_v4_v6_rejected":True,
      "identical_binary_sha256":sha}
for a in ("L0OWNED","L0FOREIGN"):
  for ip in ("V4","V6"):
    assert "HSWITNESS_L0F_HOST_"+a+"_"+ip+" FROM" in p["l0_foreign_host"]
for who in ("owned","other"):
  log=p["l0_foreign_"+who]
  assert "_READY" in log and "_RECEIVED_" not in log
print("HSAFE_DISTINCT_VERIFIER_BOX_L0_L1_ORIGINAL_OP_UUID_NIC_PACKET_POSTDELETE=PASS")
print("HSAFE_DISTINCT_VERIFIER_BOX_L0_FOREIGN_BOX_BIDIR_IPV4_IPV6=PASS")
res={
 "independent_verifier_box":"boxweave-test-ser2",
 "source_test_execution_box":"boxweave-test-cli",
 "candidate_source":commit,
 "candidate_binary_sha256":sha,
 "pre_snapshot_sha256":files["predelete"]["sha256"],
 "post_snapshot_sha256":files["postdelete"]["sha256"],
 "scope":"same-binary installed SDK+native+separate-Box verifier, L0/L1 spoof and sibling; L0 foreign Box",
 "formal_all_BH_TC_signoff":False,
 "layers":sign,
 "l0_foreign_box_bidir_pass":True,
 "status":"SCOPED_INDEPENDENT_V_PASS"
}
(d/"hgsafe-signed-review.json").write_text(json.dumps(res,indent=2,sort_keys=True)+"\n")
print("HSAFE_REVIEW_JSON_SHA256",hashlib.sha256((d/"hgsafe-signed-review.json").read_bytes()).hexdigest())

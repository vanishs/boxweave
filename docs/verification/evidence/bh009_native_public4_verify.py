#!/usr/bin/env python3
"""Read-only forensic verification of original isolated native public-v4 evidence."""
import hashlib, ipaddress, json
from pathlib import Path
home=Path(__file__).resolve().parent
acl=json.loads((home/"bh009-native-public4-acl-pre.json").read_text())
guest=json.loads((home/"bh009-native-public4-guest-pre.json").read_text())
positive=(home/"bh009-native-public4-positive.txt").read_text().strip()
assert acl["name"]=="bw-r12-public4-acl" and len(acl["ingress"])==0
rules=acl["egress"]
assert len(rules)==2
assert all(r["action"]=="reject" and r["state"]=="enabled" for r in rules)
assert [hashlib.sha256(r["destination"].encode()).hexdigest() for r in rules]==[
"d9bba25746c1923e107b93a5fd453b50fc074246d2e364c870758dc4fce2e228",
"cd16a2e4a8c121ef887c9cf64b6a80f4b28ad95954a11e816481ea2bcce221dd"]
v4=[ipaddress.ip_network(x) for x in rules[0]["destination"].split(",")]
v6=[ipaddress.ip_network(x) for x in rules[1]["destination"].split(",")]
for ips,missing in ((v4,("10.225.75.1","1.1.1.1")),(v6,("fd42:be:ce:40::1","fe80::1"))):
 for ip in missing:
  assert not any(ipaddress.ip_address(ip) in n for n in ips),(ip,ips)
for ips,forbidden in ((v4,("10.225.75.2","10.99.12.2","192.168.2.2","100.64.3.4")),
                      (v6,("fd42:be:ce:40::2","fd42:abba::2"))):
 for ip in forbidden:
  assert sum(ipaddress.ip_address(ip) in n for n in ips)==1,ip
assert len(guest)==1
inst=guest[0]
assert inst["name"]=="bw-r12-public4-a" and inst["status"]=="Running"
assert inst["config"]["volatile.uuid"]=="6f51f039-6156-49b4-aeb4-b3f79c081974"
nic=inst["expanded_devices"]["eth0"]
assert nic["network"]=="bw-r12p4br" and nic["security.acls"]=="bw-r12-public4-acl"
assert nic["security.acls.default.egress.action"]=="allow"
assert nic["security.acls.default.ingress.action"]=="reject"
for key in ("ipv4","ipv6","mac"):
 assert nic["security."+key+"_filtering"]=="true"
assert positive.endswith("status=200"),positive
remote=ipaddress.ip_address(positive.split("remote=")[1].split()[0])
assert isinstance(remote,ipaddress.IPv4Address) and remote.is_global
assert remote not in ipaddress.ip_network("100.64.0.0/10")
assert remote not in ipaddress.ip_network("198.18.0.0/15")
print("BH009_NATIVE_DISPOSABLE_EXACT_ACL_REAL_PUBLIC_IPV4_SNAPSHOT_V=PASS")
print("SCOPE: N+independent V; integrated latest MCP I direct-public-ipv4 was BLOCKED in GitHub runner")

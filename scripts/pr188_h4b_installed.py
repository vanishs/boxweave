#!/usr/bin/env python3
"""Read-only candidate capacity parity on independently installed CI fixtures."""
import json
import subprocess

def level(limit, remaining):
    return "EXHAUSTED" if not remaining else "LOW" if remaining*5<=limit else "AVAILABLE"

def check(status, doctor, scoped):
    for key in ("project_capacity","address_capacity"):
        assert status[key]==doctor[key],(key,status[key],doctor[key])
    a=status["address_capacity"]
    assert a["ipv4_prefix"] and a["blocked_ipv4"]+a["remaining_ipv4_upper_bound"]==a["nominal_ipv4"],a
    assert a["durable_held_ipv4"]+a["native_only_blocked_ipv4"]==a["blocked_ipv4"],a
    assert a["level"]==level(a["nominal_ipv4"],a["remaining_ipv4_upper_bound"]),a
    p=status["project_capacity"]
    assert p["remaining"]==max(0,p["limit"]-p["owned"]-p["pending_creates"])
    assert p["level"]==level(p["limit"],p["remaining"])
    if scoped:
        for key in ("scoped_identity_capacity","scoped_peer_capacity"):
            assert status[key]==doctor[key],(key,status[key],doctor[key])
        i=status["scoped_identity_capacity"];g=status["scoped_peer_capacity"]
        assert i["remaining_new_identities"]==i["limit"]-i["retired"]-i["owned"]-i["pending_creates"],i
        assert i["level"]==level(i["limit"],i["remaining_new_identities"])
        assert g["retained_directions"]==g["active_directions"]+g["revoked_directions"]+g["pending_or_unknown"],g
        assert g["remaining_directions"]==g["direction_limit"]-g["retained_directions"],g
        assert g["direction_level"]==level(g["direction_limit"],g["remaining_directions"])
        assert g["reserved_revoke_events"]==g["active_directions"],g
        assert g["remaining_grant_events_upper_bound"]==max(0,(g["history_limit"]-g["completed_ledger_events"]-g["active_directions"])//2),g
        assert g["grant_history_level"]==level(g["history_limit"]//2,g["remaining_grant_events_upper_bound"])
        r=doctor["diagnostic"]["root_policy"]
        assert r["state"]=="OBSERVED" and r["kernel_guard"]=="OBSERVED"
        assert r["remaining_grant_events_upper_bound"]==g["remaining_grant_events_upper_bound"],(r,g)
    else:
        assert "scoped_identity_capacity" not in status and "scoped_peer_capacity" not in status

def live_check(c, project, expected_owned, expected_retired):
    status=c.host_status()
    doctor=json.loads(subprocess.check_output(["/usr/local/bin/boxweave","doctor","-project",project],text=True,timeout=60))
    check(status,doctor,True)
    i=status["scoped_identity_capacity"];a=status["address_capacity"]
    assert i["owned"]==expected_owned and i["retired"]==expected_retired,i
    # This controlled clean fixture has exactly one held IPv4 per identity;
    # native leases may overlap but are never counted twice.
    assert a["durable_held_ipv4"]==expected_owned+expected_retired,a
    return status

"""Immutable, actor-scoped E1 capture records inside the app Store.

Only an allowlisted fingerprint subset is persisted. No raw tool responses,
host config, secrets, command output or private paths are saved.
"""
import json
import re
import secrets
import time

from .domain import Fault
from .environment_observe import diff_fingerprints, _sha

SCHEMA = """
CREATE TABLE environment_captures (
  id TEXT PRIMARY KEY,
  actor TEXT NOT NULL,
  target_identity TEXT NOT NULL,
  provider_scope TEXT NOT NULL,
  captured REAL NOT NULL,
  fingerprint_sha256 TEXT NOT NULL,
  observation TEXT NOT NULL
);
CREATE INDEX environment_captures_actor_time
 ON environment_captures(actor,captured);
"""
CAPACITY=128
_DIGEST=re.compile(r"[0-9a-f]{64}\Z")
_ID=re.compile(r"[0-9a-f]{32}\Z")
_FIELDS=("identity_digest","state","cpu","memory","disk","network_tenant")

def _actor(value):
    if value!="operator":
        raise Fault("ENVIRONMENT_ACTOR_NOT_ALLOWED",403)
    return value

def _checked(report):
    if not isinstance(report,dict):
        raise Fault("ENVIRONMENT_CAPTURE_INVALID",422)
    keys=("target_identity_digest","provider_scope_digest","fingerprint_sha256")
    for name in keys:
        value=report.get(name)
        if not isinstance(value,str) or _DIGEST.fullmatch(value) is None:
            raise Fault("ENVIRONMENT_CAPTURE_INVALID",422)
    obs=report.get("observation")
    if not isinstance(obs,dict) or set(obs)!=set(_FIELDS) or obs["identity_digest"]!=report["target_identity_digest"]:
        raise Fault("ENVIRONMENT_CAPTURE_INVALID",422)
    if obs["state"] not in ("CONTAINER_STATE_RUNNING","CONTAINER_STATE_STOPPED"):
        raise Fault("ENVIRONMENT_CAPTURE_INVALID",422)
    for key in ("cpu","memory","disk","network_tenant"):
        value=obs[key]
        if value is not None and (not isinstance(value,str) or
             not 1<=len(value)<=80 or any(ord(ch)<32 for ch in value)):
            raise Fault("ENVIRONMENT_CAPTURE_INVALID",422)
    if report.get("observation_status") not in ("partial","complete"):
        raise Fault("ENVIRONMENT_CAPTURE_INVALID",422)
    complete=all(obs[k] is not None for k in ("cpu","memory","disk","network_tenant"))
    if (report["observation_status"]=="complete") != complete:
        raise Fault("ENVIRONMENT_CAPTURE_INVALID",422)
    if report["fingerprint_sha256"] != _sha(obs):
        raise Fault("ENVIRONMENT_CAPTURE_INVALID",422)
    # Canonical JSON is small and contains only the above reviewed values.
    safe={"target_identity_digest":report["target_identity_digest"],
          "provider_scope_digest":report["provider_scope_digest"],
          "fingerprint_sha256":report["fingerprint_sha256"],
          "observation_status":report["observation_status"],
          "observation":{name:obs[name] for name in _FIELDS}}
    payload=json.dumps(safe,sort_keys=True,separators=(",",":"))
    if len(payload)>2048:
        raise Fault("ENVIRONMENT_CAPTURE_INVALID",422)
    return safe,payload

def save_capture(store, report, actor="operator", clock=time.time):
    actor=_actor(actor)
    safe,payload=_checked(report)
    ident=secrets.token_hex(16)
    when=clock()
    with store.db() as db:
        db.execute("BEGIN IMMEDIATE")
        if db.execute("SELECT COUNT(*) FROM environment_captures WHERE actor=?",
                      (actor,)).fetchone()[0]>=CAPACITY:
            raise Fault("ENVIRONMENT_CAPTURE_CAPACITY",409)
        db.execute("INSERT INTO environment_captures VALUES(?,?,?,?,?,?,?)",
                   (ident,actor,safe["target_identity_digest"],
                    safe["provider_scope_digest"],when,
                    safe["fingerprint_sha256"],payload))
    return read_capture(store,ident,actor)

def read_capture(store, ident, actor="operator"):
    actor=_actor(actor)
    if not isinstance(ident,str) or _ID.fullmatch(ident) is None:
        raise Fault("ENVIRONMENT_CAPTURE_NOT_FOUND",404)
    with store.db() as db:
        row=db.execute("SELECT id,captured,observation,target_identity,provider_scope,fingerprint_sha256 "
                       "FROM environment_captures WHERE id=? AND actor=?",(ident,actor)).fetchone()
    if row is None:raise Fault("ENVIRONMENT_CAPTURE_NOT_FOUND",404)
    try:
        result=json.loads(row["observation"])
        safe,_=_checked(result)
        if (safe["target_identity_digest"]!=row["target_identity"] or
                safe["provider_scope_digest"]!=row["provider_scope"] or
                safe["fingerprint_sha256"]!=row["fingerprint_sha256"]):
            raise Fault("ENVIRONMENT_CAPTURE_UNVERIFIED")
    except (ValueError,TypeError,KeyError,Fault) as error:
        if isinstance(error,Fault) and error.code=="ENVIRONMENT_CAPTURE_UNVERIFIED":
            raise
        raise Fault("ENVIRONMENT_CAPTURE_UNVERIFIED") from None
    return {"capture_id":row["id"],"captured_at":row["captured"],**safe}

def list_captures(store, actor="operator", target_identity=None, limit=20):
    actor=_actor(actor)
    if type(limit) is not int or not 1<=limit<=50:
        raise Fault("ENVIRONMENT_CAPTURE_LIMIT_INVALID",422)
    if target_identity is not None and (not isinstance(target_identity,str) or
                                        _DIGEST.fullmatch(target_identity) is None):
        raise Fault("ENVIRONMENT_CAPTURE_SELECTOR_INVALID",422)
    with store.db() as db:
        if target_identity is None:
            rows=db.execute("SELECT id,captured,target_identity,fingerprint_sha256 "
                            "FROM environment_captures WHERE actor=? "
                            "ORDER BY captured DESC,id LIMIT ?",(actor,limit)).fetchall()
        else:
            rows=db.execute("SELECT id,captured,target_identity,fingerprint_sha256 "
                            "FROM environment_captures WHERE actor=? AND target_identity=? "
                            "ORDER BY captured DESC,id LIMIT ?",(actor,target_identity,limit)).fetchall()
    return [{"capture_id":r["id"],"captured_at":r["captured"],
             "target_identity_digest":r["target_identity"],
             "fingerprint_sha256":r["fingerprint_sha256"]} for r in rows]

def compare_captures(store, left, right, actor="operator"):
    first=read_capture(store,left,actor)
    second=read_capture(store,right,actor)
    result=diff_fingerprints(first,second)
    return {"left":first["capture_id"],"right":second["capture_id"],**result}

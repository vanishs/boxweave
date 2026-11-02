"""Bounded optimistic same-file replacement for an exact trusted Box.

A standalone stdlib-only helper. This is a guest-side primitive, NOT a
cross-Agent dispatcher: callers must pass through durable claims and
authorization before invoking the generated connect(exec) command.
"""
import base64
import hashlib
import json
import re
import shlex
if __package__:
    from . import box_reads
else:
    import box_reads

MAX_PATCH_BYTES = 1024
_HEX = re.compile(r"[0-9a-f]{64}\Z")

# This script is sent only as a fixed, reviewed string, never from user code.
_GUEST_PATCH = r'''
import base64,fcntl,hashlib,json,os,secrets,stat,sys
v=json.loads(base64.urlsafe_b64decode(sys.argv[1]))
parts=v["path"].split("/")
fd=os.open("/home/"+v["box"],os.O_RDONLY|os.O_DIRECTORY|os.O_NOFOLLOW)
tmp=None
try:
    for part in parts[:-1]:
        child=os.open(part,os.O_RDONLY|os.O_DIRECTORY|os.O_NOFOLLOW,dir_fd=fd)
        os.close(fd);fd=child
    leaf=parts[-1]
    handle=os.open(leaf,os.O_RDONLY|os.O_NOFOLLOW|os.O_NONBLOCK,dir_fd=fd)
    try:
        fcntl.flock(handle,fcntl.LOCK_EX)
        info=os.fstat(handle)
        if not stat.S_ISREG(info.st_mode) or info.st_nlink!=1 or info.st_size>1024:
            raise ValueError("BOX_PATCH_FILE_UNSAFE")
        old=os.read(handle,1025)
        if hashlib.sha256(old).hexdigest()!=v["expected_sha256"]:
            raise ValueError("BOX_PATCH_STALE")
        current=os.stat(leaf,dir_fd=fd,follow_symlinks=False)
        if (current.st_ino,current.st_dev)!=(info.st_ino,info.st_dev):
            raise ValueError("BOX_PATCH_CHANGED")
        payload=base64.b64decode(v["data"],validate=True)
        if len(payload)>1024:raise ValueError("BOX_PATCH_TOO_LARGE")
        tmp=".boxweave-patch-"+secrets.token_hex(12)
        new_fd=os.open(tmp,os.O_WRONLY|os.O_CREAT|os.O_EXCL|os.O_NOFOLLOW,
                       stat.S_IMODE(info.st_mode)&0o600,dir_fd=fd)
        try:
            remaining=payload
        while remaining:
            remaining=remaining[os.write(new_fd,remaining):]
        os.fsync(new_fd)
        finally:os.close(new_fd)
        current=os.stat(leaf,dir_fd=fd,follow_symlinks=False)
        if (current.st_ino,current.st_dev,current.st_size)!=(info.st_ino,info.st_dev,info.st_size):
            raise ValueError("BOX_PATCH_CHANGED")
        os.replace(tmp,leaf,src_dir_fd=fd,dst_dir_fd=fd)
        tmp=None
        os.fsync(fd)
        print(json.dumps({"state":"replaced","sha256":hashlib.sha256(payload).hexdigest(),
                          "bytes":len(payload)},separators=(",",":")))
    finally:os.close(handle)
finally:
    if tmp is not None:
        try:os.unlink(tmp,dir_fd=fd)
        except OSError:pass
    os.close(fd)
'''

def verify_patch(args):
    if not isinstance(args,dict) or set(args)!={"box","path","expected_sha256","new_text","request_key"}:
        raise ValueError("BOX_PATCH_INVALID")
    checked=box_reads.verified_args("boxweave_box_read_file",
                                    {"box":args["box"],"path":args["path"]})
    text=args["new_text"]
    if not isinstance(text,str) or len(text.encode("utf-8"))>MAX_PATCH_BYTES:
        raise ValueError("BOX_PATCH_TOO_LARGE")
    if not isinstance(args["expected_sha256"],str) or _HEX.fullmatch(args["expected_sha256"]) is None:
        raise ValueError("BOX_PATCH_INVALID")
    if not isinstance(args["request_key"],str) or re.fullmatch(r"[0-9a-f]{32}",args["request_key"]) is None:
        raise ValueError("BOX_PATCH_INVALID")
    return {**checked,"action":"patch","expected_sha256":args["expected_sha256"],
            "data":base64.b64encode(text.encode("utf-8")).decode()}

def guest_command(checked):
    payload=base64.urlsafe_b64encode(json.dumps(checked,separators=(",",":")).encode()).decode()
    return "python3 -c "+shlex.quote(_GUEST_PATCH)+" "+shlex.quote(payload)

def parse_receipt(response):
    result=response.get("result",{})
    if not isinstance(result,dict) or result.get("isError"):
        raise ValueError("BOX_PATCH_RESULT_UNVERIFIED")
    structured=result.get("structuredContent")
    if isinstance(structured,dict) and type(structured.get("exit_code")) is int:
        code,stdout=structured["exit_code"],structured.get("stdout")
    else:
        chunks=result.get("content",[])
        if not isinstance(chunks,list) or len(chunks)!=1 or not isinstance(chunks[0],dict):
            raise ValueError("BOX_PATCH_RESULT_UNVERIFIED")
        raw=chunks[0].get("text")
        match=re.fullmatch(r"exit_code: (-?\d+)\n\n--- stdout ---\n([\s\S]{0,2048})",raw if isinstance(raw,str) else "")
        if match is None:raise ValueError("BOX_PATCH_RESULT_UNVERIFIED")
        code,stdout=int(match[1]),match[2]
    if code!=0 or not isinstance(stdout,str) or len(stdout)>2048:
        raise ValueError("BOX_PATCH_RESULT_UNVERIFIED")
    try:
        obj=json.loads(stdout)
    except (TypeError,ValueError):
        raise ValueError("BOX_PATCH_RESULT_UNVERIFIED") from None
    if not isinstance(obj,dict) or set(obj)!={"state","sha256","bytes"} or obj["state"]!="replaced":
        raise ValueError("BOX_PATCH_RESULT_UNVERIFIED")
    if not isinstance(obj["sha256"],str) or _HEX.fullmatch(obj["sha256"]) is None:
        raise ValueError("BOX_PATCH_RESULT_UNVERIFIED")
    if type(obj["bytes"]) is not int or not 0<=obj["bytes"]<=MAX_PATCH_BYTES:
        raise ValueError("BOX_PATCH_RESULT_UNVERIFIED")
    return obj

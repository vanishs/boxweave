"""Bounded read-only Box operations via the EXISTING reviewed connect(exec) path.

This file is installed alongside mcp_gate.py and uses only the stdlib.
The guest has no resident helper. A root is a fixed verified home directory,
not an arbitrary filesystem confinement or shell privilege boundary.
"""
import base64
import json
import re
import shlex

TOOLS = (
    {"name":"boxweave_box_read_file",
     "description":"Read up to 1024 bytes from a regular file using a home-relative path (no leading / or ..) under the verified Box user's home. Symlinks are denied. Requires guest python3 and existing connect permission.",
     "inputSchema":{"type":"object","additionalProperties":False,
       "properties":{"box":{"type":"string"},"path":{"type":"string"},
                     "offset":{"type":"integer","minimum":0},
                     "max_bytes":{"type":"integer","minimum":1,"maximum":1024}},
       "required":["box","path"]},"annotations":{"readOnlyHint":True,"destructiveHint":False}},
    {"name":"boxweave_box_list_directory",
     "description":"List up to 32 entries from a directory using a home-relative path (empty path selects home; no leading / or ..). Symlinks and recursive walks are denied.",
     "inputSchema":{"type":"object","additionalProperties":False,
       "properties":{"box":{"type":"string"},"path":{"type":"string"},
                     "offset":{"type":"integer","minimum":0},
                     "limit":{"type":"integer","minimum":1,"maximum":32}},
       "required":["box","path"]},"annotations":{"readOnlyHint":True,"destructiveHint":False}},
)

# The exact guest script is fixed, with only bounded base64-encoded JSON input.
_GUEST = r'''
import os,sys,base64,json,stat,hashlib
v=json.loads(base64.urlsafe_b64decode(sys.argv[1]))
parts=v["path"].split("/") if v["path"] else []
fd=os.open("/home/"+v["box"],os.O_RDONLY|os.O_DIRECTORY|os.O_NOFOLLOW)
try:
    dirs=parts if v["action"]=="list" else parts[:-1]
    for seg in dirs:
        nxt=os.open(seg,os.O_RDONLY|os.O_DIRECTORY|os.O_NOFOLLOW,dir_fd=fd)
        os.close(fd);fd=nxt
    if v["action"]=="list":
        names=sorted(os.listdir(fd))
        start=v["offset"];chunk=names[start:start+v["limit"]]
        out={"entries":[{"name":n,"kind":"directory" if stat.S_ISDIR(os.stat(n,dir_fd=fd,follow_symlinks=False).st_mode) else "file" if stat.S_ISREG(os.stat(n,dir_fd=fd,follow_symlinks=False).st_mode) else "other"} for n in chunk],
             "next_offset":start+len(chunk) if start+len(chunk)<len(names) else None}
    else:
        f=os.open(parts[-1],os.O_RDONLY|os.O_NOFOLLOW|os.O_NONBLOCK,dir_fd=fd)
        try:
            st=os.fstat(f)
            if not stat.S_ISREG(st.st_mode):raise ValueError("NOT_REGULAR_FILE")
            os.lseek(f,v["offset"],os.SEEK_SET)
            data=os.read(f,v["max_bytes"])
            out={"encoding":"base64","data":base64.b64encode(data).decode(),
                 "sha256":hashlib.sha256(data).hexdigest(),"offset":v["offset"],
                 "eof":v["offset"]+len(data)>=st.st_size,"size":st.st_size}
        finally:os.close(f)
    print(json.dumps(out,separators=(",",":")))
finally:os.close(fd)
'''

def verified_args(tool, args):
    if tool not in {t["name"] for t in TOOLS} or not isinstance(args,dict):
        raise ValueError("BOX_READ_INVALID")
    action="list" if tool=="boxweave_box_list_directory" else "read"
    expected={"box","path"}|({"offset","limit"} if action=="list" else {"offset","max_bytes"})
    if set(args)-expected or not {"box","path"} <= set(args):
        raise ValueError("BOX_READ_INVALID")
    box,path=args["box"],args["path"]
    if not isinstance(box,str) or re.fullmatch(r"[a-z][a-z0-9-]{0,62}",box) is None:
        raise ValueError("BOX_READ_INVALID")
    if (not isinstance(path,str) or len(path)>240 or path.startswith("/")
            or any(seg in ("", ".", "..") or len(seg)>100 or
                   any(ord(c)<32 for c in seg) for seg in path.split("/"))):
        if not (action=="list" and path==""):
            raise ValueError("BOX_PATH_INVALID")
    offset=args.get("offset",0)
    if type(offset) is not int or not 0<=offset<=1000000:
        raise ValueError("BOX_READ_INVALID")
    field="limit" if action=="list" else "max_bytes"
    value=args.get(field,32 if action=="list" else 1024)
    if type(value) is not int or not 1<=value<=(32 if action=="list" else 1024):
        raise ValueError("BOX_READ_INVALID")
    return {"action":action,"box":box,"path":path,"offset":offset,field:value}

def guest_command(verified):
    payload=base64.urlsafe_b64encode(json.dumps(verified,separators=(",",":")).encode()).decode()
    return "python3 -c "+shlex.quote(_GUEST)+" "+shlex.quote(payload)

def typed_output(message):
    """Decode only an observed completed exit-0 response; never leak arbitrary stderr."""
    result=message.get("result",{})
    if not isinstance(result,dict) or result.get("isError"):
        raise ValueError("BOX_READ_UNVERIFIED")
    struct=result.get("structuredContent")
    if isinstance(struct,dict) and type(struct.get("exit_code")) is int and isinstance(struct.get("stdout"),str):
        code,output=struct["exit_code"],struct["stdout"]
    else:
        text="\n".join(part.get("text","") for part in result.get("content",[]) if isinstance(part,dict))
        m=re.fullmatch(r"exit_code: (-?\d+)\n\n--- stdout ---\n([\s\S]{0,8192})",text)
        if not m:raise ValueError("BOX_READ_UNVERIFIED")
        code,output=int(m[1]),m[2]
    if code!=0 or len(output)>8192:raise ValueError("BOX_READ_UNVERIFIED")
    try:
        obj=json.loads(output)
    except (ValueError,TypeError):
        raise ValueError("BOX_READ_UNVERIFIED") from None
    if not isinstance(obj,dict) or not ({"entries","next_offset"}<=set(obj) or {"data","sha256","size","eof"}<=set(obj)):
        raise ValueError("BOX_READ_UNVERIFIED")
    return obj

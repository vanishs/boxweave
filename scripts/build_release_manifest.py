"""Assemble public installer assets and a source-bound release manifest after validation."""
import argparse,hashlib,json,re,shutil
from pathlib import Path

def build(root,output,source):
    root=Path(root);output=Path(output)
    if re.fullmatch("[a-f0-9]{40}",source) is None:raise ValueError("exact source commit required")
    for name in ("bootstrap.py","distribution.py","online.py","hostkeys.py","mcp_gate.py","nesting.py","client_setup.py"):
        shutil.copyfile(root/"bwmini"/name,output/name)
    shutil.copyfile(root/"install.sh",output/"install.sh")
    names=("boxweave.zip","bootstrap.py","distribution.py","online.py","hostkeys.py","mcp_gate.py","nesting.py","client_setup.py")
    manifest={"format":1,"version":(root/"VERSION").read_text().strip(),"source":source,
              "files":{name:hashlib.sha256((output/name).read_bytes()).hexdigest() for name in names}}
    (output/"release.json").write_text(json.dumps(manifest,sort_keys=True,indent=2)+"\n")
    (output/"SHA256SUMS").write_text("".join(hashlib.sha256(p.read_bytes()).hexdigest()+"  "+p.name+"\n"
                                          for p in sorted(output.iterdir()) if p.is_file() and p.name!="SHA256SUMS"))
    return manifest
if __name__=="__main__":
    p=argparse.ArgumentParser(description=__doc__);p.add_argument("--root",required=True);p.add_argument("--output",required=True);p.add_argument("--source",required=True);a=p.parse_args()
    print(json.dumps(build(a.root,a.output,a.source)))

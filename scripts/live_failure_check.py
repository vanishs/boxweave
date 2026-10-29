"""Live trust/identity failure gates; no hello after mismatch."""
import argparse
import dataclasses
import json
from pathlib import Path
import subprocess
from bwmini.domain import Fault
from bwmini.mcp import MCP
from bwmini.provider import Provider
def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--config",required=True)
    parser.add_argument("--pin-file",required=True)
    args=parser.parse_args()
    config=Path(args.config)
    c=json.loads(config.read_text())
    pin=Path(args.pin_file).resolve()
    before=pin.read_bytes()
    wrong=pin.with_name("deliberately-wrong-known-hosts")
    # Deliberately different public key; never replace the normal trusted pin.
    normal=pin.read_text().split()
    key=normal[2]
    replacement=key[:-1]+("A" if key[-1]!="A" else "B")
    wrong.write_text(normal[0]+" "+normal[1]+" "+replacement+"\n");wrong.chmod(0o600)
    argv=[part.replace("UserKnownHostsFile="+str(pin),"UserKnownHostsFile="+str(wrong)) for part in c["launcher"]]
    result=subprocess.run(argv,input=b"",stdout=subprocess.PIPE,stderr=subprocess.PIPE,timeout=20)
    assert result.returncode==255 and not result.stdout
    assert b"HOST IDENTIFICATION HAS CHANGED" in result.stderr
    assert pin.read_bytes()==before
    calls=[]
    class Observed(MCP):
        def tool(self,name,arguments):
            calls.append((name,arguments))
            return super().tool(name,arguments)
    class Checked(Provider):
        def session(self,timeout=30):
            return Observed(self.argv,min(30,timeout))
    p=Checked(c)
    logical=next(iter(c["targets"]))
    target=p.observe(logical);calls.clear()
    try:
        p.hello(dataclasses.replace(target,hostname="wrong-test-host"),"/home/"+target.user,30)
        raise AssertionError("Workload identity accepted")
    except Fault as error:
        assert error.code=="IDENTITY_MISMATCH"
    assert len(calls)==1 and calls[0][1].get("exec")=="whoami && hostname"
    summary={"result":"passed","host_pin":{"ssh_exit":255,"normal_pin_unchanged":True,"protocol_not_started":True},
             "workload_identity":{"code":"IDENTITY_MISMATCH","identity_calls":1,"hello_calls":0}}
    config.with_name("failure-result.json").write_text(json.dumps(summary,indent=2))
    print(json.dumps(summary))
if __name__=="__main__":main()

"""Public client-descriptor generation only; never reads credentials or launches a process."""
import base64,hashlib,re,json
from pathlib import Path,PureWindowsPath,PurePosixPath
from .domain import Fault
PROFILES=('codex/windows/native','codex/windows/wsl','zcode/windows/native','zcode/windows/wsl')
FIELDS={'profile','agent_executable','agent_core','agent_version','profile_directory','host','port','user','identity_file','known_hosts','server_key','transport_executable','distribution','builtin_path','builtin_sha256','replace'}
def bad():raise Fault('INVALID_PUBLIC_PROFILE',422)
def text(value):
 if not isinstance(value,str) or len(value)>1024 or any(ord(x)<32 for x in value):bad()
 if re.search(r'-----BEGIN|eyJ[A-Za-z0-9_-]{20,}\.|(?:api[_-]?key|token|password|secret)\s*[=:]|Bearer\s',value,re.I):bad()
 return value
def windows(value):
 value=text(value);p=PureWindowsPath(value)
 if not p.is_absolute() or not re.fullmatch('[A-Za-z]:',p.drive) or '..' in p.parts or any(c in value for c in '\"<>|&;$'):bad()
 return value
def posix(value):
 value=text(value);p=PurePosixPath(value)
 if not p.is_absolute() or '..' in p.parts or '\\' in value:bad()
 return value
def generate(data):
 if not isinstance(data,dict) or set(data)!=FIELDS:bad()
 for k,v in data.items():
  if k not in ('port','replace'):text(v)
 profile=data['profile']
 if profile not in PROFILES or type(data['port']) is not int or not 1<=data['port']<=65535 or type(data['replace']) is not bool:bad()
 if not re.fullmatch('[A-Za-z0-9.-]{1,253}',data['host']) or not re.fullmatch('[a-z_][a-z0-9_-]{0,62}',data['user']) or data['user']=='root':bad()
 if not re.fullmatch(r'[A-Za-z0-9 ._+()\-]{1,80}',data['agent_version']):bad()
 pin=data['server_key'].split()
 try:
  if len(pin)!=2 or pin[0]!='ssh-ed25519':bad()
  raw=base64.b64decode(pin[1],validate=True)
  if len(raw)!=51 or not raw.startswith(b'\0\0\0\x0bssh-ed25519\0\0\0\x20'):bad()
 except (ValueError,TypeError):bad()
 wsl=profile.endswith('/wsl');zcode=profile.startswith('zcode/')
 executable=windows(data['transport_executable']);agent=windows(data['agent_executable'])
 if not executable.lower().endswith('/wsl.exe') and not executable.lower().endswith('\\wsl.exe') and wsl:bad()
 if not wsl and not executable.lower().endswith(('ssh.exe',)):bad()
 if not zcode and not agent.lower().endswith('codex.exe'):bad()
 if zcode and not agent.lower().endswith('node.exe'):bad()
 argv=[agent]
 if zcode:
  argv.append(windows(data['agent_core']))
  if not argv[1].lower().endswith('zcode.cjs') or not re.fullmatch('[0-9a-f]{64}',data['builtin_sha256']):bad()
  builtin={'path':windows(data['builtin_path']),'sha256':data['builtin_sha256']}
 elif any(data[k] for k in ('agent_core','builtin_path','builtin_sha256')):bad()
 if wsl:
  if not re.fullmatch('[A-Za-z0-9_.-]{1,80}',data['distribution']):bad()
 elif data['distribution']:bad()
 helper=Path(__file__).resolve().parents[1]/'clients/profile_client.py'
 connection={'host':data['host'],'port':data['port'],'user':data['user'],'identity_file':(posix if wsl else windows)(data['identity_file']),'known_hosts':(posix if wsl else windows)(data['known_hosts']),'server_key':data['server_key'],'executable':executable}
 if wsl:connection['distribution']=data['distribution']
 descriptor={'schema':2,'profile':profile,'source_sha256':hashlib.sha256(helper.read_bytes()).hexdigest(),'agent_argv':argv,'agent_version':data['agent_version'],'profile_directory':windows(data['profile_directory']),'connection':connection}
 if zcode:descriptor['zcode_builtin']=builtin
 descriptor_text=json.dumps(descriptor,sort_keys=True,indent=2)+'\n'
 descriptor_hash=hashlib.sha256(descriptor_text.encode()).hexdigest()
 script="param([Parameter(Mandatory=$true)][string]$PythonExecutable,[Parameter(Mandatory=$true)][string]$ProfileClientPath)\n$ErrorActionPreference='Stop'\nif(-not [IO.Path]::IsPathRooted($PythonExecutable) -or -not [IO.Path]::IsPathRooted($ProfileClientPath)){throw 'Absolute executable and helper paths are required'}\n$descriptor=Join-Path $PSScriptRoot 'boxweave-client.json'\nif((Get-FileHash -LiteralPath $ProfileClientPath -Algorithm SHA256).Hash.ToLowerInvariant() -ne 'HELPER_SHA'){throw 'PAYLOAD_MISMATCH'}\nif((Get-FileHash -LiteralPath $descriptor -Algorithm SHA256).Hash.ToLowerInvariant() -ne 'DESCRIPTOR_SHA'){throw 'DESCRIPTOR_MISMATCH'}\n& $PythonExecutable $ProfileClientPath register --descriptor $descriptor REPLACE_FLAG\nexit $LASTEXITCODE\n"
 script=script.replace('HELPER_SHA',descriptor['source_sha256']).replace('DESCRIPTOR_SHA',descriptor_hash).replace('REPLACE_FLAG','--replace' if data['replace'] else '')
 return {'descriptor':descriptor,'descriptor_text':descriptor_text,'descriptor_sha256':descriptor_hash,'enrollment_script':script,'replace_selected':data['replace'],'registration_arguments':['register','--descriptor','<ABSOLUTE_DOWNLOADED_DESCRIPTOR_PATH>']+(['--replace'] if data['replace'] else []),'guidance':['Public descriptor only. Review the host pin and helper digest using an authenticated operator source.','Save at an absolute path. Use the independently reviewed clients/profile_client.py with the registration arguments.','Local identity/pin/ACL, exact Agent version, connection and registration gates still run on the selected Windows runtime. No Web-generated data bypasses them.'],'execution':'not_performed','credential_contents':'not_requested'}

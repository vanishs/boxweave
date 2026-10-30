"""Read-only documentation/traceability checks. Does not validate historical claims as current runs."""
import argparse,collections,json,re,sys,urllib.parse
from pathlib import Path
QUESTIONS=('goals','boundaries','run','design','tests','risks','legacy','state')
REQUIRED_ROLES={'product','requirements','acceptance','scope','architecture','decisions','tasks','traceability','sources','legacy','risks','status','handoff','maintenance','install','tests'}
def readme_narrative_errors(text):
 """Reject obsolete product/stage narratives, not ordinary technical words."""
 # Executable examples and exact identifiers retain their technical spelling.
 prose=re.sub(r"\x60\x60\x60[\s\S]*?\x60\x60\x60|~~~[\s\S]*?~~~", "", text)
 prose=re.sub(r"\x60[^\x60\n]*\x60", "", prose)
 patterns=(
  (r"\bphase[ \t-]*[1-7]\b", "README has an internal development-stage narrative"),
  (r"\b(?:archived?|legacy)[ \t]+(?:v)?0\.5\.1\b", "README has an archived-release narrative"),
  (r"\bboxweave[ \t]+v2\b|\b(?:this[ \t]+is[ \t]+(?:the[ \t]+)?|our[ \t]+|current[ \t]+)v2[ \t]+(?:app(?:lication)?|implementation|rewrite)\b",
   "README brands the current product as v2 instead of BoxWeave 0.X.X"),
 )
 return [message for pattern,message in patterns if re.search(pattern,prose,re.I)]


def release_errors(root):
 """Check public release metadata when this is a real repository checkout."""
 import tomllib
 root=Path(root)
 if not (root/'VERSION').exists():return []
 errors=[]
 try:
  version=(root/'VERSION').read_text(encoding='utf-8').strip()
  project=tomllib.loads((root/'pyproject.toml').read_text(encoding='utf-8'))['project']['version']
  web=json.loads((root/'web/package.json').read_text(encoding='utf-8'))['version']
  lock=json.loads((root/'web/package-lock.json').read_text(encoding='utf-8'))
  if not re.fullmatch(r'0\.[0-9]+\.[0-9]+',version) or any(v!=version for v in (project,web,lock['version'],lock['packages']['']['version'])):
   errors.append('Release version mismatch across VERSION, Python project, Web package and lock root')
  readme=(root/'README.md').read_text(encoding='utf-8')
  errors.extend(readme_narrative_errors(readme))
  if version not in readme[:2000]:errors.append('README does not declare the current release version')
  if re.search(r'Phase\s*7.{0,70}(has not started|not started|eligible for)',readme[:3000],re.I):
   errors.append('README has obsolete historical phase claim')
  if not all(word in readme[:4500] for word in ('Windows','Codex','ZCode','WSL','macOS')):
   errors.append('README current selected client matrix/boundaries missing')
 except (OSError,KeyError,ValueError,TypeError) as error:
  errors.append('Release metadata missing or malformed')
 return errors

def audit(root):
 root=Path(root).resolve();errors=release_errors(root)
 def read(p):
  try:return p.read_text(encoding='utf-8')
  except (OSError,UnicodeError):errors.append('Unreadable document: '+str(p.relative_to(root)));return ''
 def target(base,value):
  value=urllib.parse.unquote(value.strip('<>').split('#')[0])
  if not value or '://' in value or value.startswith(('mailto:','app:','plugin:')):return None
  p=(base/value).resolve()
  if not p.is_relative_to(root):errors.append('Link leaves repository: '+value);return None
  return p
 registry=root/'docs/process/DOCUMENT_AUTHORITY.json'
 try:data=json.loads(read(registry))
 except ValueError:return {'result':'failed','errors':errors+['Invalid authority JSON']}
 authority=data.get('authority',{});cold=data.get('cold_start',{})
 if not REQUIRED_ROLES<=set(authority):errors.append('Missing required authority role')
 if len(set(authority.values()))!=len(authority):errors.append('Duplicate authority file')
 paths={}
 for role,name in authority.items():
  p=target(root,name)
  if p is None or not p.is_file():errors.append('Missing authority file: '+role)
  else:paths[role]=p
 documents=list((root/'docs').rglob('*.md'))+[root/'AGENTS.md',root/'README.md']
 graph={}
 for p in documents:
  links=[]
  for value in re.findall(r'\]\(([^)]+)\)',read(p)):
   q=target(p.parent,value)
   if q is None:continue
   if not q.exists():errors.append('Broken local link: '+str(p.relative_to(root))+' -> '+value)
   else:links.append(q)
  graph[p.resolve()]=links
 visited={root/'AGENTS.md',root/'docs/INDEX.md'};front=list(visited)
 for _ in range(3):
  front=[q for p in front for q in graph.get(p,[]) if q not in visited];visited.update(front)
 routes={}
 for question in QUESTIONS:
  role=cold.get(question);p=paths.get(role)
  if p is None or p not in visited:errors.append('Cold-start route missing: '+question)
  else:routes[question]=str(p.relative_to(root))
 req=[];ac=[];pairs=[]
 if all(x in paths for x in ('requirements','acceptance','traceability')):
  req=re.findall(r'^\|\s*(REQ-\d{3})\s*\|',read(paths['requirements']),re.M)
  ac=re.findall(r'^##\s+(AC-\d{3})\b',read(paths['acceptance']),re.M)
  pairs=re.findall(r'^\|\s*(REQ-\d{3})\s*/\s*(AC-\d{3})\s*\|',read(paths['traceability']),re.M)
  if not req or len(req)!=len(set(req)) or len(ac)!=len(set(ac)):errors.append('Missing or duplicate normative ID')
  if set(req)!={x for x,y in pairs} or set(ac)!={y for x,y in pairs}:errors.append('Requirement/acceptance traceability gap')
 return {'result':'failed' if errors else 'passed','errors':errors,'documents_checked':len(documents),'authority_roles':len(authority),'requirements':len(req),'acceptance':len(ac),'cold_start_routes':routes,'scope':'read-only local links/authority/ID coverage/navigation; not semantic proof or historical hash revalidation'}
def main():
 p=argparse.ArgumentParser(description=__doc__);p.add_argument('--root',type=Path,default=Path(__file__).resolve().parents[1]);a=p.parse_args();result=audit(a.root);print(json.dumps(result));return 0 if result['result']=='passed' else 1
if __name__=='__main__':raise SystemExit(main())

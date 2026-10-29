import React,{useState} from 'react';
type Api=(path:string,method?:string,data?:unknown)=>Promise<any>;
const initial={profile:'codex/windows/native',agent_executable:'',agent_core:'',agent_version:'',profile_directory:'',host:'',port:22,user:'',identity_file:'',known_hosts:'',server_key:'',transport_executable:'C:/Windows/System32/OpenSSH/ssh.exe',distribution:'',builtin_path:'',builtin_sha256:'',replace:false};
type Data=typeof initial;
export function ClientSetup({api}:{api:Api}){
 const [data,setData]=useState<Data>(initial),[result,setResult]=useState<any>(null),[error,setError]=useState(''),[busy,setBusy]=useState(false);
 const wsl=data.profile.endsWith('/wsl'),zcode=data.profile.startsWith('zcode/');
 function change(key:keyof Data,value:string|number|boolean){setData({...data,[key]:value});setResult(null);}
 function field(key:keyof Data,label:string,placeholder=''){return <label key={key}>{label}<input aria-label={label} value={String(data[key])} placeholder={placeholder} required onChange={e=>change(key,e.target.value)}/></label>;}
 function download(name:string,text:string){const url=URL.createObjectURL(new Blob([text],{type:'text/plain;charset=utf-8'}));const a=document.createElement('a');a.href=url;a.download=name;a.click();setTimeout(()=>URL.revokeObjectURL(url),1000);}
 return <section aria-label="Client setup"><h2>Client configuration</h2><p>Generate public configuration for your selected Windows Agent and SSH runtime. Enter paths and the public server key; never paste private keys, API keys or sign-in data. Nothing is installed or connected by this page.</p>
 {error&&<p role="alert">{error}</p>}
 <form onSubmit={e=>{e.preventDefault();setBusy(true);setError('');void api('client-profile','POST',data).then(setResult).catch(e=>setError(e.message)).finally(()=>setBusy(false));}}>
 <label>Client profile<select aria-label="Client profile" value={data.profile} onChange={e=>{const p=e.target.value;setData({...data,profile:p,transport_executable:p.endsWith('/wsl')?'C:/Windows/System32/wsl.exe':'C:/Windows/System32/OpenSSH/ssh.exe',distribution:p.endsWith('/wsl')?data.distribution:'',agent_executable:'',agent_core:'',builtin_path:'',builtin_sha256:''});setResult(null);}}>
 {['codex/windows/native','codex/windows/wsl','zcode/windows/native','zcode/windows/wsl'].map(p=><option key={p}>{p}</option>)}</select></label>
 {field('agent_executable',zcode?'Node executable':'Codex executable',zcode?'C:/Tools/node.exe':'C:/Tools/codex.exe')}
 {zcode&&field('agent_core','ZCode core script','C:/ZCode/resources/glm/zcode.cjs')}
 {field('agent_version','Exact Agent version')}
 {field('profile_directory','Dedicated profile directory','C:/My Agent profile')}
 {field('host','SSH host')}{field('user','SSH user')}
 <label>SSH port<input aria-label="SSH port" type="number" min={1} max={65535} required value={data.port} onChange={e=>change('port',Number(e.target.value))}/></label>
 {field('transport_executable','Transport executable')}
 {wsl&&field('distribution','Explicit WSL distribution','Ubuntu')}
 {field('identity_file','Private key path only',wsl?'/home/user/.ssh/id_ed25519':'C:/Keys/id_ed25519')}
 {field('known_hosts','Known-hosts path',wsl?'/home/user/.ssh/known_hosts':'C:/Keys/known_hosts')}
 {field('server_key','Public SSH server key','ssh-ed25519 ...')}
 {zcode&&<>{field('builtin_path','Public ZCode catalog path')}{field('builtin_sha256','Public catalog SHA256')}</>}
 <label><input type="checkbox" checked={data.replace} onChange={e=>change('replace',e.target.checked)}/>Allow replacement of the BoxWeave registration; preserve unrelated settings</label>
 <button disabled={busy}>Generate public setup</button></form>
 {result&&<div className="review"><h3>Review before downloading</h3><p>Replacement: {result.replace_selected?'explicitly selected':'disabled'}. Execution: {result.execution}.</p>
 <textarea aria-label="Generated public descriptor" readOnly rows={14} value={result.descriptor_text}/>
 {result.guidance.map((x:string)=><p key={x}>{x}</p>)}
 <p>Save both downloads under their exact names in one new private folder. Review them and independently verify the helper and host pin. The script requires explicit absolute Python and profile-client paths; it verifies both hashes before running the existing local validator.</p>
 <button onClick={()=>download('boxweave-client.json',result.descriptor_text)}>Download public descriptor</button>
 <button onClick={()=>download('Register-BoxWeave.ps1',result.enrollment_script)}>Download reviewed enrollment script</button></div>}
 </section>;
}

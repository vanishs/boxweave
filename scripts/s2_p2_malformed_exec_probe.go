//go:build ignore

// Official MCP client against an installed BoxWeave service and actual
// native guest. Only the test proxy mutates the observed native response.
package main

import (
 "context"
 "encoding/json"
 "fmt"
 "net/http"
 "os"
 "path/filepath"
 "strings"
 "time"

 "github.com/modelcontextprotocol/go-sdk/mcp"
)
type auth struct{token string}
func (a auth)RoundTrip(r *http.Request)(*http.Response,error){
 c:=r.Clone(r.Context());c.Header.Set("Authorization","Bearer "+a.token)
 return http.DefaultTransport.RoundTrip(c)
}
func check(ok bool,what string){if !ok{panic("S2_P2_REAL_MALFORMED_FAIL: "+what)}}
func field(m map[string]any,k string)string{v,_:=m[k].(string);return v}
func child(m map[string]any,k string)map[string]any{v,_:=m[k].(map[string]any);return v}
func conn(ctx context.Context,token string)*mcp.ClientSession{
 client:=mcp.NewClient(&mcp.Implementation{Name:"s2-p2-native-malformed-verifier",Version:"1"},nil)
 s,e:=client.Connect(ctx,&mcp.StreamableClientTransport{
  Endpoint:"http://127.0.0.1:18665/mcp",HTTPClient:&http.Client{Transport:auth{token}},
  DisableStandaloneSSE:true,MaxRetries:-1,
 },nil)
 check(e==nil&&s!=nil,"test local MCP unavailable");return s
}
func invoke(ctx context.Context,s *mcp.ClientSession,n string,a map[string]any)(map[string]any,error){
 r,e:=s.CallTool(ctx,&mcp.CallToolParams{Name:n,Arguments:a})
 if e!=nil{return nil,e};if r==nil||r.IsError{return nil,fmt.Errorf("native MCP tool refused: %s",n)}
 b,e:=json.Marshal(r.StructuredContent);if e!=nil{return nil,e}
 var v map[string]any;e=json.Unmarshal(b,&v);return v,e
}
type saved struct{BoxID string `json:"box_id"`;CreateID string `json:"create_id"`;ExecID string `json:"exec_id"`;NativeID string `json:"native_id"`}
func save(path string,v saved){b,e:=json.Marshal(v);check(e==nil&&os.WriteFile(path,b,0600)==nil,"cannot save exact IDs")}
func load(path string)saved{b,e:=os.ReadFile(path);check(e==nil,"cannot read original IDs");var v saved;check(json.Unmarshal(b,&v)==nil,"bad original IDs");return v}
func poll(ctx context.Context,s *mcp.ClientSession,id string,expected string)map[string]any{
 var op map[string]any
 for i:=0;i<105;i++{
  out,e:=invoke(ctx,s,"operation_get",map[string]any{"operation_id":id,"wait_ms":800})
  check(e==nil,"original receipt unreadable")
  op=child(out,"operation")
  check(op!=nil,"missing durable original operation")
  state:=field(op,"state")
  if state==expected{return op}
  check(state!="FAILED"&&state!="CANCELLED"&&state!="UNKNOWN"&&state!="SUCCEEDED",
   "original settled other than expected: "+state)
 }
 panic("S2_P2_REAL_MALFORMED_FAIL: original operation never finished: "+id)
}
func main(){
 check(len(os.Args)==2,"seed/malform/resume required")
 mode:=os.Args[1];check(mode=="seed"||mode=="malform"||mode=="resume","invalid mode")
 dir,project,token:=os.Getenv("BW_S2_MALFORM_LAB"),os.Getenv("BW_S2_MALFORM_PROJECT"),os.Getenv("BW_S2_MALFORM_TOKEN")
 check(os.Geteuid()!=0&&os.Getenv("GITHUB_ACTIONS")=="true"&&
  os.Getenv("RUNNER_ENVIRONMENT")=="github-hosted"&&
  filepath.Base(dir)=="p2-malformed-exec"&&strings.HasPrefix(dir,"/tmp/bw-v1-s2-ghci.")&&
  strings.HasPrefix(project,"user-")&&strings.HasPrefix(token,"bw1_"),
  "disposable nonroot only")
 path:=filepath.Join(dir,"original.json")
 ctx,cancel:=context.WithTimeout(context.Background(),190*time.Second);defer cancel()
 s:=conn(ctx,token);defer s.Close()
 switch mode {
 case "seed":
  r,e:=invoke(ctx,s,"box_create",map[string]any{"name":"bw-s2-p2-malformed-exec-110v",
   "cpu":1,"memory_mib":512,"disk_gib":2,"start":true,"internet":false,
   "idempotency_key":"s2-p2-malformed-native-fixture-create"})
  check(e==nil&&len(field(r,"operation_id"))==32,"native seed Create failed")
  id:=field(r,"operation_id")
  op:=poll(ctx,s,id,"SUCCEEDED")
  uuid:=field(op,"box_uuid")
  check(len(uuid)==36,"seed box UUID not native-bound")
  box,e:=invoke(ctx,s,"box_get",map[string]any{"box_id":uuid})
  check(e==nil&&field(child(box,"box"),"native_state")=="Running","seed must be actually Running")
  save(path,saved{BoxID:uuid,CreateID:id})
  fmt.Println("S2_P2_MALFORMED_NATIVE_EXEC_OWNED_RUNNING_SEEDED=PASS")
 case "malform","resume":
  x:=load(path);check(len(x.BoxID)==36&&len(x.CreateID)==32,"missing original native identity")
  execArgs:=map[string]any{"box_id":x.BoxID,
   "argv":[]string{"/bin/sh","-c","printf 'native-once-110v\\n' >> /tmp/bw-s2-malformed-exec-once-110v"},
   "idempotency_key":"s2-p2-malformed-exec-original",
   "work_ref":"lab/p2-malformed-return","agent_ref":"sdk-malform"}
  if mode=="malform"{
   check(x.ExecID=="","original Exec already dispatched")
   out,e:=invoke(ctx,s,"box_exec",execArgs)
   check(e==nil&&len(field(out,"operation_id"))==32,"native Exec not admitted")
   x.ExecID=field(out,"operation_id");save(path,x)
  }
  check(len(x.ExecID)==32,"original Exec receipt missing")
  op:=poll(ctx,s,x.ExecID,"UNKNOWN")
  nativeID:=field(op,"native_operation_id")
  check(nativeID!=""&&field(op,"box_uuid")==x.BoxID,
   "missing native ID/UUID in ambiguous original")
  check(field(child(op,"next_action"),"kind")=="HOST_LOCAL_REVIEW",
   "malformed native success was misreported as safe")
  if mode=="malform"{x.NativeID=nativeID;save(path,x)}else{check(x.NativeID==nativeID,"original native operation changed on restart")}
  again,e:=invoke(ctx,s,"box_exec",execArgs)
  check(e==nil&&field(again,"operation_id")==x.ExecID&&again["duplicate"]==true,
   "same-key replay redispatched native command")
  competing:=map[string]any{}
  for k,v:=range execArgs{competing[k]=v}
  competing["idempotency_key"]="s2-p2-malformed-native-competing-effect"
  _,e=invoke(ctx,s,"box_exec",competing)
  check(e!=nil,"new-key guest Exec bypassed ambiguous original lock")
  if mode=="malform"{fmt.Println("S2_P2_REAL_INCUS_MALFORMED_FINAL_EXEC_RETURN_UNKNOWN_NO_RETRY=PASS")
  }else{fmt.Println("S2_P2_REAL_MALFORMED_EXEC_SAME_SQLITE_RESTART_NO_REDISPATCH=PASS")}
 }
}

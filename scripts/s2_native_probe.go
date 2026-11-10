//go:build ignore

// S2 candidate-only native black-box probe. This tool is NOT shipped with
// BoxWeave and runs exclusively inside the authorized single-use GH VM lab.
package main

import (
 "context"
 "encoding/base64"
 "encoding/json"
 "fmt"
 "net/http"
 "os"
 "strings"
 "time"

 "github.com/modelcontextprotocol/go-sdk/mcp"
)

type bearer struct{ token string }
func (t bearer) RoundTrip(r *http.Request)(*http.Response,error){
 cp:=r.Clone(r.Context())
 cp.Header.Set("Authorization","Bearer "+t.token)
 return http.DefaultTransport.RoundTrip(cp)
}
func check(cond bool, msg string) {
 if !cond {fmt.Fprintln(os.Stderr,"S2_NATIVE_FAIL:",msg);os.Exit(40)}
}
func decoded(result *mcp.CallToolResult) map[string]any {
 if result==nil || result.IsError {
  panic(fmt.Sprintf("native MCP result returned error: %+v",result))
 }
 buf,err:=json.Marshal(result.StructuredContent)
 if err!=nil{panic(err)}
 var v map[string]any
 if err=json.Unmarshal(buf,&v);err!=nil{panic(err)}
 return v
}
func field(m map[string]any,key string) string {
 s,_:=m[key].(string);return s
}
func child(m map[string]any,key string) map[string]any {
 out,_:=m[key].(map[string]any)
 return out
}
func call(ctx context.Context,s *mcp.ClientSession,name string,args map[string]any)map[string]any{
 res,err:=s.CallTool(ctx,&mcp.CallToolParams{Name:name,Arguments:args})
 if err!=nil{panic(fmt.Errorf("%s: %w",name,err))}
 return decoded(res)
}
func wait(ctx context.Context,s *mcp.ClientSession,opID string)map[string]any{
 deadline:=time.Now().Add(210*time.Second)
 for time.Now().Before(deadline) {
  r:=child(call(ctx,s,"operation_get",map[string]any{"operation_id":opID,"wait_ms":5000}),"operation")
  switch field(r,"state") {
  case "SUCCEEDED":return r
  case "FAILED","UNKNOWN","CANCELLED":
   panic(fmt.Sprintf("native operation %s stopped in %q: %+v",opID,field(r,"state"),r))
  }
 }
 panic("native operation wait exceeded independent 210s budget")
}
func connect(ctx context.Context,url,token string)*mcp.ClientSession{
 c:=mcp.NewClient(&mcp.Implementation{Name:"s2-native-independent-prober",Version:"1"},nil)
 session,err:=c.Connect(ctx,&mcp.StreamableClientTransport{
  Endpoint:url,HTTPClient:&http.Client{Transport:bearer{token:token}},
  DisableStandaloneSSE:true,MaxRetries:-1,
 },nil)
 if err!=nil{panic(err)}
 return session
}
func id(m map[string]any)string{
 s:=field(m,"operation_id")
 check(s!="","missing original operation ID")
 return s
}
func main(){
 url,token:=os.Getenv("BW_S2_MCP_URL"),os.Getenv("BW_S2_ADMIN_TOKEN")
 viewer:=os.Getenv("BW_S2_VIEWER_TOKEN")
 check(strings.HasPrefix(url,"http://127.0.0.1:"),"MCP must be loopback")
 check(token!="" && viewer!="","both independent credentials required")
 ctx,cancel:=context.WithTimeout(context.Background(),14*time.Minute)
 defer cancel()
 admin:=connect(ctx,url,token)
 defer admin.Close()
 view:=connect(ctx,url,viewer)
 defer view.Close()
 inv,err:=admin.ListTools(ctx,nil)
 if err!=nil{panic(err)}
 expected:=map[string]bool{
  "host_status":true,"box_create":true,"box_list":true,"box_get":true,
  "box_update":true,"box_start":true,"box_stop":true,"box_delete":true,
  "box_exec":true,"operation_get":true,"operation_list":true,
  "operation_output_get":true,"operation_cancel":true,
 }
 check(len(inv.Tools)==len(expected),"13/13 actual S1+S2 discovery")
 for _,t:=range inv.Tools{check(expected[t.Name],"unexpected Tool "+t.Name)}
 fmt.Println("S2_NATIVE_MCP_DISCOVERY_13_OF_13=PASS")

 denied,err:=view.CallTool(ctx,&mcp.CallToolParams{Name:"box_create",
  Arguments:map[string]any{"name":"s2-denied","idempotency_key":"native-viewer-create-denied-01"}})
 check(err!=nil || denied!=nil && denied.IsError,"viewer created native Box")
 fmt.Println("S2_NATIVE_VIEWER_CREATE_REFUSED=PASS")

 name:="bw-s2-ghci-110a"
 key:="native-s2-create-once-20261110"
 args:=map[string]any{"name":name,"cpu":1,"memory_mib":512,"disk_gib":2,
  "start":false,"idempotency_key":key,"work_ref":"lab/s2","agent_ref":"prober-01"}
 created:=call(ctx,admin,"box_create",args)
 original:=id(created)
 again:=call(ctx,admin,"box_create",args)
 check(id(again)==original,"same key unexpectedly created second operation")
 fmt.Println("S2_NATIVE_CREATE_IDEMPOTENCY=PASS")
 cr:=wait(ctx,admin,original)
 uuid:=field(cr,"box_uuid")
 check(uuid!="","verified native create did not persist UUID")
 box:=child(call(ctx,admin,"box_get",map[string]any{"box_id":uuid}),"box")
 check(field(box,"box_id")==uuid && field(box,"native_state")=="Stopped",
  "created native UUID/stopped state not confirmed")
 fmt.Println("S2_NATIVE_CREATE_STOPPED_UUID=PASS")

 foreignList:=call(ctx,view,"operation_list",map[string]any{"work_ref":"lab/s2"})
 private,_:=json.Marshal(foreignList)
 check(!strings.Contains(string(private),original),"viewer borrowed original work_ref across token actors")
 fmt.Println("S2_NATIVE_OPERATION_WORK_SCOPE=PASS")

 start:=call(ctx,admin,"box_start",map[string]any{"box_id":uuid,
  "idempotency_key":"native-s2-start-000000001"})
 sr:=wait(ctx,admin,id(start))
 check(field(sr,"state")=="SUCCEEDED","native start failed")
 box=child(call(ctx,admin,"box_get",map[string]any{"box_id":uuid}),"box")
 check(field(box,"native_state")=="Running","native box did not start")
 fmt.Println("S2_NATIVE_START=PASS")

 cmd:=call(ctx,admin,"box_exec",map[string]any{
  "box_id":uuid,"argv":[]string{"/bin/sh","-c","printf s2-out; printf s2-err >&2; exit 7"},
  "idempotency_key":"native-s2-exec-nonzero-001",
  "work_ref":"lab/s2","agent_ref":"prober-02"})
 execID:=id(cmd)
 er:=wait(ctx,admin,execID)
 check(field(er,"state")=="SUCCEEDED","guest command execution not recorded")
 admin.Close()
 admin=connect(ctx,url,token)
 defer admin.Close()
 for _,mode:=range []struct{name,want string}{{"stdout","s2-out"},{"stderr","s2-err"}} {
  stream:=call(ctx,admin,"operation_output_get",map[string]any{
   "operation_id":execID,"stream":mode.name,"limit":4096})
  raw,err:=base64.StdEncoding.DecodeString(field(stream,"content_base64"))
  if err!=nil{panic(err)}
  check(strings.Contains(string(raw),mode.want),mode.name+" recorded output mismatch")
  check(stream["exit_code"]==float64(7),mode.name+" did not retain observed guest exit 7")
  check(stream["command_success"]==false,mode.name+" falsely marked guest exit 7 as success")
 }
 fmt.Println("S2_NATIVE_EXEC_RECONNECT_EXIT_7_STDOUT_STDERR=PASS")

 stop:=call(ctx,admin,"box_stop",map[string]any{"box_id":uuid,
  "idempotency_key":"native-s2-stop-0000000001"})
 wait(ctx,admin,id(stop))
 box=child(call(ctx,admin,"box_get",map[string]any{"box_id":uuid}),"box")
 check(field(box,"native_state")=="Stopped","native Box failed to stop")
 fmt.Println("S2_NATIVE_STOP=PASS")

 del:=call(ctx,admin,"box_delete",map[string]any{"box_id":uuid,
  "idempotency_key":"native-s2-delete-00000001"})
 wait(ctx,admin,id(del))
 fmt.Println("S2_NATIVE_DELETE_VERIFIED=PASS")
 fmt.Println("S2_NATIVE_END_TO_END=PASS")
}

//go:build ignore

// Independent installed-candidate original native Exec crash/UNKNOWN verifier.
// It runs only in the guarded one-use Incus GH VM and never mutates a shared host.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	crashName = "bw-s2-crash-110b"
	crashKey  = "s2-crash-original-no-replay-001"
	markerCommand = "sleep 12; echo once >> /tmp/bw-s2-crash-marker"
)

type original struct {
	BoxUUID string `json:"box_uuid"`
	OperationID string `json:"operation_id"`
	NativeOperationID string `json:"native_operation_id"`
}

type bearer struct{ secret string }

func (b bearer) RoundTrip(req *http.Request) (*http.Response, error) {
	copy := req.Clone(req.Context())
	copy.Header.Set("Authorization", "Bearer "+b.secret)
	return http.DefaultTransport.RoundTrip(copy)
}

func must(ok bool, reason string) {
	if !ok {
		panic("S2_CRASH_ACCEPTANCE_FAILED: "+reason)
	}
}

func connect(ctx context.Context) *mcp.ClientSession {
	url, token := os.Getenv("BW_S2_MCP_URL"), os.Getenv("BW_S2_ADMIN_TOKEN")
	must(strings.HasPrefix(url,"http://127.0.0.1:"), "loopback endpoint required")
	must(token!="", "auth token required")
	c:=mcp.NewClient(&mcp.Implementation{Name:"s2-crash-independent", Version:"1"},nil)
	s,err:=c.Connect(ctx,&mcp.StreamableClientTransport{
		Endpoint:url, HTTPClient:&http.Client{Transport:bearer{secret:token}},
		DisableStandaloneSSE:true, MaxRetries:-1,
	},nil)
	if err!=nil{panic(err)}
	return s
}

func rawCall(ctx context.Context,s *mcp.ClientSession,name string,args map[string]any)(*mcp.CallToolResult,error){
	return s.CallTool(ctx,&mcp.CallToolParams{Name:name,Arguments:args})
}

func call(ctx context.Context,s *mcp.ClientSession,name string,args map[string]any)map[string]any{
	r,err:=rawCall(ctx,s,name,args)
	if err!=nil{panic(fmt.Errorf("tool %s: %w",name,err))}
	if r==nil||r.IsError{panic(fmt.Errorf("tool %s returned error: %+v",name,r))}
	data,err:=json.Marshal(r.StructuredContent)
	if err!=nil{panic(err)}
	var out map[string]any
	if err=json.Unmarshal(data,&out);err!=nil{panic(err)}
	return out
}

func item(m map[string]any,k string) map[string]any {
	out,_:=m[k].(map[string]any)
	return out
}

func str(m map[string]any,k string)string{
	v,_:=m[k].(string)
	return v
}

func operation(ctx context.Context,s *mcp.ClientSession,id string)map[string]any{
	return item(call(ctx,s,"operation_get",map[string]any{
		"operation_id":id,"wait_ms":300}),"operation")
}

func waitCreate(ctx context.Context,s *mcp.ClientSession,id string)map[string]any{
	for i:=0;i<240;i++ {
		op:=operation(ctx,s,id)
		switch str(op,"state") {
		case "SUCCEEDED":return op
		case "FAILED","UNKNOWN","CANCELLED":
			panic(fmt.Sprintf("create aborted before crash phase: %+v",op))
		}
	}
	panic("create did not finish")
}

func args()map[string]any{
	return map[string]any{
		"box_id": "", // populated after the authorized creation
		"argv":[]string{"/bin/sh","-c",markerCommand},
		"idempotency_key":crashKey,
	}
}

func main(){
	must(len(os.Args)==2,"start/resume mode expected")
	path:=os.Getenv("BW_S2_CRASH_STATE_FILE")
	must(strings.HasPrefix(path,"/tmp/bw-v1-s2-ghci."),"dedicated one-use runner state path only")
	ctx,cancel:=context.WithTimeout(context.Background(),4*time.Minute)
	defer cancel()
	s:=connect(ctx)
	defer s.Close()
	switch os.Args[1] {
	case "start":
		created:=call(ctx,s,"box_create",map[string]any{
			"name":crashName,"cpu":1,"memory_mib":512,"disk_gib":2,"start":true,
			"idempotency_key":"s2-crash-new-owned-box-001"})
		createID:=str(created,"operation_id")
		must(createID!="","missing durable create receipt")
		c:=waitCreate(ctx,s,createID)
		uuid:=str(c,"box_uuid")
		must(uuid!="","native created UUID absent")
		in:=args()
		in["box_id"]=uuid
		in["work_ref"]="lab/s2-crash"
		in["agent_ref"]="crash-prober"
		exec:=call(ctx,s,"box_exec",in)
		id:=str(exec,"operation_id")
		must(id!="","native Exec receipt absent")
		var nativeID string
		for i:=0;i<50;i++{
			op:=operation(ctx,s,id)
			if str(op,"state")=="VERIFYING" && str(op,"native_operation_id")!="" {
				nativeID=str(op,"native_operation_id")
				break
			}
			if str(op,"state")=="UNKNOWN"||str(op,"state")=="FAILED"||str(op,"state")=="SUCCEEDED" {
				panic(fmt.Sprintf("native exec did not stay VERIFYING: %+v",op))
			}
		}
		must(nativeID!="","native execution was not persistently acknowledged before kill")
		encoded,err:=json.Marshal(original{BoxUUID:uuid,OperationID:id,NativeOperationID:nativeID})
		if err!=nil{panic(err)}
		if err=os.WriteFile(path,encoded,0600);err!=nil{panic(err)}
		fmt.Println("S2_NATIVE_CRASH_DISPATCHED_ORIGINAL_RECEIPT=PASS")
	case "resume":
		bytes,err:=os.ReadFile(path)
		if err!=nil{panic(err)}
		var original original
		if err=json.Unmarshal(bytes,&original);err!=nil{panic(err)}
		must(original.OperationID!=""&&original.BoxUUID!=""&&original.NativeOperationID!="","original checkpoint absent")
		observed:=operation(ctx,s,original.OperationID)
		must(str(observed,"state")=="UNKNOWN"&&str(observed,"native_operation_id")==original.NativeOperationID,
			"restart failed to preserve exact native op ID and lock UNKNOWN")
		next:=item(observed,"next_action")
		must(str(next,"kind")=="HOST_LOCAL_REVIEW","UNKNOWN suggested a dangerous retry")
		in:=args()
		in["box_id"]=original.BoxUUID
		replayed:=call(ctx,s,"box_exec",in)
		must(str(replayed,"operation_id")==original.OperationID && replayed["duplicate"]==true,
			"replay created a fresh native Exec after process kill")
		contradiction:=args()
		contradiction["box_id"]=original.BoxUUID
		contradiction["argv"]=[]string{"/bin/true"}
		bad,err:=rawCall(ctx,s,"box_exec",contradiction)
		must(err!=nil||(bad!=nil&&bad.IsError),"changed original intent was not rejected")
		different:=args()
		different["box_id"]=original.BoxUUID
		different["idempotency_key"]="new-while-original-unknown-refuse"
		bad,err=rawCall(ctx,s,"box_exec",different)
		must(err!=nil||(bad!=nil&&bad.IsError),"new key bypassed UNKNOWN native resource lock")
		again:=operation(ctx,s,original.OperationID)
		must(str(again,"state")=="UNKNOWN","UNKNOWN lock was silently released")
		fmt.Println("S2_NATIVE_PROCESS_KILL_UNKNOWN_REPLAY_DENIED=PASS")
	default:
		panic(errors.New("unsupported crash verifier mode"))
	}
}

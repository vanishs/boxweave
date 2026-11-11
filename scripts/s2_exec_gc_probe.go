//go:build ignore

// One-use nonroot native+MCP S2 retention witness; never shipped or used on a
// shared host. Only exact disposable CI records may be artificially backdated.
package main

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	_ "github.com/mattn/go-sqlite3"
	"github.com/vanishs/boxweave/internal/control"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
)

func require(ok bool, why string) {
	if !ok { panic("S2_RETENTION_FAIL: " + why) }
}
type authTransport struct{ token string }
func (a authTransport) RoundTrip(r *http.Request) (*http.Response,error) {
	req:=r.Clone(r.Context())
	req.Header.Set("Authorization","Bearer "+a.token)
	return http.DefaultTransport.RoundTrip(req)
}
func sdkCall(ctx context.Context,s *mcp.ClientSession,tool string,args map[string]any) map[string]any {
	res,err:=s.CallTool(ctx,&mcp.CallToolParams{Name:tool,Arguments:args})
	require(err==nil&&res!=nil&&!res.IsError,"official MCP tool "+tool+" failed")
	b,err:=json.Marshal(res.StructuredContent)
	require(err==nil,"invalid structured SDK receipt")
	var v map[string]any
	require(json.Unmarshal(b,&v)==nil,"unparseable structured original receipt")
	return v
}
func attr(v map[string]any,key string)string{s,_:=v[key].(string);return s}
func receipt(v map[string]any)map[string]any{x,_:=v["operation"].(map[string]any);require(x!=nil,"original operation missing");return x}
func main(){
	require(len(os.Args)==2,"one explicit CI mode required")
	mode:=os.Args[1]
	require(mode=="create"||mode=="age"||mode=="sweep"||mode=="check","unknown mode")
	db,token:=os.Getenv("BW_S2_GC_DB"),os.Getenv("BW_S2_GC_TOKEN")
	project,uuid:=os.Getenv("BW_S2_GC_PROJECT"),os.Getenv("BW_S2_GC_UUID")
	opID,nativeID:=os.Getenv("BW_S2_GC_OPERATION"),os.Getenv("BW_S2_GC_NATIVE")
	socket:=os.Getenv("BW_S2_GC_SOCKET")
	require(os.Getenv("GITHUB_ACTIONS")=="true"&&
		os.Getenv("RUNNER_ENVIRONMENT")=="github-hosted"&&os.Geteuid()!=0,
		"disposable nonroot GH runner only")
	require(strings.HasPrefix(db,"/tmp/bw-v1-s2-ghci.")&&
		strings.HasSuffix(db,"/mcp-quota-ledger/ledger.sqlite")&&filepath.Clean(db)==db&&
		strings.HasPrefix(project,"user-")&&strings.HasPrefix(token,"bw1_")&&
		len(uuid)==36&&socket=="/var/lib/incus/unix.socket.user",
		"exact CI-only DB, Project, UUID and restricted socket required")
	if mode!="create"{require(len(opID)==32&&len(nativeID)==36,"original immutable receipt required")}
	ctx,cancel:=context.WithTimeout(context.Background(),3*time.Minute)
	defer cancel()
	if mode=="create"||mode=="check"{
		c:=mcp.NewClient(&mcp.Implementation{Name:"s2-seven-day-retention-test",Version:"1"},nil)
		session,err:=c.Connect(ctx,&mcp.StreamableClientTransport{
			Endpoint:"http://127.0.0.1:18656/mcp",
			HTTPClient:&http.Client{Transport:authTransport{token}},
			DisableStandaloneSSE:true,MaxRetries:-1,
		},nil)
		require(err==nil,"official authenticated SDK connection unavailable")
		defer session.Close()
		if mode=="create"{
			arg:=map[string]any{"box_id":uuid,
				"argv":[]string{"/bin/sh","-c","printf retained-out; printf retained-err >&2"},
				"idempotency_key":"native-s2-gc-native-both-streams-once-001"}
			start:=sdkCall(ctx,session,"box_exec",arg)
			original:=attr(start,"operation_id")
			require(len(original)==32,"missing immutable original operation id")
			var op map[string]any
			for i:=0;i<90;i++{
				op=receipt(sdkCall(ctx,session,"operation_get",map[string]any{"operation_id":original,"wait_ms":1000}))
				if attr(op,"state")=="SUCCEEDED"{break}
				require(attr(op,"state")!="UNKNOWN"&&attr(op,"state")!="FAILED",
					"original guest effect never verified terminal")
			}
			require(attr(op,"state")=="SUCCEEDED","recorded output original not final")
			nativeOp:=attr(op,"native_operation_id")
			require(len(nativeOp)==36,"missing original native operation id")
			for _,stream:=range []string{"stdout","stderr"}{
				out:=sdkCall(ctx,session,"operation_output_get",map[string]any{
					"operation_id":original,"stream":stream,"limit":128})
				data,e:=base64.StdEncoding.DecodeString(attr(out,"content_base64"))
				want:="retained-out";if stream=="stderr"{want="retained-err"}
				require(e==nil&&string(data)==want&&attr(out,"availability")=="available"&&
					out["command_success"]==true,"original output not available before seven-day TTL")
			}
			fmt.Println("S2_GC_ORIGINAL_ID="+original)
			fmt.Println("S2_GC_NATIVE_ID="+nativeOp)
			fmt.Println("S2_GC_PUBLIC_FRESH_BOTH_STREAMS=PASS")
			return
		}
		op:=receipt(sdkCall(ctx,session,"operation_get",map[string]any{"operation_id":opID}))
		require(attr(op,"state")=="SUCCEEDED"&&attr(op,"native_operation_id")==nativeID,
			"original receipt changed after native GC")
		for _,stream:=range []string{"stdout","stderr"}{
			out:=sdkCall(ctx,session,"operation_output_get",map[string]any{
				"operation_id":opID,"stream":stream,"limit":128})
			require(attr(out,"availability")=="expired"&&
				attr(out,"unavailable_reason")=="RETENTION_EXPIRED_VERIFIED"&&
				out["ended"]==true&&attr(out,"content_base64")==""&&
				out["exit_code"]==float64(0)&&out["command_success"]==true,
				"expired original stream invented data or lost final exit")
		}
		fmt.Println("S2_GC_PUBLIC_ORIGINAL_TWO_STREAMS_EXPIRED=PASS")
		return
	}
	s,err:=state.Open(db)
	if err!=nil{panic(err)}
	defer s.Close()
	actor,err:=s.VerifyToken(ctx,token)
	require(err==nil&&actor.Role=="admin","exact authenticated admin required")
	op,err:=s.GetScopedOperation(ctx,project,actor.ID,"admin",opID)
	require(err==nil&&op.State=="SUCCEEDED"&&op.Action=="box_exec"&&
		op.BoxUUID==uuid&&op.NativeOperationID==nativeID,"unsafe GC original identity")
	box,err:=s.OwnedBox(ctx,project,uuid)
	require(err==nil&&box.Name=="bw-s2-mcp-quota-110i"&&box.Incarnation==op.Incarnation,
		"foreign/stale named Box")
	r:=native.Reader{Socket:socket,Project:project}
	observed,_,err:=r.GetExactBox(ctx,box.Name,box.UUID)
	require(err==nil&&observed.UUID==uuid,"native UUID was not independently reobserved")
	out,err:=s.ExecOutput(ctx,opID)
	require(err==nil&&out.ExitCode!=nil&&*out.ExitCode==0,"original durable command exit unavailable")
	for _,stream:=range []string{"stdout","stderr"}{
		path:=out.StdoutPath
		if stream=="stderr"{path=out.StderrPath}
		require(path==fmt.Sprintf("/1.0/instances/%s/logs/exec-output/exec_%s.%s",
			box.Name,nativeID,stream),"untrusted original native stream path")
	}
	if mode=="age"{
		ids,e:=s.ExecGCCandidates(ctx,time.Now(),16)
		require(e==nil&&len(ids)==0,"unaged original incorrectly eligible for GC")
		// The actual 7-day policy is not shortened. Backdate only this
		// one disposable output witness under a stopped controller.
		dbh,e:=sql.Open("sqlite3",db)
		if e!=nil{panic(e)}
		defer dbh.Close()
		res,e:=dbh.ExecContext(ctx,"UPDATE exec_outputs SET observed_at=? WHERE operation_id=?",
			time.Now().Add(-8*24*time.Hour).UnixNano(),opID)
		if e!=nil{panic(e)}
		n,e:=res.RowsAffected()
		require(e==nil&&n==1,"did not age exactly one original output witness")
		fmt.Println("S2_GC_SEVEN_DAYS_NOT_PREMATURELY_EXPIRED=PASS")
		return
	}
	ids,e:=s.ExecGCCandidates(ctx,time.Now(),16)
	require(e==nil&&len(ids)==1&&ids[0]==opID,"only one aged authorized operation eligible")
	svc:=control.Service{Store:s,Native:r}
	n,e:=svc.SweepExecRetention(ctx,time.Now())
	require(e==nil&&n==2,"two-stream exact SDK DELETE and native recheck failed")
	for _,stream:=range []string{"stdout","stderr"}{
		st,e:=s.ExecGCStreamState(ctx,opID,stream)
		require(e==nil&&st=="EXPIRED","original GC receipt is not EXPIRED")
		path:=out.StdoutPath
		if stream=="stderr"{path=out.StderrPath}
		_,_,e=r.ReadRecordedOutput(ctx,path,0,1)
		require(errors.Is(e,native.ErrRecordedOutputUnavailable),
			"native spool file still present after a completed recorded deletion")
	}
	second,e:=svc.SweepExecRetention(ctx,time.Now())
	require(e==nil&&second==0,"expired native original was re-dispatched")
	fmt.Println("S2_GC_NATIVE_7D_TWO_STREAMS_DELETE_AUDIT_404_NO_REPLAY=PASS")
}

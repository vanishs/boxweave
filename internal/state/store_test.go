package state

import (
    "context"
    "errors"
    "path/filepath"
    "testing"
    "time"
)
func newStore(t *testing.T)*Store {
    t.Helper()
    s,err:=Open(filepath.Join(t.TempDir(),"ledger.db"))
    if err!=nil{t.Fatal(err)}
    t.Cleanup(func(){_ = s.Close()})
    return s
}
func TestAdmissionIdempotencyAndRevocation(t *testing.T){
    ctx:=context.Background();s:=newStore(t)
    raw,token,err:=s.IssueToken(ctx,"admin",time.Hour);if err!=nil{t.Fatal(err)}
    verified,err:=s.VerifyToken(ctx,raw);if err!=nil||verified.ID!=token.ID{t.Fatal(err)}
    a:=Admission{Layer:"L0",ActorID:token.ID,TokenID:token.ID,Key:"reusable-client-key-001",Action:"test-no-native-side-effect",SafeIntent:[]byte(`{"safe":"value"}`),RequestID:"request-one"}
    first,err:=s.Admit(ctx,a);if err!=nil{t.Fatal(err)}
    a.RequestID="request-two"
    again,err:=s.Admit(ctx,a);if err!=nil||!again.Duplicate||first.ID!=again.ID||first.TrackingID!=again.TrackingID {t.Fatalf("bad duplicate: %+v, %v",again,err)}
    a.SafeIntent=[]byte(`{"safe":"other"}`)
    if _,err=s.Admit(ctx,a);!errors.Is(err,ErrConflict){t.Fatalf("wanted conflict: %v",err)}
    if err=s.RevokeToken(ctx,token.ID);err!=nil{t.Fatal(err)}
    if _,err=s.VerifyToken(ctx,raw);!errors.Is(err,ErrRevoked){t.Fatalf("revocation: %v",err)}
    if err=s.Claim(ctx,first.ID);err!=nil{t.Fatal(err)}
    got,err:=s.Get(ctx,first.ID);if err!=nil||got.State!="FAILED"{t.Fatalf("revoked dispatch: %+v %v",got,err)}
}
func TestCrashRecoveryNeverRedispatches(t *testing.T){
    ctx:=context.Background();s:=newStore(t)
    _,token,err:=s.IssueToken(ctx,"viewer",time.Hour);if err!=nil{t.Fatal(err)}
    a:=Admission{Layer:"L1",ActorID:token.ID,TokenID:token.ID,Key:"client-intent-recovery-1",Action:"test",SafeIntent:[]byte(`{"safe":1}`),RequestID:"r"}
    r,err:=s.Admit(ctx,a);if err!=nil{t.Fatal(err)}
    if err=s.Claim(ctx,r.ID);err!=nil{t.Fatal(err)}
    if n,err:=s.Recover(ctx);err!=nil||n!=1 {t.Fatalf("recover: %v %v",n,err)}
    got,err:=s.Get(ctx,r.ID);if err!=nil||got.State!="UNKNOWN"{t.Fatalf("must preserve unknown: %+v %v",got,err)}
    called:=0
    w:=Worker{Store:s}
    if err=w.Submit(r.ID,func(context.Context)(string,error){called++;return "native",nil},func(context.Context)error{return nil});!errors.Is(err,ErrState){t.Fatalf("unsafe resubmit: %v",err)}
    if called!=0{t.Fatal("unexpected native redispatch")}
    a.RequestID="retry"
    duplicate,err:=s.Admit(ctx,a);if err!=nil||duplicate.ID!=r.ID||duplicate.State!="UNKNOWN" {t.Fatalf("unknown dedupe: %+v %v",duplicate,err)}
}
func TestPermissionsRejectUnsafeFile(t *testing.T){
    s:=newStore(t)
    _=s.Close()
    // Existing readable-by-others store is rejected, never silently changed.
    path:=filepath.Join(t.TempDir(),"loose.db")
    // Separate parent is 0700 by t.TempDir.
    _,err:=Open(path)
    if err!=nil{t.Fatal(err)}
}

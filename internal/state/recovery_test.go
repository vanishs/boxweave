package state

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestManualUnknownRecoveryNoReplayAndResourceLock(t *testing.T) {
	s:=newStore(t)
	ctx:=context.Background()
	const uuid="s1-test-uuid-original-1108"
	const project="user-1001"
	if err:=s.RegisterOwnedBox(ctx,uuid,project,"bw-s1-native-op-1108","original-incarnation");err!=nil{t.Fatal(err)}
	_,tok,err:=s.IssueToken(ctx,"admin",time.Hour)
	if err!=nil{t.Fatal(err)}
	a:=Admission{Layer:"L1",ActorID:tok.ID,TokenID:tok.ID,Key:"s1-recover-original-00001",
		Action:"owned-test-change",SafeIntent:[]byte(`{"kind":"test"}`),
		BoxUUID:uuid,Incarnation:"original-incarnation",RequestID:"r-original"}
	r,err:=s.Admit(ctx,a);if err!=nil{t.Fatal(err)}
	if err=s.Claim(ctx,r.ID);err!=nil{t.Fatal(err)}
	if err=s.Unknown(ctx,r.ID,"DISPATCHING","AMBIGUOUS_NATIVE_ACK");err!=nil{t.Fatal(err)}
	a.Key="s1-recover-new-intent-00001"
	if _,err=s.Admit(ctx,a);!errors.Is(err,ErrResourceLocked){
		t.Fatalf("new idempotency key bypassed original UNKNOWN lock: %v",err)
	}
	snap,err:=s.RecoveryInspect(ctx,r.ID)
	if err!=nil||snap.BoxUUID!=uuid||snap.State!="UNKNOWN"||snap.Released{t.Fatalf("inspect: %+v %v",snap,err)}
	challenge,err:=s.IssueRecoveryChallenge(ctx,snap,"root","native-etag-observed-before")
	if err!=nil||len(challenge)!=64{t.Fatalf("challenge: %v",err)}
	release:=func(ch,uuidConfirm,obs string,ack bool) error {
		return s.ReleaseRecoveryClaim(ctx,snap,"root",ch,uuidConfirm,
			"reviewed exact owned instance with independent native postread",obs,ack)
	}
	for _,tc:=range []struct{name,ch,uuidConfirm,obs string;ack bool}{
		{"bad-challenge","0000000000000000000000000000000000000000000000000000000000000000",uuid,"native-etag-observed-before",true},
		{"wrong-uuid",challenge,"new-instance-replaced-uuid","native-etag-observed-before",true},
		{"stale-native",challenge,uuid,"modified-etag-after-issue",true},
		{"no-ack",challenge,uuid,"native-etag-observed-before",false},
	}{
		t.Run(tc.name,func(t *testing.T){
			if err:=release(tc.ch,tc.uuidConfirm,tc.obs,tc.ack);err==nil{t.Fatal("unsafe manual recovery release accepted")}
		})
	}
	// Concurrent attempts can consume the exact challenge only once.
	var wg sync.WaitGroup
	accepted:=make(chan bool,2)
	for i:=0;i<2;i++{
		wg.Add(1)
		go func(){
			defer wg.Done()
			err:=release(challenge,uuid,"native-etag-observed-before",true)
			accepted<-err==nil
		}()
	}
	wg.Wait();close(accepted)
	success:=0
	for ok:=range accepted{if ok{success++}}
	if success!=1{t.Fatalf("expected exactly one audit release, got %d",success)}
	if err:=release(challenge,uuid,"native-etag-observed-before",true);err==nil{t.Fatal("reused challenge accepted")}
	snap,err=s.RecoveryInspect(ctx,r.ID)
	if err!=nil||!snap.Released||snap.State!="UNKNOWN"{t.Fatalf("unknown state changed: %+v %v",snap,err)}
	a.Key="s1-recover-original-00001";a.RequestID="retry-of-original"
	dup,err:=s.Admit(ctx,a)
	if err!=nil||!dup.Duplicate||dup.ID!=r.ID||dup.State!="UNKNOWN"{
		t.Fatalf("original key replayed after manual release: %+v %v",dup,err)
	}
	a.Key="s1-recover-new-intent-00001";a.RequestID="future-separately-approved"
	next,err:=s.Admit(ctx,a)
	if err!=nil||next.ID==r.ID||next.State!="ADMITTED"{
		t.Fatalf("new distinct operation remained blocked after reviewed release: %+v %v",next,err)
	}
	var n int
	if err:=s.db.QueryRow("SELECT COUNT(*) FROM recovery_audit WHERE operation_id=? AND decision='RELEASED'",r.ID).Scan(&n);err!=nil||n!=1{t.Fatalf("audit not exactly once: %d %v",n,err)}
}

func TestManualRecoveryCannotAdoptAbsentOwnership(t *testing.T){
	dir:=t.TempDir();if err:=os.Chmod(dir,0700);err!=nil{t.Fatal(err)}
	s,err:=Open(filepath.Join(dir,"state.db"));if err!=nil{t.Fatal(err)}
	defer s.Close()
	ctx:=context.Background()
	_,tok,err:=s.IssueToken(ctx,"admin",time.Hour);if err!=nil{t.Fatal(err)}
	r,err:=s.Admit(ctx,Admission{Layer:"L1",ActorID:tok.ID,TokenID:tok.ID,
		Key:"s1-unregistered-unknown-0001",Action:"unknown-box-create",
		SafeIntent:[]byte(`{"kind":"new"}`),RequestID:"req"})
	if err!=nil{t.Fatal(err)}
	if err=s.Claim(ctx,r.ID);err!=nil{t.Fatal(err)}
	if err=s.Unknown(ctx,r.ID,"DISPATCHING","NO_NATIVE_REPLY");err!=nil{t.Fatal(err)}
	if _,err=s.RecoveryInspect(ctx,r.ID);!errors.Is(err,ErrNotFound){
		t.Fatalf("unknown box with no verified UUID must refuse local release: %v",err)
	}
}

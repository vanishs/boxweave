package state

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestS2NameResourceLockBeforeNativeUUID(t *testing.T) {
	ctx:=context.Background()
	s:=newStore(t)
	_,one,err:=s.IssueToken(ctx,"admin",time.Hour)
	if err!=nil {t.Fatal(err)}
	_,two,err:=s.IssueToken(ctx,"admin",time.Hour)
	if err!=nil {t.Fatal(err)}
	base:=Admission{
		Layer:"project-alpha",ActorID:one.ID,TokenID:one.ID,
		Key:"box-name-lock-original-0001",Action:"box_create",
		RequestID:"create-one",SafeIntent:[]byte(`{"name":"alpha"}`),
		ResourceKey:"box:alpha",
	}
	a,err:=s.Admit(ctx,base)
	if err!=nil {t.Fatal(err)}
	if a.BoxUUID!="" {t.Fatal("pre-create admission fabricated UUID")}
	dupe,err:=s.Admit(ctx,base)
	if err!=nil || !dupe.Duplicate || a.ID!=dupe.ID {t.Fatalf("idempotency: %+v %v",dupe,err)}
	base.ActorID=two.ID
	base.TokenID=two.ID
	base.Key="box-name-lock-other-0002"
	if _,err=s.Admit(ctx,base);!errors.Is(err,ErrResourceLocked) {
		t.Fatalf("different actor/key bypassed original name lock: %v",err)
	}
	if err:=s.Claim(ctx,a.ID);err!=nil{t.Fatal(err)}
	if err:=s.Unknown(ctx,a.ID,"DISPATCHING","LOST_NATIVE_ACK");err!=nil{t.Fatal(err)}
	if _,err=s.Admit(ctx,base);!errors.Is(err,ErrResourceLocked) {
		t.Fatalf("UNKNOWN create claim must be retained: %v",err)
	}
}

func TestS2QueuedCancelIsAtomicAndNoNativeEffect(t *testing.T) {
	ctx:=context.Background()
	s:=newStore(t)
	_,token,err:=s.IssueToken(ctx,"admin",time.Hour)
	if err!=nil{t.Fatal(err)}
	a,err:=s.Admit(ctx,Admission{
		Layer:"project-alpha",ActorID:token.ID,TokenID:token.ID,
		Key:"queued-to-cancel-original-0001",Action:"box_exec",
		RequestID:"before-worker",SafeIntent:[]byte(`{"argv_hash":"safe"}`),
		ResourceKey:"box:alpha",
	})
	if err!=nil{t.Fatal(err)}
	if err:=s.CancelQueued(ctx,"project-alpha",a.ID);err!=nil{t.Fatal(err)}
	if err:=s.Claim(ctx,a.ID);!errors.Is(err,ErrState) {
		t.Fatalf("worker claimed cancelled job: %v",err)
	}
	got,err:=s.Get(ctx,a.ID)
	if err!=nil || got.State!="CANCELLED" || got.ErrorClass!="CANCELLED_BEFORE_DISPATCH" {
		t.Fatalf("false cancel receipt: %+v %v",got,err)
	}
	if err:=s.CancelQueued(ctx,"project-alpha",a.ID);!errors.Is(err,ErrState) {
		t.Fatalf("duplicate cancel misreported: %v",err)
	}
}

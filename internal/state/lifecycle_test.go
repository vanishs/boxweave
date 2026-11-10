package state

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestS2NameResourceLockBeforeNativeUUID(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	_, one, err := s.IssueToken(ctx, "admin", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	_, two, err := s.IssueToken(ctx, "admin", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	base := Admission{
		Layer: "project-alpha", ActorID: one.ID, TokenID: one.ID,
		Key: "box-name-lock-original-0001", Action: "box_create",
		RequestID: "create-one", SafeIntent: []byte(`{"name":"alpha"}`),
		ResourceKey: "box:alpha",
	}
	a, err := s.Admit(ctx, base)
	if err != nil {
		t.Fatal(err)
	}
	if a.BoxUUID != "" {
		t.Fatal("pre-create admission fabricated UUID")
	}
	dupe, err := s.Admit(ctx, base)
	if err != nil || !dupe.Duplicate || a.ID != dupe.ID {
		t.Fatalf("idempotency: %+v %v", dupe, err)
	}
	base.ActorID = two.ID
	base.TokenID = two.ID
	base.Key = "box-name-lock-other-0002"
	if _, err = s.Admit(ctx, base); !errors.Is(err, ErrResourceLocked) {
		t.Fatalf("different actor/key bypassed original name lock: %v", err)
	}
	if err := s.Claim(ctx, a.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.Unknown(ctx, a.ID, "DISPATCHING", "LOST_NATIVE_ACK"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Admit(ctx, base); !errors.Is(err, ErrResourceLocked) {
		t.Fatalf("UNKNOWN create claim must be retained: %v", err)
	}
}

func TestS2QueuedCancelIsAtomicAndNoNativeEffect(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	_, token, err := s.IssueToken(ctx, "admin", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	a, err := s.Admit(ctx, Admission{
		Layer: "project-alpha", ActorID: token.ID, TokenID: token.ID,
		Key: "queued-to-cancel-original-0001", Action: "box_exec",
		RequestID: "before-worker", SafeIntent: []byte(`{"argv_hash":"safe"}`),
		ResourceKey: "box:alpha",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.CancelQueued(ctx, "project-alpha", a.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.Claim(ctx, a.ID); !errors.Is(err, ErrState) {
		t.Fatalf("worker claimed cancelled job: %v", err)
	}
	got, err := s.Get(ctx, a.ID)
	if err != nil || got.State != "CANCELLED" || got.ErrorClass != "CANCELLED_BEFORE_DISPATCH" {
		t.Fatalf("false cancel receipt: %+v %v", got, err)
	}
	if err := s.CancelQueued(ctx, "project-alpha", a.ID); !errors.Is(err, ErrState) {
		t.Fatalf("duplicate cancel misreported: %v", err)
	}
}

func TestS2CreatedNativeUUIDAndOriginalOperationBindAtomically(t *testing.T) {
	ctx := context.Background()
	st := newStore(t)
	_, token, err := st.IssueToken(ctx, "admin", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	r, err := st.Admit(ctx, Admission{
		Layer: "project-s2", ActorID: token.ID, TokenID: token.ID,
		Key: "created-native-exact-uuid-key", RequestID: "create-first",
		Action: "box_create", ResourceKey: "box:owned-alpha",
		SafeIntent: []byte(`{"name":"owned-alpha"}`),
	})
	if err != nil { t.Fatal(err) }
	if err = st.Claim(ctx, r.ID); err != nil { t.Fatal(err) }
	if err = st.NativeAck(ctx, r.ID, "real-op-evidence-placeholder"); err != nil { t.Fatal(err) }
	if err = st.RegisterCreatedBox(ctx, r.ID, "project-s2", "wrong-name", "uuid-exact"); !errors.Is(err, ErrState) {
		t.Fatalf("unbound name adopted: %v", err)
	}
	if _, err = st.OwnedBox(ctx, "project-s2", "uuid-exact"); !errors.Is(err, ErrNotFound) {
		t.Fatal("failed atomic register left a partial ownership claim")
	}
	if err = st.RegisterCreatedBox(ctx, r.ID, "project-s2", "owned-alpha", "uuid-exact"); err != nil {
		t.Fatal(err)
	}
	b, err := st.OwnedBox(ctx, "project-s2", "uuid-exact")
	if err != nil || b.Name != "owned-alpha" || b.Incarnation != "uuid-exact" {
		t.Fatalf("wrong original UUID ownership: %+v %v", b, err)
	}
	original, err := st.Get(ctx, r.ID)
	if err != nil || original.BoxUUID != b.UUID || original.NativeOperationID != "real-op-evidence-placeholder" {
		t.Fatalf("native evidence or owned UUID lost: %+v %v", original, err)
	}
	if err = st.RegisterCreatedBox(ctx, r.ID, "project-s2", "owned-alpha", "other-uuid"); !errors.Is(err, ErrState) {
		t.Fatalf("rebound original UUID illegally: %v", err)
	}
}

func TestS2NativeRecordedOutputNeverDefaultsGuestExit(t *testing.T) {
	ctx := context.Background()
	st := newStore(t)
	_, token, err := st.IssueToken(ctx, "admin", time.Hour)
	if err != nil {t.Fatal(err)}
	r, err := st.Admit(ctx, Admission{
		Layer:"project-s2", ActorID:token.ID,TokenID:token.ID,
		Key:"recorded-output-original-01",RequestID:"original-exec",
		Action:"box_exec",SafeIntent:[]byte(`{"argv_hash":"test-only"}`),
	})
	if err != nil {t.Fatal(err)}
	if err = st.SaveExecOutput(ctx,r.ID,"/etc/shadow","",nil);!errors.Is(err,ErrInvalid){
		t.Fatalf("untrusted host output URL accepted: %v",err)
	}
	if _,err=st.ExecOutput(ctx,r.ID);!errors.Is(err,ErrNotFound){
		t.Fatalf("fake output after refused URL: %v",err)
	}
	exit:=7
	if err=st.SaveExecOutput(ctx,r.ID,"/1.0/instances/owned-alpha/logs/exec-output/stdout",
		"/1.0/instances/owned-alpha/logs/exec-output/stderr",&exit);err!=nil {t.Fatal(err)}
	output,err:=st.ExecOutput(ctx,r.ID)
	if err!=nil||output.ExitCode==nil||*output.ExitCode!=7 || output.StdoutPath=="" {
		t.Fatalf("nonzero guest exit/output evidence was lost: %+v %v",output,err)
	}
}

func TestS2RecoverUndispatchedDoesNotReplay(t *testing.T){
	ctx:=context.Background()
	st:=newStore(t)
	_,token,err:=st.IssueToken(ctx,"admin",time.Hour)
	if err!=nil{t.Fatal(err)}
	r,err:=st.Admit(ctx,Admission{Layer:"project-s2",ActorID:token.ID,TokenID:token.ID,
		Key:"undispatched-process-crash",Action:"box_create",ResourceKey:"box:new",
		SafeIntent:[]byte(`{"name":"new"}`),RequestID:"original"})
	if err!=nil{t.Fatal(err)}
	n,err:=st.Recover(ctx)
	if err!=nil||n!=1{t.Fatalf("recover=%d err=%v",n,err)}
	got,err:=st.Get(ctx,r.ID)
	if err!=nil||got.State!="FAILED"||got.ErrorClass!="PROCESS_RESTART_BEFORE_DISPATCH"{
		t.Fatalf("queued crash was replayed/left locked: %+v %v",got,err)
	}
	if err:=st.Claim(ctx,r.ID);!errors.Is(err,ErrState){t.Fatalf("undispatched task re-claimed: %v",err)}
}

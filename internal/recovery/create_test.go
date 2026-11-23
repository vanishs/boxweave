package recovery

import (
	"context"
	"errors"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/lxc/incus/v7/shared/api"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
)

type stagedCreateWitness struct {
	witness
	binding                      native.VerifiedStaticPeerBinding
	idleErr, pinErr, pinCheckErr error
	idleChecks                   *int
	idleFailAt                   int
}

func (w stagedCreateWitness) ObserveOriginalCreateStaticPeerBinding(context.Context, string, string, string) (native.VerifiedStaticPeerBinding, error) {
	return w.binding, w.pinErr
}
func (w stagedCreateWitness) VerifyPinnedDeleteNIC(_ context.Context, name, uuid, bridge, acl, mac, v4, v6 string) error {
	return w.pinCheckErr
}
func (w stagedCreateWitness) OriginalOperationInactive(context.Context, string) error { return nil }
func (w stagedCreateWitness) ObserveCreateRecoveryIdle(context.Context) error {
	if w.idleChecks != nil {
		(*w.idleChecks)++
		if *w.idleChecks >= w.idleFailAt {
			return errors.New("late active operation")
		}
	}
	return w.idleErr
}

func TestN3StagedCreateRecoveryProofMatrix(t *testing.T) {
	for _, tc := range []struct {
		name, reason   string
		change         func(*stagedCreateWitness)
		noAck, noStage bool
	}{
		{"terminal", "CREATE_ORIGINAL_POSTSTATE_VERIFIED", nil, false, false},
		{"expired", "CREATE_ORIGINAL_POSTSTATE_VERIFIED", func(w *stagedCreateWitness) { w.opErr = api.StatusErrorf(http.StatusNotFound, "expired") }, false, false},
		{"active", "CREATE_ORIGINAL_NATIVE_SUCCESS_UNVERIFIED", func(w *stagedCreateWitness) { w.op.StatusCode = api.Running }, false, false},
		{"shape", "CREATE_ORIGINAL_POSTSTATE_UNVERIFIED", func(w *stagedCreateWitness) { w.box.CPU = "3" }, false, false},
		{"reservation", "CREATE_STATIC_RESERVATION_OR_LEDGER_CONFLICT", func(w *stagedCreateWitness) { w.binding.IPv4 = "10.243.18.12" }, false, false},
		{"missing ack", "CREATE_ACK_LOST_STATIC_POSTSTATE_VERIFIED", nil, true, false},
		{"missing ack active", "CREATE_ORIGINAL_NATIVE_ACK_UNPROVED", func(w *stagedCreateWitness) { w.idleErr = errors.New("active") }, true, false},
		{"missing ack late active", "CREATE_ORIGINAL_NATIVE_ACK_UNPROVED", func(w *stagedCreateWitness) { checks := 0; w.idleChecks = &checks; w.idleFailAt = 2 }, true, false},
		{"missing ack wrong marker", "CREATE_ORIGINAL_POSTSTATE_UNVERIFIED", func(w *stagedCreateWitness) { w.markerErr = errors.New("wrong marker") }, true, false},
		{"missing ack no binding", "CREATE_STATIC_NATIVE_BINDING_UNVERIFIED", func(w *stagedCreateWitness) { w.pinErr = errors.New("missing native lease") }, true, false},
		{"missing ack mismatched reservation", "CREATE_STATIC_RESERVATION_OR_LEDGER_CONFLICT", func(w *stagedCreateWitness) { w.binding.IPv4 = "10.243.18.12" }, true, false},
		{"missing staged claim", "CREATE_STATIC_RESERVATION_MISSING", nil, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			store, e := state.Open(filepath.Join(t.TempDir(), "private", "state.db"))
			if e != nil {
				t.Fatal(e)
			}
			defer store.Close()
			_, tok, e := store.IssueToken(ctx, time.Hour)
			if e != nil {
				t.Fatal(e)
			}
			r, e := store.Admit(ctx, state.Admission{Layer: "layer", ActorID: tok.ID, TokenID: tok.ID, Key: "n3-create-witness-original", RequestID: "original", Action: "box_create", ResourceKey: "box:owned", SafeIntent: []byte(`{"name":"owned","cpu":2,"memory_mib":512,"disk_gib":10,"start":true,"nesting":false,"internet":true,"instance_options":{}}`)})
			if e != nil {
				t.Fatal(e)
			}
			if e = store.Claim(ctx, r.ID); e != nil {
				t.Fatal(e)
			}
			p := state.VerifiedCreateBinding{Bridge: "bwbr0", ACL: "bw-peer", MAC: "02:16:3e:11:22:33", IPv4: "10.243.18.11", IPv6: "fd43:18::11"}
			if !tc.noStage {
				if ok, e := store.StageCreateBinding(ctx, r.ID, "layer", "owned", p); e != nil || !ok {
					t.Fatal(ok, e)
				}
			}
			phase := "DISPATCHING"
			if !tc.noAck {
				if e = store.NativeAck(ctx, r.ID, nu); e != nil {
					t.Fatal(e)
				}
				phase = "VERIFYING"
			}
			if e = store.Unknown(ctx, r.ID, phase, "CREATE_UNCERTAIN"); e != nil {
				t.Fatal(e)
			}
			w := stagedCreateWitness{witness: known(), binding: native.VerifiedStaticPeerBinding(p)}
			w.box.RootDiskSize = "10GiB"
			w.box.Internet = true
			if tc.change != nil {
				tc.change(&w)
			}
			svc := Service{Store: store, Native: w, Layer: "layer"}
			result, e := svc.Inspect(ctx, r.ID, "")
			if e != nil || result.View.Reason != tc.reason || result.NativeRetried {
				t.Fatal(result, e)
			}
			want := "BLOCKED"
			if tc.reason == "CREATE_ORIGINAL_POSTSTATE_VERIFIED" || tc.reason == "CREATE_ACK_LOST_STATIC_POSTSTATE_VERIFIED" {
				want = "SAFE_RECOVER"
			}
			if result.View.Disposition != want || result.View.ClaimProtection != "RETAINED" {
				t.Fatal(result)
			}
			got, e := store.Get(ctx, r.ID)
			if e != nil || got.State != "UNKNOWN" || got.BoxUUID != "" {
				t.Fatal(got, e)
			}
			result, e = svc.Reconcile(ctx, state.RecoveryRequest{Layer: "layer", ActorID: tok.ID, TokenID: tok.ID, OriginalID: r.ID, Key: "n3-create-reconcile-witness"})
			if e != nil || result.Recovery == nil || result.NativeRetried {
				t.Fatal(result, e)
			}
			got, e = store.Get(ctx, r.ID)
			if want == "SAFE_RECOVER" {
				if e != nil || got.State != "SUCCEEDED" || got.BoxUUID != bu || (tc.noAck && got.NativeOperationID != "") || result.Recovery.Disposition != "SETTLED" {
					t.Fatal(got, result, e)
				}
			} else {
				if e != nil || got.State != "UNKNOWN" || got.BoxUUID != "" || result.Recovery.Disposition != "BLOCKED" {
					t.Fatal(got, result, e)
				}
			}
		})
	}
}

func TestN3CompletedCreateNativePinDiagnosis(t *testing.T) {
	ctx := context.Background()
	st, e := state.Open(filepath.Join(t.TempDir(), "secure", "state.db"))
	if e != nil {
		t.Fatal(e)
	}
	defer st.Close()
	_, tok, e := st.IssueToken(ctx, time.Hour)
	if e != nil {
		t.Fatal(e)
	}
	original, e := st.Admit(ctx, state.Admission{Layer: "layer", ActorID: tok.ID, TokenID: tok.ID, Key: "n3-native-health-original-create", RequestID: "create", Action: "box_create", ResourceKey: "box:owned", SafeIntent: []byte(`{"name":"owned","cpu":2,"memory_mib":512,"disk_gib":10,"start":true,"nesting":false,"internet":true,"instance_options":{}}`)})
	if e != nil {
		t.Fatal(e)
	}
	if e = st.Claim(ctx, original.ID); e != nil {
		t.Fatal(e)
	}
	pin := state.VerifiedCreateBinding{Bridge: "bwbr0", ACL: "bw-peer", MAC: "02:16:3e:11:22:33", IPv4: "10.243.18.11", IPv6: "fd43:18::11"}
	if ok, e := st.StageCreateBinding(ctx, original.ID, "layer", "owned", pin); e != nil || !ok {
		t.Fatal(ok, e)
	}
	if e = st.NativeAck(ctx, original.ID, nu); e != nil {
		t.Fatal(e)
	}
	if e = st.Unknown(ctx, original.ID, "VERIFYING", "TERMINAL_ACK_LOST"); e != nil {
		t.Fatal(e)
	}
	target, e := st.ReadRecoveryTarget(ctx, "layer", original.ID)
	if e != nil {
		t.Fatal(e)
	}
	req := state.RecoveryRequest{Layer: "layer", ActorID: tok.ID, TokenID: tok.ID, OriginalID: original.ID, Key: "n3-native-health-original-settle"}
	proof := state.RecoveryProof{Kind: state.RecoverySettle, Reason: "CREATE_ORIGINAL_POSTSTATE_VERIFIED", NativeID: nu, UUID: bu, Fingerprint: "verified", CreateBinding: pin}
	if _, e = st.CommitRecovery(ctx, req, target, proof); e != nil {
		t.Fatal(e)
	}
	for _, tc := range []struct {
		name, reason string
		change       func(*stagedCreateWitness)
	}{
		{"match", "CREATE_PIN_LEDGER_COMPLETE", nil},
		{"drift", "CREATE_NATIVE_PIN_DRIFT", func(w *stagedCreateWitness) { w.pinCheckErr = native.ErrUnsafeNetwork }},
		{"read failure", "CREATE_NATIVE_PIN_UNVERIFIED", func(w *stagedCreateWitness) { w.pinCheckErr = native.ErrUnverified }},
		{"lease tuple mismatch", "CREATE_NATIVE_PIN_DRIFT", func(w *stagedCreateWitness) { w.binding.IPv4 = "10.243.18.99" }},
		{"lease unavailable", "CREATE_NATIVE_PIN_UNVERIFIED", func(w *stagedCreateWitness) { w.pinErr = native.ErrUnverified }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := stagedCreateWitness{witness: known(), binding: native.VerifiedStaticPeerBinding(pin)}
			if tc.change != nil {
				tc.change(&w)
			}
			r, e := (Service{Store: st, Layer: "layer", Native: w}).Inspect(ctx, original.ID, "")
			if e != nil || r.View.Reason != tc.reason || r.NativeRetried {
				t.Fatal(r, e)
			}
			want := "BLOCKED"
			if tc.name == "match" {
				want = state.RecoveryNotNeeded
			}
			if r.View.Disposition != want {
				t.Fatal(r)
			}
			got, e := st.Get(ctx, original.ID)
			if e != nil || got.State != "SUCCEEDED" || got.BoxUUID != bu {
				t.Fatal(got, e)
			}
		})
	}
}

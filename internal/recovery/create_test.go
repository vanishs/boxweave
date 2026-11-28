package recovery

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/peerscope"
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

const scopedProject = "user-1001"
const scopedBridge = "incusbr-1001"

func scopedCreateFixture(t *testing.T, ack, stage bool) (*state.Store, string, string, state.Receipt, state.VerifiedCreateBinding) {
	t.Helper()
	ctx := context.Background()
	dir := t.TempDir()
	if e := os.Chmod(dir, 0700); e != nil {
		t.Fatal(e)
	}
	path := filepath.Join(dir, "ledger.sqlite")
	s, e := state.Open(path)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { s.Close() })
	_, tok, e := s.IssueToken(ctx, time.Hour)
	if e != nil {
		t.Fatal(e)
	}
	in := state.ScopedCreateIntent{Schema: 2, Project: scopedProject, Bridge: scopedBridge, Name: "owned", CPU: 2, MemoryMiB: 512, DiskGiB: 10, Start: true, Internet: true, BootAutostart: true, Options: json.RawMessage("{}")}
	raw, e := json.Marshal(in)
	if e != nil {
		t.Fatal(e)
	}
	r, e := s.AdmitScoped(ctx, state.Admission{Layer: in.Project, ActorID: tok.ID, TokenID: tok.ID, Key: "n3-create-witness-original", RequestID: "original", Action: "box_create", ResourceKey: "box:owned", SafeIntent: raw})
	if e != nil {
		t.Fatal(e)
	}
	if e = s.Claim(ctx, r.ID); e != nil {
		t.Fatal(e)
	}
	pin := state.VerifiedCreateBinding{Bridge: scopedBridge, ACL: peerscope.QuarantineACL, MAC: "02:16:3e:11:22:33", IPv4: "10.243.18.11", IPv6: "fd43:18::11"}
	if stage {
		if ok, e := s.StageCreateBinding(ctx, r.ID, in.Project, in.Name, pin); e != nil || !ok {
			t.Fatal(ok, e)
		}
	}
	if ack {
		if e = s.NativeAck(ctx, r.ID, nu); e != nil {
			t.Fatal(e)
		}
	}
	pin.ACL, e = peerscope.ACLName(in.Project, in.Bridge, bu)
	if e != nil {
		t.Fatal(e)
	}
	return s, path, tok.ID, r, pin
}

func scopedCreateNativeWitness(pin state.VerifiedCreateBinding) stagedCreateWitness {
	w := stagedCreateWitness{witness: known(), binding: native.VerifiedStaticPeerBinding(pin)}
	w.box.Project = scopedProject
	w.box.Internet = true
	w.box.RootDiskSize = "10GiB"
	w.box.NativeOptions = map[string]string{"boot.autostart": "true"}
	return w
}
func TestN3CompletedCreateNativePinDiagnosis(t *testing.T) {
	ctx := context.Background()
	st, _, _, original, pin := scopedCreateFixture(t, true, true)
	if e := st.FinalizeVerifiedScopedCreate(ctx, original.ID, scopedProject, "owned", bu, pin); e != nil {
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
			w := scopedCreateNativeWitness(pin)
			if tc.change != nil {
				tc.change(&w)
			}
			r, e := (Service{Store: st, Layer: scopedProject, Native: w}).Inspect(ctx, original.ID, "")
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

func TestN3StagedCreateRecoveryProofMatrix(t *testing.T) {
	for _, tc := range []struct{ name, reason string }{
		{"complete", "SCOPED_LIFECYCLE_COMPLETED_NO_REPLAY"},
		{"missing-ack", "SCOPED_LIFECYCLE_ORIGINAL_ACK_UNPROVED"},
		{"missing-stage", "SCOPED_LIFECYCLE_ORIGINAL_OR_CLAIMS_CHANGED"},
		{"partial-root", "SCOPED_LIFECYCLE_ROOT_PROOF_REQUIRED"},
		{"wrong-original", "SCOPED_LIFECYCLE_ROOT_PROOF_REQUIRED"},
		{"wrong-native", "SCOPED_LIFECYCLE_ROOT_PROOF_REQUIRED"},
		{"wrong-digest", "SCOPED_LIFECYCLE_ORIGINAL_OR_CLAIMS_CHANGED"},
		{"wrong-intent", "SCOPED_CREATE_INTENT_UNVERIFIED"},
		{"shape", "SCOPED_CREATE_FINAL_POSTSTATE_UNVERIFIED"},
		{"marker", "SCOPED_CREATE_FINAL_POSTSTATE_UNVERIFIED"},
		{"native-pin", "SCOPED_CREATE_BINDING_UNVERIFIED"},
		{"reservation", "SCOPED_LIFECYCLE_ORIGINAL_OR_CLAIMS_CHANGED"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			ack := tc.name != "missing-ack"
			stage := tc.name != "missing-stage"
			s, path, tok, r, pin := scopedCreateFixture(t, ack, stage)
			phase := "DISPATCHING"
			if ack {
				phase = "VERIFYING"
			}
			if e := s.Unknown(ctx, r.ID, phase, "CREATE_UNCERTAIN"); e != nil {
				t.Fatal(e)
			}
			if e := os.Chmod(path, 0600); e != nil {
				t.Fatal(e)
			}
			ro, e := state.OpenNetworkAuditReadOnly(path)
			if e != nil {
				t.Fatal(e)
			}
			defer ro.Close()
			sha := strings.Repeat("d", 64)
			if ack && stage {
				orig, e := ro.ReadRootScopedCreateUnknownOriginal(ctx, r.ID, scopedProject, scopedBridge)
				if e != nil {
					t.Fatal(e)
				}
				sha = state.ScopedLifecycleOriginalSHA256(orig)
			}
			w := scopedCreateNativeWitness(pin)
			proof := state.ScopedLifecycleRecoveryProof{RecoveryOnly: true, OriginalSHA256: sha, Record: state.ScopedLifecycleRecord{Schema: 2, Kind: "CREATE", Status: "VERIFIED", OriginalID: r.ID, NativeID: nu, Project: scopedProject, Name: "owned", UUID: bu, Binding: pin, Start: true, Internet: true, BootAutostart: true, HostSHA256: strings.Repeat("a", 64), GuardSHA256: strings.Repeat("b", 64), PlanSHA256: strings.Repeat("c", 64)}}
			switch tc.name {
			case "partial-root":
				proof.Record.Status = "ACL_VERIFIED"
			case "wrong-original":
				proof.Record.OriginalID = strings.Repeat("f", 32)
			case "wrong-native":
				proof.Record.NativeID = "other"
			case "wrong-digest":
				proof.OriginalSHA256 = strings.Repeat("f", 64)
			case "wrong-intent":
				proof.Record.BootAutostart = false
			case "shape":
				w.box.CPU = "3"
			case "marker":
				w.markerErr = errors.New("wrong marker")
			case "native-pin":
				w.pinCheckErr = native.ErrUnsafeNetwork
			case "reservation":
				proof.Record.Binding.IPv4 = "10.243.18.99"
			}
			calls := 0
			svc := Service{Store: s, Layer: scopedProject, Native: w, ScopedLifecycleWitness: func(_ context.Context, id string) (state.ScopedLifecycleRecoveryProof, error) {
				calls++
				if id != r.ID {
					t.Fatal("wrong original")
				}
				return proof, nil
			}}
			result, e := svc.Inspect(ctx, r.ID, "")
			if e != nil || result.View.Reason != tc.reason || result.NativeRetried || result.View.ClaimProtection != "RETAINED" {
				t.Fatal(result, e)
			}
			got, e := s.Get(ctx, r.ID)
			if e != nil || got.State != "UNKNOWN" || got.BoxUUID != "" {
				t.Fatal(got, e)
			}
			result, e = svc.Reconcile(ctx, state.RecoveryRequest{Layer: scopedProject, ActorID: tok, TokenID: tok, OriginalID: r.ID, Key: "n3-create-reconcile-witness"})
			if e != nil || result.Recovery == nil || result.NativeRetried {
				t.Fatal(result, e)
			}
			got, e = s.Get(ctx, r.ID)
			if tc.name == "complete" {
				if e != nil || got.State != "SUCCEEDED" || got.BoxUUID != bu || result.Recovery.Disposition != "SETTLED" || calls != 2 {
					t.Fatal(got, result, e, calls)
				}
			} else if e != nil || got.State != "UNKNOWN" || got.BoxUUID != "" || result.Recovery.Disposition != "BLOCKED" || !ack && calls != 0 {
				t.Fatal(got, result, e, calls)
			}
		})
	}
}

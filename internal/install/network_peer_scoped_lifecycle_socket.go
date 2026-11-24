package install

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"os/user"
	"strconv"
	"time"

	"github.com/vanishs/boxweave/internal/state"
)

const scopedLifecycleSocket = "/run/boxweave-scoped-lifecycle.sock"

type ScopedLifecycleReceipt = scopedLifecycleRecord

// Fixed installed Root server: only original IDs arrive from the restricted
// controller. Scope, DB, policy, native identity and effect direction are
// independently derived from protected local state.
func NativeScopedLifecycleServe(ctx context.Context) error {
	r, _, err := nativePeerFreshVerifiedPlan()
	if err != nil {
		return err
	}
	if err = nativePeerFreshQuarantineReceipt(r); err != nil {
		return err
	}
	if _, err = nativePeerFreshGuardCurrent(ctx, r.Project, r.Bridge); err != nil {
		return err
	}
	u, err := user.Lookup("boxweave")
	if err != nil {
		return err
	}
	uid, err := strconv.ParseUint(u.Uid, 10, 32)
	if err != nil || uid == 0 || r.Project != "user-"+u.Uid {
		return errors.New("scoped lifecycle actor mismatch")
	}
	gid, err := strconv.Atoi(u.Gid)
	if err != nil || gid < 1 {
		return errors.New("scoped lifecycle group mismatch")
	}
	db, err := state.OpenNetworkAuditReadOnly(nativePeerRootInstalledDB)
	if err != nil {
		return err
	}
	defer db.Close()
	if err = nativePeerRootReclaimStaleSocket(scopedLifecycleSocket, 0, uint32(gid)); err != nil {
		return err
	}
	ln, err := net.ListenUnix("unix", &net.UnixAddr{Name: scopedLifecycleSocket, Net: "unix"})
	if err != nil {
		return err
	}
	ln.SetUnlinkOnClose(false)
	old, err := os.Lstat(scopedLifecycleSocket)
	if err != nil {
		ln.Close()
		return err
	}
	defer func() {
		ln.Close()
		if now, e := os.Lstat(scopedLifecycleSocket); e == nil && os.SameFile(old, now) {
			os.Remove(scopedLifecycleSocket)
		}
	}()
	if err = os.Chown(scopedLifecycleSocket, 0, gid); err != nil {
		return err
	}
	if err = os.Chmod(scopedLifecycleSocket, 0660); err != nil {
		return err
	}
	return nativePeerRootServeListener(ctx, ln, uint32(uid), func(callCtx context.Context, req NativePeerRootRequest) (NativePeerRootResponse, error) {
		if req != (NativePeerRootRequest{OperationID: req.OperationID}) {
			return NativePeerRootResponse{}, errors.New("scoped lifecycle accepts only an exact original ID")
		}
		orig, err := db.Get(callCtx, req.OperationID)
		if err != nil {
			return NativePeerRootResponse{}, errors.New("unverified lifecycle original")
		}
		if orig.Action == "box_peer_set" {
			var rec state.ScopedPeerRootReceipt
			switch orig.State {
			case "DISPATCHING":
				rec, err = nativeScopedPeerApply(callCtx, db, req.OperationID)
			case "UNKNOWN":
				rec, err = nativeScopedPeerRecover(callCtx, db, req.OperationID)
			default:
				return NativePeerRootResponse{}, errors.New("unverified Peer original")
			}
			if err != nil {
				return NativePeerRootResponse{}, err
			}
			return NativePeerRootResponse{Status: rec.Status, ScopedPeer: &rec}, nil
		}
		if orig.State != "VERIFYING" {
			return NativePeerRootResponse{}, errors.New("unverified lifecycle phase")
		}
		var receipt scopedLifecycleRecord
		switch orig.Action {
		case "box_create":
			receipt, err = nativeScopedCreateApply(callCtx, db, req.OperationID)
		case "box_delete":
			receipt, err = nativeScopedDeleteApply(callCtx, db, req.OperationID)
		default:
			return NativePeerRootResponse{}, errors.New("unsupported scoped lifecycle action")
		}
		if err != nil {
			return NativePeerRootResponse{}, err
		}
		return NativePeerRootResponse{Status: receipt.Status, ScopedLifecycle: &receipt}, nil
	})
}

func NativeScopedLifecycleCall(ctx context.Context, original, kind string) (ScopedLifecycleReceipt, error) {
	var zero ScopedLifecycleReceipt
	if kind != "CREATE" && kind != "DELETE" {
		return zero, errors.New("invalid lifecycle kind")
	}
	out, err := nativeScopedRootCall(ctx, original)
	if err != nil {
		return zero, err
	}
	r := out.ScopedLifecycle
	if out.ScopedPeer != nil || out.Error != "" || out.Status != "VERIFIED" || r == nil || !scopedLifecycleValid(*r) || r.Status != "VERIFIED" || r.Kind != kind || r.OriginalID != original {
		return zero, errors.New("scoped lifecycle root outcome unverified")
	}
	return *r, nil
}

func NativeScopedPeerCall(ctx context.Context, original string) (state.ScopedPeerRootReceipt, error) {
	var zero state.ScopedPeerRootReceipt
	out, e := nativeScopedRootCall(ctx, original)
	if e != nil {
		return zero, e
	}
	r := out.ScopedPeer
	if out.ScopedLifecycle != nil || out.Error != "" || out.Status != "COMPLETE_VERIFIED" || r == nil || r.OriginalID != original || r.Status != out.Status || r.VerifiedSteps != 4 {
		return zero, errors.New("scoped Peer root outcome unverified")
	}
	return *r, nil
}

func nativeScopedRootCall(ctx context.Context, original string) (NativePeerRootResponse, error) {
	var zero NativePeerRootResponse
	if !scopedLifecycleID.MatchString(original) {
		return zero, errors.New("invalid scoped lifecycle original")
	}
	var dial net.Dialer
	conn, err := dial.DialContext(ctx, "unix", scopedLifecycleSocket)
	if err != nil {
		return zero, err
	}
	defer conn.Close()
	uc, ok := conn.(*net.UnixConn)
	if !ok || nativePeerRootPeerUID(uc, 0) != nil {
		return zero, errors.New("unverified root lifecycle peer")
	}
	deadline := time.Now().Add(215 * time.Second)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	if err = uc.SetDeadline(deadline); err != nil {
		return zero, err
	}
	stop := context.AfterFunc(ctx, func() { uc.Close() })
	defer stop()
	if err = json.NewEncoder(uc).Encode(NativePeerRootRequest{OperationID: original}); err != nil {
		return zero, err
	}
	if err = uc.CloseWrite(); err != nil {
		return zero, err
	}
	dec := json.NewDecoder(io.LimitReader(uc, 8192))
	dec.DisallowUnknownFields()
	var out NativePeerRootResponse
	if err = dec.Decode(&out); err != nil {
		return zero, err
	}
	var trailing any
	if err = dec.Decode(&trailing); err != io.EOF {
		return zero, errors.New("ambiguous root lifecycle reply")
	}
	return out, nil
}

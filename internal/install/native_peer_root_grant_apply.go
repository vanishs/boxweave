package install

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"time"

	incus "github.com/lxc/incus/v7/client"
	"github.com/lxc/incus/v7/shared/api"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
)

// NativePeerRootGrantApplyOperation is the B3 grant branch of the B2 root
// original journal. A failed or interrupted native effect remains UNKNOWN;
// it never returns a speculative grant or exposes a general ACL/root API.
func NativePeerRootGrantApplyOperation(ctx context.Context, s *state.Store,
	project, bridge, acl, original, approvedSHA string, approved bool) (NativePeerConntrackJournalReceipt, error) {
	return nativePeerRootGrantApplyWithTestFault(ctx, s, project, bridge, acl, original, approvedSHA, approved, nil, nil)
}

// Fault seam is package-private and is never available from IPC/production.
// It lets native tests prove that a real completed ETag PUT followed by an
// uncertain response retains stopped NICs and the DISPATCH_UNKNOWN journal.
func nativePeerRootGrantApplyWithTestFault(ctx context.Context, s *state.Store,
	project, bridge, acl, original, approvedSHA string, approved bool, afterACL, afterResume func() error) (NativePeerConntrackJournalReceipt, error) {
	return nativePeerRootGrantApplyWithCompletionFault(ctx, s, project, bridge, acl,
		original, approvedSHA, approved, afterACL, afterResume, nil)
}

// Private test-only fault before the final COMPLETE journal write. Production
// never accepts this hook from the Controller, MCP or root Unix socket.
func nativePeerRootGrantApplyWithCompletionFault(ctx context.Context, s *state.Store,
	project, bridge, acl, original, approvedSHA string, approved bool,
	afterACL, afterResume, beforeComplete func() error) (NativePeerConntrackJournalReceipt, error) {
	return nativePeerRootGrantApplyWithContentionFault(ctx, s, project, bridge, acl,
		original, approvedSHA, approved, afterACL, afterResume, beforeComplete, nil)
}

// The last hook runs ONLY in a package-private native test, after the entire
// Project is isolated and native ACL preimage is read, but before the ETag PUT.
// Production and the root IPC invoke the wrapper above with nil.
func nativePeerRootGrantApplyWithContentionFault(ctx context.Context, s *state.Store,
	project, bridge, acl, original, approvedSHA string, approved bool,
	afterACL, afterResume, beforeComplete, beforePUT func() error) (NativePeerConntrackJournalReceipt, error) {
	var zero NativePeerConntrackJournalReceipt
	if !approved || os.Geteuid() != 0 || s == nil || !s.IsNetworkAuditReadOnly() {
		return zero, errors.New("grant root requires operator approval and read-only state")
	}
	path, err := nativePeerConntrackOperationJournalPath(original)
	if err != nil {
		return zero, err
	}
	unlock, err := nativeScriptletAcquireLock(nativePeerConntrackLockPath)
	if err != nil {
		return zero, err
	}
	defer unlock()
	pre, err := nativePeerRootGrantReview(ctx, s, project, bridge, acl, original)
	if err != nil {
		return zero, err
	}
	if approvedSHA == "" || approvedSHA != pre.Plan.PlanSHA256 {
		return zero, errors.New("grant root operator approval pre-isolation digest mismatch")
	}
	if err := nativePeerConntrackPrepareOriginalJournalDirectory(); err != nil {
		return zero, err
	}
	return nativePeerConntrackDispatchOnceGuarded(ctx, path, pre.Plan, approvedSHA, approved,
		func(ctx context.Context) (NativePeerConntrackReport, error) {
			root, err := incus.ConnectIncusUnixWithContext(ctx, "/var/lib/incus/unix.socket", nil)
			if err != nil {
				return NativePeerConntrackReport{}, err
			}
			check := func(ctx context.Context) error {
				if _, err := nativePeerBridgeAuditFromStoreWithClient(ctx, s, root, project, bridge, acl); err != nil {
					return err
				}
				global, err := nativePeerGlobalNetworkScopeSnapshot(ctx, root, pre.Target)
				if err != nil || global != pre.GlobalSHA {
					return errors.New("global grant native scope drift")
				}
				routes, err := nativePeerHostRoutes(ctx, pre.Target)
				if err != nil || routes != pre.RoutesSHA {
					return errors.New("grant host routes changed")
				}
				proof, err := s.ReadRootPeerGrant(ctx, original, project, bridge, acl)
				if err != nil || !reflect.DeepEqual(proof, pre.Proof) {
					return errors.New("grant owner/claims/whole allowlist changed during execution")
				}
				return nil
			}
			if err := native.IsolateSharedPeerACLWithClient(ctx, root, project, bridge, acl, pre.Target.Boxes, check); err != nil {
				return NativePeerConntrackReport{}, err
			}
			if err := nativePeerConntrackAllStopped(ctx, root, pre.Target); err != nil {
				return NativePeerConntrackReport{}, err
			}
			if err := nativePeerBridgeKernelNoLivePorts(ctx, bridge); err != nil {
				return NativePeerConntrackReport{}, err
			}
			current, etag, err := root.GetNetworkACL(acl)
			if err != nil || current == nil || etag != pre.ACLTag ||
				!nativePeerACLMatches(current, pre.Before) {
				return NativePeerConntrackReport{}, errors.New("grant ACL baseline drifted across stop barrier")
			}
			nw, _, err := root.GetNetwork(bridge)
			if err != nil {
				return NativePeerConntrackReport{}, err
			}
			want, err := NativePeerACLFromPolicySnapshot(NativePrepareTarget{Project: project, ACL: acl}, nw,
				pre.Proof.Target, pre.Proof.Revoked)
			if err != nil || !reflect.DeepEqual(want, pre.After) {
				return NativePeerConntrackReport{}, errors.New("bridge config or grant compiler target drifted")
			}
			if err := check(ctx); err != nil {
				return NativePeerConntrackReport{}, err
			}
			if beforePUT != nil {
				if err := beforePUT(); err != nil {
					return NativePeerConntrackReport{}, fmt.Errorf("native grant private pre-PUT contender: %w", err)
				}
			}
			if err := root.UpdateNetworkACL(acl, pre.After.NetworkACLPut, etag); err != nil {
				return NativePeerConntrackReport{}, fmt.Errorf("native grant ETag fanout uncertain: %w", err)
			}
			updated, _, err := root.GetNetworkACL(acl)
			if err != nil || updated == nil || !nativePeerACLMatches(updated, pre.After) {
				return NativePeerConntrackReport{}, errors.New("native grant ACL whole-policy readback mismatch")
			}
			if err := check(ctx); err != nil {
				return NativePeerConntrackReport{}, err
			}
			if afterACL != nil {
				if err := afterACL(); err != nil {
					return NativePeerConntrackReport{}, fmt.Errorf("post-grant ACL result uncertain: %w", err)
				}
			}
			// The B2 revoke conntrack helper deliberately requires a DROP.
			// Instead purge only four original pinned tuples while the entire
			// protected bridge has no live ports. This must never be global -F.
			report, err := nativePeerConntrackPurgeExact(ctx, nativePeerConntrackOS, pre.Source, pre.Destination)
			report.NativeDigestSHA256 = pre.Plan.NativeDigestSHA256
			if err != nil {
				return report, err
			}
			if err := nativePeerConntrackAllStopped(ctx, root, pre.Target); err != nil {
				return report, err
			}
			if err := nativePeerBridgeKernelNoLivePorts(ctx, bridge); err != nil {
				return report, err
			}
			post, err := nativePeerBridgeAuditFromStoreWithClient(ctx, s, root, project, bridge, acl)
			if err != nil {
				return report, err
			}
			report.PostNativeDigestSHA256 = post.NativeDigestSHA256
			if err := check(ctx); err != nil {
				return report, err
			}

			// The last fallible checks must run inside the resume stop-on-error
			// guard, never after it has declared resumption successful.
			finalCheck := func(ctx context.Context) error {
				if err := check(ctx); err != nil {
					return err
				}
				applied, _, err := root.GetNetworkACL(acl)
				if err != nil || applied == nil ||
					!nativePeerACLMatches(applied, pre.After) {
					return errors.New("grant final resumed ACL readback mismatch")
				}
				rules, err := nativePeerRootKernelRules(ctx)
				if err != nil {
					return err
				}
				if err := nativePeerRootVerifyGrantNFT(rules, pre.Target.Project, pre.Target.Boxes,
					pre.Running, pre.Proof.Target); err != nil {
					return err
				}
				if err := nativePeerRootVerifyRevokedNFT(rules, pre.Target.Project, pre.Running,
					pre.Proof.Revoked); err != nil {
					return err
				}
				if afterResume != nil {
					if err := afterResume(); err != nil {
						return fmt.Errorf("injected final grant check failed: %w", err)
					}
				}
				return check(ctx)
			}
			count, err := nativePeerRootResumeGrantVerified(ctx, root, pre, finalCheck)
			if err != nil {
				return report, err
			}
			report.VerifiedNICs = count
			report.KernelVerified = true
			return report, nil
		}, beforeComplete, func() error {
			return nativePeerRootContainFailedCompletion(pre)
		})
}

// A completed resume is not a completed mutation if the journal cannot
// durably commit. Re-identify each NIC by UUID before any stop by name;
// never stop a replacement/foreign object. Failure retains UNKNOWN/claims.
func nativePeerRootContainFailedCompletion(pre nativePeerGrantPreflight) error {
	return nativePeerRootContainFailedTarget(pre.Target)
}

// Both grant and revoke must re-isolate after ambiguous final journal writes.
func nativePeerRootContainFailedTarget(target NativePeerBridgeAuditTarget) error {
	stopCtx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	root, err := incus.ConnectIncusUnixWithContext(stopCtx, "/var/lib/incus/unix.socket", nil)
	if err != nil {
		return fmt.Errorf("post-resume journal failure could not reach native Incus: %w", err)
	}
	owned := root.UseProject(target.Project)
	var failures []error
	for _, box := range target.Boxes {
		inst, _, err := owned.GetInstance(box.Name)
		if err != nil {
			failures = append(failures, fmt.Errorf("missing exact protected instance %s: %w", box.Name, err))
			continue
		}
		if err := native.ValidatePeerACLBox(inst, target.Project, target.Bridge, target.ACL, box, false); err != nil {
			failures = append(failures, fmt.Errorf("protected NIC identity drift %s: %w", box.Name, err))
			continue
		}
		if inst.Status == "Stopped" && inst.Config["boot.autostart"] == "false" {
			continue
		}
		op, err := owned.UpdateInstanceState(box.Name, api.InstanceStatePut{Action: "stop", Timeout: 20}, "")
		if err != nil || op == nil {
			failures = append(failures, fmt.Errorf("protected NIC journal-failure stop %s uncertain: %v", box.Name, err))
			continue
		}
		if err := op.WaitContext(stopCtx); err != nil {
			failures = append(failures, fmt.Errorf("protected NIC journal-failure stop ACK %s: %w", box.Name, err))
		}
	}
	if err := nativePeerConntrackAllStopped(stopCtx, root, target); err != nil {
		failures = append(failures, fmt.Errorf("journal-failure NIC status remains unsafe: %w", err))
	}
	if err := nativePeerBridgeKernelNoLivePorts(stopCtx, target.Bridge); err != nil {
		failures = append(failures, fmt.Errorf("journal-failure live bridge ports: %w", err))
	}
	return errors.Join(failures...)
}

// A Start SDK failure can mean that a guest actually resumed. On ANY
// uncertainty, attempt to STOP every formerly protected NIC, including the
// member whose start was ambiguous. Root journal and SQLite remain UNKNOWN.
func nativePeerRootResumeGrantVerified(ctx context.Context, root incus.InstanceServer,
	pre nativePeerGrantPreflight, finalCheck func(context.Context) error) (int, error) {
	if finalCheck == nil || len(pre.Running) < 2 {
		return 0, errors.New("grant resume needs final checks and both live endpoints")
	}
	owned := root.UseProject(pre.Target.Project)
	var resumed []string
	success := false
	defer func() {
		if success {
			return
		}
		stopCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		// The original request context may already be cancelled.
		if fresh, e := incus.ConnectIncusUnixWithContext(stopCtx, "/var/lib/incus/unix.socket", nil); e == nil {
			owned = fresh.UseProject(pre.Target.Project)
		}
		for _, box := range pre.Target.Boxes {
			op, e := owned.UpdateInstanceState(box.Name, api.InstanceStatePut{Action: "stop", Timeout: 20}, "")
			if e == nil && op != nil {
				_ = op.WaitContext(stopCtx)
			}
		}
	}()
	for _, name := range pre.Running {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		op, err := owned.UpdateInstanceState(name, api.InstanceStatePut{Action: "start", Timeout: 30}, "")
		if err != nil || op == nil {
			return 0, fmt.Errorf("native grant Start uncertain for %q: %w", name, err)
		}
		resumed = append(resumed, name)
		if err := op.WaitContext(ctx); err != nil {
			return 0, err
		}
		var box native.PeerACLBox
		for _, b := range pre.Target.Boxes {
			if b.Name == name {
				box = b
				break
			}
		}
		inst, _, err := owned.GetInstance(name)
		if err != nil {
			return 0, err
		}
		if err := native.ValidatePeerACLBox(inst, pre.Target.Project, pre.Target.Bridge, pre.Target.ACL, box, false); err != nil {
			return 0, err
		}
		if (inst.Status != "Running" && inst.Status != "Ready") || inst.Config["boot.autostart"] != "false" {
			return 0, errors.New("native grant NIC resume or autostart ownership drift")
		}
		rules, err := nativePeerRootKernelRules(ctx)
		if err != nil {
			return 0, err
		}
		if err := nativePeerRootVerifyGrantNFT(rules, pre.Target.Project, pre.Target.Boxes, resumed, pre.Proof.Target); err != nil {
			return 0, err
		}
	}
	if err := finalCheck(ctx); err != nil {
		return 0, err
	}
	success = true
	return len(resumed), nil
}

// A successful later grant must preserve previously revoked pair DROP rules
// in each live protected Incus chain, not just the new positive allowlist.
func nativePeerRootVerifyRevokedNFT(rules, project string, running []string,
	revoked []state.PeerGrantBinding) error {
	denied, err := nativePeerBindingsFromSnapshot(revoked)
	if err != nil {
		return err
	}
	for _, p := range denied {
		if err := nativePeerRootVerifyNFT(rules, project, running, p); err != nil {
			return err
		}
	}
	return nil
}

package install

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"slices"

	incus "github.com/lxc/incus/v7/client"
	"github.com/lxc/incus/v7/shared/api"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/peerscope"
	"github.com/vanishs/boxweave/internal/state"
)

func nativeScopedDeleteReceipt(id string) (scopedLifecycleRecord, error) {
	var zero scopedLifecycleRecord
	path, e := scopedLifecyclePath(id)
	if e != nil {
		return zero, e
	}
	d, e := nativePeerFreshJournalOpenAt(path, 0)
	if e != nil {
		return zero, e
	}
	defer d.Close()
	a, e := scopedLifecycleRead(d, "DISPATCH_UNKNOWN")
	if e != nil || a.Kind != "DELETE" || a.OriginalID != id {
		return zero, errors.New("invalid original Root Delete intent")
	}
	b, e := scopedLifecycleRead(d, "VERIFIED")
	a.Status = "VERIFIED"
	if e != nil || a != b {
		return zero, errors.New("missing terminal Root Delete receipt")
	}
	return b, nil
}
func nativeScopedReceiptMatches(r nativePeerFreshJournalRecord, rec scopedLifecycleRecord, id, kind string, b state.ScopedNetworkAuditBinding) bool {
	return rec.Status == "VERIFIED" && rec.Kind == kind && rec.OriginalID == id && rec.Project == r.Project && rec.HostSHA256 == r.HostSHA256 && rec.GuardSHA256 == r.GuardSHA256 && rec.Name == b.Name && rec.UUID == b.UUID && b.UUID == b.Incarnation && rec.Binding == (state.VerifiedCreateBinding{Bridge: r.Bridge, ACL: b.ACL, MAC: b.MAC, IPv4: b.IPv4, IPv6: b.IPv6})
}

// Root-only host provenance witness. Retired addresses remain fenced; the
// actual retired UUID and ACL must be absent, not merely marked RELEASED.
// All current native owners must match their original Create receipts.
func nativeScopedPeerGlobal(ctx context.Context, c incus.InstanceServer, db *state.Store, r nativePeerFreshJournalRecord, proof state.ScopedPeerOriginalProof, s nativePeerScopedSDKSnapshot) (string, error) {
	if c == nil || db == nil || proof.Intent.Project != r.Project || proof.Intent.Bridge != r.Bridge {
		return "", errors.New("wrong scoped Root scope")
	}
	pf, pr, nw, e := nativePeerFreshQuarantineRead(ctx, c, r, "")
	if e != nil || !reflect.DeepEqual(nw, s.Network) {
		return "", errors.New("scoped root quarantine/bridge drift")
	}
	ledger, e := db.ReadNativeNetworkLedger(ctx, r.Project)
	if e != nil || ledger.OrphanBindings != 0 || ledger.OrphanFences != 0 || len(ledger.Boxes) != len(proof.Inventory.Boxes) {
		return "", errors.New("scoped native lineage incomplete")
	}
	var receipts []scopedLifecycleRecord
	claims := []native.PeerACLBox{}
	expected := []string{peerscope.QuarantineACL}
	for i, b := range proof.Inventory.Boxes {
		l := ledger.Boxes[i]
		if l.UUID != b.UUID || l.Incarnation != b.Incarnation || l.Name != b.Name || l.OriginalCount != 1 || !l.OriginalProof || l.BindingCount != 1 || l.BindingState != "PINNED" || l.Bridge != r.Bridge || l.ACL != b.ACL || l.MAC != b.MAC || l.IPv4 != b.IPv4 || l.IPv6 != b.IPv6 || l.FenceCount != 2 || l.FenceIPv4 != 1 || l.FenceIPv6 != 1 {
			return "", errors.New("scoped current owner lineage mismatch")
		}
		rec, e := nativeScopedCreateReceipt(l.OriginalID)
		o := s.Evidence.Observed[b.Name]
		if e != nil || !nativeScopedReceiptMatches(r, rec, l.OriginalID, "CREATE", b) || o.Instance == nil || o.Instance.Config["user.boxweave.create_operation"] != l.OriginalID {
			return "", errors.New("scoped current owner missing Root Create receipt")
		}
		if e = nativeScopedCreateLeaseCheck(c, r.Bridge, b.Name, rec.Binding, true); e != nil {
			return "", e
		}
		receipts = append(receipts, rec)
		expected = append(expected, b.ACL)
		claims = append(claims, native.PeerACLBox{Name: b.Name, UUID: b.UUID, MAC: b.MAC, IPv4: b.IPv4, IPv6: b.IPv6})
	}
	all, e := c.GetInstancesAllProjects(api.InstanceTypeAny)
	if e != nil || len(all) != len(proof.Inventory.Boxes) {
		return "", errors.New("foreign native instance inventory")
	}
	for _, inst := range all {
		o, ok := s.Evidence.Observed[inst.Name]
		if !ok || inst.Project != r.Project || o.Instance == nil || inst.Config["volatile.uuid"] != o.Instance.Config["volatile.uuid"] {
			return "", errors.New("foreign native instance")
		}
	}
	names, e := c.GetNetworkACLNames()
	if e != nil {
		return "", e
	}
	slices.Sort(names)
	slices.Sort(expected)
	if !slices.Equal(names, expected) {
		return "", errors.New("unowned root ACL inventory")
	}
	for _, old := range proof.Inventory.Retired {
		a, e := nativeScopedCreateReceipt(old.CreateID)
		if e != nil || !nativeScopedReceiptMatches(r, a, old.CreateID, "CREATE", old.ScopedNetworkAuditBinding) {
			return "", errors.New("retired Root Create receipt unverified")
		}
		b, e := nativeScopedDeleteReceipt(old.DeleteID)
		if e != nil || !nativeScopedReceiptMatches(r, b, old.DeleteID, "DELETE", old.ScopedNetworkAuditBinding) {
			return "", errors.New("retired Root Delete receipt unverified")
		}
		if _, _, e = c.GetNetworkACL(old.ACL); !api.StatusErrorCheck(e, http.StatusNotFound) {
			return "", errors.New("retired ACL absence unverified")
		}
		for _, inst := range all {
			if inst.Config["volatile.uuid"] == old.UUID {
				return "", errors.New("retired native UUID still present")
			}
		}
		receipts = append(receipts, a, b)
	}
	t := NativePeerBridgeAuditTarget{Project: r.Project, Bridge: r.Bridge, Boxes: claims}
	networks, e := nativePeerGlobalNetworkScopeSnapshot(ctx, c, t)
	if e != nil {
		return "", e
	}
	routes, e := nativePeerHostRoutes(ctx, t)
	if e != nil {
		return "", e
	}
	return nativePrepareStateDigest("scoped-peer-global-provenance-v2", r, pf, pr, nw, ledger, receipts, names, networks, routes), nil
}

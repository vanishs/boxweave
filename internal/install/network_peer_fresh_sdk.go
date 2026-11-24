package install

import (
	"context"
	"errors"
	"os"
	"os/user"
	"reflect"
	"strings"

	incus "github.com/lxc/incus/v7/client"
)

func nativePeerFreshSDKSnapshot(ctx context.Context, root incus.InstanceServer, project, bridge string) (nativePeerFreshEvidence, error) {
	var zero nativePeerFreshEvidence
	if root == nil || !strings.HasPrefix(project, "user-") || bridge != "incusbr-"+strings.TrimPrefix(project, "user-") {
		return zero, errors.New("exact dedicated native project/bridge required")
	}
	root = root.UseProject("default")
	// Existing fresh inventory rejects guests, foreign projects, customized
	// bridge, foreign profile consumers, leases and forwards.
	scope, err := inspectFreshIPv4(ctx, root, project, bridge)
	if err != nil {
		return zero, err
	}
	if !freshIPv4Disjoint(scope.Prefix, scope.Blocked) {
		return zero, errors.New("fresh bridge overlaps another native network or host route")
	}
	names, err := root.GetNetworkACLNames()
	if err != nil {
		return zero, err
	}
	if err = nativePeerFreshACLInventory(names); err != nil {
		return zero, err
	}
	if err = nativePeerBridgeKernelNoLivePorts(ctx, bridge); err != nil {
		return zero, err
	}
	nw, tag, err := root.GetNetwork(bridge)
	if err != nil || nw == nil || tag != scope.ETag || !reflect.DeepEqual(nw.Writable(), scope.Before) {
		return zero, errors.New("fresh native bridge changed during admission")
	}
	pr, ptag, err := root.GetProject(project)
	if err != nil || pr == nil || ptag == "" || pr.Config["restricted"] != "true" ||
		pr.Config["features.networks"] != "false" || pr.Config["restricted.networks.access"] != bridge {
		return zero, errors.New("fresh native project evidence unavailable")
	}
	pf, ftag, err := root.UseProject(project).GetProfile("default")
	if err != nil || pf == nil || ftag == "" || len(pf.UsedBy) != 0 ||
		pf.Devices["eth0"]["network"] != bridge || nativePrepareProfileEligible(pf.Devices, NativePrepareTarget{Network: bridge}) != nil {
		return zero, errors.New("fresh default profile evidence unavailable or modified")
	}
	// Label only, no shared ACL is created or adopted.
	guard, err := nativePeerFreshInitialGuard(NativePrepareTarget{Project: project, Network: bridge, ACL: "bw-fresh-initial-guard"}, nw)
	if err != nil {
		return zero, err
	}
	return nativePeerFreshEvidence{Project: project, Bridge: bridge, Guard: guard,
		NativeSHA256: nativePrepareStateDigest(nativePeerFreshMode, scope.Before, scope.ETag, scope.Blocked, pr, ptag, pf, ftag, names)}, nil
}

// No installed caller: full IPv6/global nft provenance, guarded setup
// locking, durable mode metadata and boot ordering still required.
func nativePeerFreshCollectReadOnly(ctx context.Context, upgrade bool) (nativePeerFreshEvidence, error) {
	var zero nativePeerFreshEvidence
	if upgrade {
		return zero, errors.New("scoped Peer v2 requires fresh installation; existing upgrade unsupported")
	}
	if os.Geteuid() != 0 || ctx == nil {
		return zero, errors.New("fresh scoped admission requires local root")
	}
	return nativePeerFreshAdmission(ctx, false, os.Lstat, func(ctx context.Context) (nativePeerFreshEvidence, error) {
		u, err := user.Lookup("boxweave")
		if err != nil || u.Uid == "0" {
			return zero, errors.New("dedicated service identity missing")
		}
		root, err := prepareClient(ctx)
		if err != nil {
			return zero, err
		}
		return nativePeerFreshSDKSnapshot(ctx, root, "user-"+u.Uid, "incusbr-"+u.Uid)
	})
}

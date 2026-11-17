package install

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	incus "github.com/lxc/incus/v7/client"
)

// Executes only on a brand-new disposable Incus 7.6 CI daemon. This tests
// the actual product root SDK Server PUT and exact global-config rollback;
// it does not authorize mutating any shared or installed L0 Incus daemon.
func TestS4B2ScriptletProductRootNativeApplyRollback(t *testing.T) {
	if os.Getenv("BOXWEAVE_S4B2_SCRIPTLET_ROOT_NATIVE") != "disposable-7.6" {
		t.Skip("disposable root-only Incus 7.6 integration")
	}
	if os.Geteuid() != 0 || os.Getenv("GITHUB_ACTIONS") != "true" ||
		!strings.HasPrefix(os.Getenv("RUNNER_TEMP"), "/") {
		t.Fatal("unsafe native root CI context")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	root, err := incus.ConnectIncusUnixWithContext(ctx, "/var/lib/incus/unix.socket", nil)
	if err != nil {
		t.Fatal(err)
	}
	host, err := migrationHostDigest()
	if err != nil {
		t.Fatal(err)
	}
	target := NativeScriptletTarget{OwnerProject: "bwplan", ACLProject: "default", ACL: "bw-s4b2-scriptlet",
		WriterFingerprint: os.Getenv("BOXWEAVE_S4B2_SCRIPTLET_FPR")}
	before, _, err := root.GetServer()
	if err != nil {
		t.Fatal(err)
	}
	original := nativeScriptletCopyConfig(before.Config)
	plan, err := nativeScriptletInspectWithClient(ctx, root, target, host)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("S4_B2_SCRIPTLET_NATIVE_PRODUCT_PLAN_SHA=%s, restricted_cert_count=%d",
		plan.ReviewSHA256, plan.RestrictedClientCount)
	path := filepath.Join(os.Getenv("RUNNER_TEMP"), "s4b2-product-scriptlet-intent.json")
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Fatal("preexisting native Scriptlet journal", err)
	}
	result, err := nativeScriptletApplyWithClient(ctx, root, target, host, path, plan.ReviewSHA256, true)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "ROUTE_APPLIED_PARITY_PENDING" || result.ReadyForPeer {
		t.Fatalf("product Scriptlet routed prematurely ready: %+v", result)
	}
	live, _, err := root.GetServer()
	if err != nil {
		t.Fatal(err)
	}
	source, err := NativeScriptletPolicy(target)
	if err != nil {
		t.Fatal(err)
	}
	if live.Config["authorization.client.tls-restricted"] != "scriptlet" ||
		live.Config["authorization.scriptlet"] != source {
		t.Fatal("Go SDK native Server config postread disagrees with product policy")
	}
	t.Log("S4_B2_PRODUCT_ROOT_NATIVE_SCRIPTLET_ACTUAL_UPDATE_AND_EXACT_READBACK=PASS")
	rolled, err := nativeScriptletRollbackWithClient(ctx, root, path, plan.ReviewSHA256, true)
	if err != nil {
		t.Fatal(err)
	}
	restored, _, err := root.GetServer()
	if err != nil {
		t.Fatal(err)
	}
	if rolled.Status != "ROLLED_BACK" || !reflect.DeepEqual(restored.Config, original) {
		t.Fatal("native global Scriptlet original config failed rollback")
	}
	t.Log("S4_B2_PRODUCT_ROOT_NATIVE_ORIGINAL_GLOBAL_ROUTE_EXACT_ROLLBACK=PASS")
}

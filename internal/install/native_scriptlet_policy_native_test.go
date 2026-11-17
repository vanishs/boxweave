package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Disposable CI-only bridge from this exact Go compiler into real Incus
// Scriptlet TLS permission tests. Never runs during product startup.
func TestS4B2ScriptletNativePolicySource(t *testing.T) {
	path := os.Getenv("BOXWEAVE_S4B2_SCRIPTLET_POLICY_OUTPUT")
	if path == "" {
		t.Skip("not running isolated native permission workflow")
	}
	if os.Getenv("GITHUB_ACTIONS") != "true" || filepath.Dir(path) != os.Getenv("RUNNER_TEMP") {
		t.Fatal("generated policy output requires disposable GitHub native workflow")
	}
	fingerprint := os.Getenv("BOXWEAVE_S4B2_SCRIPTLET_FPR")
	target := NativeScriptletTarget{OwnerProject: "bwlegacy", ACLProject: "default",
		ACL: "bw-s4b2-scriptlet", WriterFingerprint: fingerprint}
	policy, err := NativeScriptletPolicy(target)
	if err != nil {
		t.Fatal(err)
	}
	if len(policy) < 100 || !strings.Contains(policy, "network_acl:default/bw-s4b2-scriptlet") {
		t.Fatal("compiled native ACL source incomplete")
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.WriteString(policy); err != nil {
		f.Close()
		t.Fatal(err)
	}
	if err = f.Sync(); err != nil {
		f.Close()
		t.Fatal(err)
	}
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}
	t.Logf("S4_B2_PRODUCT_GENERATED_SCRIPTLET_SOURCE=PASS digest=%s", nativeScriptletSourceDigest(policy))
}

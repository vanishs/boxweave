package install

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// NativeScriptletTarget identifies one exclusive ACL and dedicated restricted
// TLS certificate. This target is host-local, never an MCP argument.
type NativeScriptletTarget struct {
	OwnerProject      string `json:"owner_project"`
	ACLProject        string `json:"acl_project"`
	ACL               string `json:"acl"`
	WriterFingerprint string `json:"writer_fingerprint"`
}

var scriptletFingerprint = regexp.MustCompile(`^[a-f0-9]{64}$`)

// NativeScriptletPolicy generates deterministic source for an installer audit.
// The current verified native parity is scoped, not all possible legacy
// entitlements; no production routing occurs from this function.
func NativeScriptletPolicy(t NativeScriptletTarget) (string, error) {
	if !safePrereqName.MatchString(t.OwnerProject) || t.OwnerProject == "default" ||
		t.ACLProject != "default" || !safePrereqName.MatchString(t.ACL) ||
		!strings.HasPrefix(t.ACL, "bw-") || !scriptletFingerprint.MatchString(t.WriterFingerprint) {
		return "", errors.New("Scriptlet target requires owned Project, default ACL and exact restricted TLS fingerprint")
	}
	source := fmt.Sprintf("def authorize(details, object, entitlement):\n    if details.Protocol != \"tls\":\n        return False\n    if details.Certificate == None or details.Certificate.restricted != True:\n        return False\n    if details.IsAllProjectsRequest:\n        return False\n    if details.Username == \"%s\":\n        if details.Certificate.projects != [\"default\"]:\n            return False\n        return object == \"network_acl:default/%s\" and details.ProjectName == \"default\" and details.Path == \"/1.0/network-acls/%s\" and details.Method in [\"GET\", \"PUT\"] and entitlement in [\"can_view\", \"can_edit\"]\n    parts = object.split(\":\")\n    if len(parts) != 2:\n        return False\n    obj_type = parts[0]\n    scope = parts[1].split(\"/\")[0]\n    if obj_type == \"server\":\n        return entitlement in [\"can_view\", \"can_view_resources\", \"can_view_metrics\"]\n    if obj_type in [\"storage_pool\", \"certificate\"]:\n        return entitlement == \"can_view\"\n    if obj_type == \"project\" and entitlement == \"can_edit\":\n        return False\n    allowed_types = [\"project\", \"image\", \"image_alias\", \"instance\", \"network\", \"network_acl\", \"network_address_set\", \"network_zone\", \"profile\", \"storage_bucket\", \"storage_volume\"]\n    if obj_type in allowed_types and scope in details.Certificate.projects:\n        return True\n    inherited_types = [\"image\", \"profile\", \"storage_volume\", \"storage_bucket\", \"network\", \"network_zone\"]\n    if obj_type in inherited_types and scope == \"default\" and entitlement == \"can_view\" and details.ProjectName in details.Certificate.projects:\n        return True\n    return False\n", t.WriterFingerprint, t.ACL, t.ACL)
	return source, nil
}

func nativeScriptletSourceDigest(source string) string {
	sum := sha256.Sum256([]byte("s4b2-native-scriptlet-source-v1\n" + source))
	return hex.EncodeToString(sum[:])
}

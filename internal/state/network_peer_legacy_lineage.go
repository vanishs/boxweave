package state

import (
	"bytes"
	"encoding/json"
	"strings"
)

// Compatibility-only immutable history decoder. Do not infer a migration
// or native permission from a historical shared-ACL original.
func legacyPeerHistoricalClaims(raw []byte, project, bridge string, p PeerGrantBinding, resource string, allow bool) []string {
	var expectedClaims []string
	var old RootPeerRevokeSafeIntent
	if json.Unmarshal(raw, &old) == nil {
		canonical, e := json.Marshal(old)
		if e == nil && bytes.Equal(canonical, raw) &&
			strings.HasPrefix(old.ACL, "bw-") && auditName(old.ACL) &&
			old.Project == project && old.Bridge == bridge && old.Allow == allow &&
			old.SourceUUID == p.SourceUUID && old.TargetUUID == p.TargetUUID &&
			old.SourceIncarnation == p.SourceIncarnation &&
			old.TargetIncarnation == p.TargetIncarnation &&
			resource == "acl:default/"+old.ACL {
			expectedClaims = []string{"acl:default/" + old.ACL,
				"box:" + p.SourceUUID, "box:" + p.TargetUUID}
		}
	}
	return expectedClaims
}

func legacyPeerHistoricalEvents(allow bool) (string, string) {
	var event string
	if allow {
		event = "PEER_ROOT_SCOPED_GRANT_KERNEL_VERIFIED"
	} else {
		event = "PEER_ROOT_SCOPED_REVOCATION_VERIFIED"
	}
	alt := "PEER_ROOT_COMPLETED_UNKNOWN_GRANT_READONLY_NO_REPLAY"
	if !allow {
		alt = "ROOT_COMPLETED_UNKNOWN_RECONCILED_NO_REPLAY"
	}
	return event, alt
}

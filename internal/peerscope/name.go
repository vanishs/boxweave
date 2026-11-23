// Package peerscope holds the single deterministic ACL name derivation
// shared by state and native policy code. It does not grant any native rights.
package peerscope

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"regexp"
)

var safeResource = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,62}$`)
var safeUUID = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// ACLName computes a stable per-Box ACL but NEVER proves its ownership,
// UsedBy, true native incarnation or local kernel enforcement.
func ACLName(project, bridge, uuid string) (string, error) {
	if !safeResource.MatchString(project) || !safeResource.MatchString(bridge) ||
		!safeUUID.MatchString(uuid) || project == "default" {
		return "", errors.New("invalid native per-Box ACL scope")
	}
	digest := sha256.Sum256([]byte(project + "\x00" + bridge + "\x00" + uuid))
	return "bw-peer-" + hex.EncodeToString(digest[:24]), nil
}

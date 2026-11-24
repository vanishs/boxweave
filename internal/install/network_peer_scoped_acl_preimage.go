package install

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/lxc/incus/v7/shared/api"
)

// NativePeerScopedACLPreimageWitness describes ONE exact read-only native
// all-Project ACL baseline. No evidence of nft/conntrack or a native write.
type NativePeerScopedACLPreimageWitness struct {
	Project, Bridge string
	ACLCount        int
	SHA256          string
}
type nativePeerScopedACLCanonicalEntry struct {
	UUID, ACL, ETag, UsedBy, BootAutostart, Description string
	Config                                              api.ConfigMap
	Ingress, Egress                                     []string
}

func nativePeerScopedCanonicalRules(rules []api.NetworkACLRule) ([]string, error) {
	out := make([]string, 0, len(rules))
	seen := map[string]bool{}
	for _, r := range rules {
		data, err := json.Marshal(r)
		if err != nil {
			return nil, err
		}
		key := string(data)
		if seen[key] {
			return nil, errors.New("duplicate exact native ACL rule")
		}
		seen[key] = true
		out = append(out, key)
	}
	slices.Sort(out)
	return out, nil
}

// NativePeerScopedVerifyObservedACLPreimage checks ALL independently read
// tenant ACL rules against an already compiled v2 A/B/C/D policy. It
// rejects a foreign ACCEPT, missing DROP or any unexpected rule/config/
// description. Native Incus ACL rule order is independent, so its readback
// is compared as a strict unordered set of full serialized rule objects.
// The result includes all exact ETags, boot choices and rule content.
//
// IMPORTANT: this is an additional read-only Root stage only. A future Root
// executor must independently recapture and compare the same complete
// original+Incus+sysfs+nft snapshot under locks immediately before effects;
// calling this function from MCP or using its SHA alone grants NO rights.
func NativePeerScopedVerifyObservedACLPreimage(plan NativePeerScopedRootTransition,
	observed NativePeerScopedRootEvidence) (NativePeerScopedACLPreimageWitness, error) {
	var zero NativePeerScopedACLPreimageWitness
	if plan.OriginalID == "" || len(plan.ExpectedBeforeByUUID) < 2 ||
		len(plan.ExpectedAfterByUUID) != len(plan.ExpectedBeforeByUUID) ||
		len(observed.Observed) != len(plan.ExpectedBeforeByUUID) ||
		len(observed.AllProjectNames) != len(plan.ExpectedBeforeByUUID) ||
		len(plan.OriginalBootAutostart) != len(plan.ExpectedBeforeByUUID) ||
		plan.UnrelatedACLsVerifiedUnchanged != len(plan.ExpectedBeforeByUUID)-2 ||
		!strings.HasPrefix(plan.Project, "user-") {
		return zero, errors.New("Root native ACL preimage lacks complete original scope")
	}
	seenIDs, seenNames := map[string]bool{}, map[string]bool{}
	entries := make([]nativePeerScopedACLCanonicalEntry, 0, len(observed.Observed))
	for name, owner := range observed.Observed {
		if name == "" || seenNames[name] || owner.Instance == nil || owner.ACL == nil ||
			owner.ACLTag == "" || owner.Instance.Name != name ||
			owner.Instance.Project != plan.Project || owner.ACL.Project != "default" ||
			len(owner.ACL.UsedBy) != 1 ||
			!nativePeerGuardOwnedURL(owner.ACL.UsedBy[0], "instances", name, plan.Project) {
			return zero, errors.New("native ACL original owner/Project/ETag evidence missing")
		}
		uuid := owner.Instance.Config["volatile.uuid"]
		expected, ok := plan.ExpectedBeforeByUUID[uuid]
		if !ok || seenIDs[uuid] ||
			owner.Instance.Config["boot.autostart"] != plan.OriginalBootAutostart[uuid] {
			return zero, errors.New("native original UUID or explicit boot preference changed")
		}
		expectedName, e := NativePeerScopedACLName(plan.Project, plan.Bridge, uuid)
		if e != nil || owner.ACL.Name != expectedName ||
			expected.Name != expectedName || !maps.Equal(owner.ACL.Config, expected.Config) ||
			owner.ACL.Description != expected.Description {
			return zero, errors.New("native ACL name, description or policy config differs from Root plan")
		}
		expectedIngress, e1 := nativePeerScopedCanonicalRules(expected.Ingress)
		expectedEgress, e2 := nativePeerScopedCanonicalRules(expected.Egress)
		actualIngress, e3 := nativePeerScopedCanonicalRules(owner.ACL.Ingress)
		actualEgress, e4 := nativePeerScopedCanonicalRules(owner.ACL.Egress)
		if e1 != nil || e2 != nil || e3 != nil || e4 != nil ||
			!slices.Equal(expectedIngress, actualIngress) ||
			!slices.Equal(expectedEgress, actualEgress) {
			return zero, errors.New("native ACL before-image has missing, extra or changed rules")
		}
		seenIDs[uuid], seenNames[name] = true, true
		entries = append(entries, nativePeerScopedACLCanonicalEntry{
			UUID: uuid, ACL: expectedName, ETag: owner.ACLTag,
			UsedBy:        owner.ACL.UsedBy[0],
			BootAutostart: plan.OriginalBootAutostart[uuid],
			Description:   expected.Description, Config: expected.Config,
			Ingress: expectedIngress, Egress: expectedEgress,
		})
	}
	for _, name := range observed.AllProjectNames {
		if !seenNames[name] {
			return zero, errors.New("native root Project instance list changed")
		}
	}
	// The bridge may only be used by the exact tenant default profile and
	// independently observed exact tenant instances; foreign/root profile
	// consumers must not appear when verifying an exclusive bridge.
	used := map[string]bool{}
	for _, url := range observed.RootNetworkUsedBy {
		if used[url] {
			return zero, errors.New("duplicate bridge UserBy entry")
		}
		used[url] = true
		if nativePeerGuardOwnedURL(url, "profiles", "default", plan.Project) {
			continue
		}
		matched := false
		for name := range seenNames {
			if nativePeerGuardOwnedURL(url, "instances", name, plan.Project) {
				matched = true
				break
			}
		}
		if !matched {
			return zero, fmt.Errorf("foreign root bridge consumer in native ACL witness: %q", url)
		}
	}
	if len(used) == 0 {
		return zero, errors.New("native root bridge consumers unavailable")
	}
	slices.SortFunc(entries, func(a, b nativePeerScopedACLCanonicalEntry) int {
		return strings.Compare(a.UUID, b.UUID)
	})
	raw, e := json.Marshal(entries)
	if e != nil {
		return zero, e
	}
	digest := sha256.Sum256(raw)
	return NativePeerScopedACLPreimageWitness{
		Project: plan.Project, Bridge: plan.Bridge, ACLCount: len(entries),
		SHA256: hex.EncodeToString(digest[:]),
	}, nil
}

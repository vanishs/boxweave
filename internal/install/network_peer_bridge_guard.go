package install

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// nativePeerBridgeConsumersOwned is a necessary (not sufficient) trust
// boundary for a dedicated, root-observed managed bridge. It rejects foreign
// Project, unknown same-Project instances, profiles and raw-parent uses shown
// in root GetNetwork.UsedBy. Never trust restricted-Project masked GETs.
//
// The controller MUST ALSO verify each returned native UUID, expanded NIC,
// per-NIC anti-spoof, static lease and bridge kernel port membership at grant
// admission and before release. A privileged host administrator is trusted
// above this boundary and can change native topology out of band.
func nativePeerBridgeConsumersOwned(usedBy []string, project string, ownedNativeNames map[string]string) error {
	if !safePrereqName.MatchString(project) || project == "default" || len(usedBy) == 0 {
		return errors.New("peer bridge owner or native references unavailable")
	}
	seen := make(map[string]bool, len(usedBy))
	foundDefault := false
	for _, ref := range usedBy {
		u, err := url.Parse(ref)
		if err != nil || u.Scheme != "" || u.Host != "" || u.User != nil ||
			u.Fragment != "" || u.RawFragment != "" || u.Opaque != "" ||
			!strings.HasPrefix(u.Path, "/1.0/") {
			return fmt.Errorf("untrusted peer bridge reference")
		}
		q := u.Query()
		if len(q) != 1 || len(q["project"]) != 1 || q.Get("project") != project ||
			u.RawQuery != "project="+url.QueryEscape(project) || seen[ref] {
			return fmt.Errorf("foreign, duplicated or ambiguous peer bridge consumer")
		}
		seen[ref] = true
		switch {
		case u.Path == "/1.0/profiles/default":
			foundDefault = true
		case strings.HasPrefix(u.Path, "/1.0/instances/"):
			name := strings.TrimPrefix(u.Path, "/1.0/instances/")
			if !safePrereqName.MatchString(name) || ownedNativeNames[name] == "" {
				return fmt.Errorf("unknown or foreign instance on peer bridge")
			}
		default:
			return fmt.Errorf("non-BoxWeave NIC/profile or raw bridge consumer")
		}
	}
	if !foundDefault {
		return errors.New("peer bridge default Profile ownership not visible")
	}
	return nil
}

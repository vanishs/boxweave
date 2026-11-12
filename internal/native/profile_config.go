package native

import (
	"fmt"
	"strconv"
	"strings"
)

// SafePerUserProfileConfig permits either the empty default Incus Profile or
// the exact service-user mapping created by Incus v7.6 cmd/incus-user/server.go.
// Other raw.idmap maps, raw.lxc and arbitrary inherited Profile config are not
// accepted before native Box admission.
func SafePerUserProfileConfig(config map[string]string, uid, gid string) bool {
	u, errU := strconv.Atoi(uid)
	g, errG := strconv.Atoi(gid)
	if errU != nil || errG != nil || u < 1 || g < 1 {
		return false
	}
	if len(config) == 0 {
		return true
	}
	if len(config) != 1 {
		return false
	}
	expected := fmt.Sprintf("uid %d %d\ngid %d %d", u, u, g, g)
	return strings.TrimSpace(config["raw.idmap"]) == expected
}

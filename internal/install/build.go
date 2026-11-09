package install

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

type BuildIdentity struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
}

var exactBuildCommit = regexp.MustCompile(`^[0-9a-f]{40}$`)

// The candidate executable itself declares its build identity. Reading the
// version of the OLD invoking CLI is not evidence about the NEW executable.
func verifiedSourceBuild(p Plan, binary string) (BuildIdentity, error) {
	if _, _, err := ServiceIdentity(p); err != nil {
		return BuildIdentity{}, err
	}
	out, err := exec.Command("runuser", "-u", p.User, "--", binary, "build-info").Output()
	if err != nil {
		return BuildIdentity{}, fmt.Errorf("target binary identity unavailable to restricted actor: %w", err)
	}
	var build BuildIdentity
	if err = json.Unmarshal(out, &build); err != nil {
		return BuildIdentity{}, errors.New("invalid target binary build identity JSON")
	}
	if !safeValue.MatchString(build.Version) || strings.TrimSpace(build.Version) == "" || !exactBuildCommit.MatchString(build.Commit) {
		return BuildIdentity{}, errors.New("unverified version or source commit in candidate binary")
	}
	return build, nil
}

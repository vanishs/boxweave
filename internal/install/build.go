package install

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
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
	uid, _, err := ServiceIdentity(p)
	if err != nil {
		return BuildIdentity{}, err
	}
	// Root runs the candidate under the restricted identity. An already
	// restricted recovery worker executes directly (runuser requires root).
	var cmd *exec.Cmd
	switch os.Geteuid() {
	case 0:
		cmd = exec.Command("runuser", "-u", p.User, "--", binary, "build-info")
	case uid:
		cmd = exec.Command(binary, "build-info")
	default:
		return BuildIdentity{}, errors.New("wrong OS identity for executable verification")
	}
	out, err := cmd.Output()
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

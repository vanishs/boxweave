package install

import (
	"errors"
	"regexp"
)

const nativePeerRootUnitPath = "/etc/systemd/system/boxweave-native-peer-root.service"
const nativeRetirementUnitPath = "/etc/systemd/system/boxweave-ipv4-retirement.service"
const nativeRetirementUnitName = "boxweave-ipv4-retirement.service"
const nativePeerRootInstalledBinary = "/usr/local/bin/boxweave"

var nativePeerRootUnitToken = regexp.MustCompile("^[a-zA-Z0-9_-]+$")

// NativePeerRootScope describes a historical unit for readback only.
type NativePeerRootScope struct {
	DBPath, Project, Bridge, ACL string
	OriginalID, ApprovedPlanSHA  string
	Approved                     bool
	RetirementReview             bool
	RetirementCompact            bool
}

func retirementOnlyScope(cfg NativePeerRootScope) bool {
	return cfg.RetirementReview && cfg.OriginalID == "" && cfg.ApprovedPlanSHA == ""
}

// The original exact peer grant/revoke approval remains mandatory for any
// ordinary Peer grant/revoke request. A retirement scope independently
// verifies exact deleted originals of its fixed Project; only explicit
// RetirementCompact approval permits the journaled retired-DROP compaction.
func nativePeerRootScopeApproved(cfg NativePeerRootScope) bool {
	if !cfg.Approved || (cfg.RetirementCompact && !cfg.RetirementReview) {
		return false
	}
	if cfg.OriginalID == "" && cfg.ApprovedPlanSHA == "" {
		return cfg.RetirementReview
	}
	return cfg.OriginalID != "" && cfg.ApprovedPlanSHA != "" && scopedLifecycleID.MatchString(cfg.OriginalID) && validRetirementReviewSHA(cfg.ApprovedPlanSHA)
}

// NativePeerRootUnitText reconstructs historical unit bytes for observation.
// No current installer can stage or execute this retired unit.
func NativePeerRootUnitText(cfg NativePeerRootScope) (string, error) {
	if !cfg.Approved || cfg.DBPath != nativePeerRootInstalledDB ||
		cfg.Project == "" || cfg.Bridge == "" || cfg.ACL == "" ||
		!nativePeerRootUnitToken.MatchString(cfg.Project) ||
		!nativePeerRootUnitToken.MatchString(cfg.Bridge) ||
		!nativePeerRootUnitToken.MatchString(cfg.ACL) ||
		!nativePeerRootScopeApproved(cfg) {
		return "", errors.New("invalid exact approved native root service scope")
	}
	// The systemd service runs only a pinned command, never root-exposed
	// Config, ExecStartPre, shell, environment, and never an unaudited update.
	command := nativePeerRootInstalledBinary + " native-peer-root-serve -db " + cfg.DBPath +
		" -project " + cfg.Project + " -network " + cfg.Bridge + " -acl " + cfg.ACL +
		""
	if cfg.OriginalID != "" {
		command += " -operation-id " + cfg.OriginalID + " -peer-plan-sha256 " + cfg.ApprovedPlanSHA
	}
	if cfg.RetirementReview {
		command += " -retirement-review"
	}
	if cfg.RetirementCompact {
		command += " -retirement-compact"
	}
	command += " -approve-local"
	header := "[Unit]\nDescription=BoxWeave native peer root single-original approval\nAfter=incus.service\nRequires=incus.service\n\n"
	if retirementOnlyScope(cfg) {
		header = "[Unit]\nDescription=BoxWeave exact retired IPv4 verifier\nAfter=incus.service boxweave.service\nRequires=incus.service\nPartOf=boxweave.service\nBindsTo=boxweave.service\n\n"
	}
	return header +
		"[Service]\nType=simple\nUser=root\nGroup=root\nUMask=0077\nExecStart=" + command +
		"\nRestart=on-failure\nRestartSec=1s\nNoNewPrivileges=true\nPrivateTmp=true\nProtectHome=true\nProtectSystem=full" +
		"\nRestrictAddressFamilies=AF_UNIX AF_NETLINK\n\n[Install]\nWantedBy=multi-user.target\n", nil
}

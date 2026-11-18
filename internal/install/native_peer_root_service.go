package install

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	"github.com/vanishs/boxweave/internal/state"
)

const nativePeerRootUnitPath = "/etc/systemd/system/boxweave-native-peer-root.service"
const nativePeerRootInstalledBinary = "/usr/local/bin/boxweave"
const nativePeerRootInstalledDB = "/var/lib/boxweave/ledger.sqlite"

var nativePeerRootUnitToken = regexp.MustCompile("^[a-zA-Z0-9_-]+$")

// NativePeerRootUnitText creates a one-original, one-pre-mutation-SHA
// operator-approved unit. It is not a general root daemon, host shell or
// public MCP service. No untrusted unit, binary, path or environment is used.
func NativePeerRootUnitText(cfg NativePeerRootScope) (string, error) {
	if !cfg.Approved || cfg.DBPath != nativePeerRootInstalledDB ||
		cfg.Project == "" || cfg.Bridge == "" || cfg.ACL == "" ||
		!nativePeerRootUnitToken.MatchString(cfg.Project) ||
		!nativePeerRootUnitToken.MatchString(cfg.Bridge) ||
		!nativePeerRootUnitToken.MatchString(cfg.ACL) ||
		nativePeerRootValidateRequest(NativePeerRootRequest{
			OperationID: cfg.OriginalID, PlanSHA256: cfg.ApprovedPlanSHA}) != nil ||
		cfg.ApprovedPlanSHA == "" {
		return "", errors.New("invalid exact approved native root service scope")
	}
	// The systemd service runs only a pinned command, never root-exposed
	// Config, ExecStartPre, shell, environment, and never an unaudited update.
	command := nativePeerRootInstalledBinary + " native-peer-root-serve -db " + cfg.DBPath +
		" -project " + cfg.Project + " -network " + cfg.Bridge + " -acl " + cfg.ACL +
		" -operation-id " + cfg.OriginalID + " -peer-plan-sha256 " + cfg.ApprovedPlanSHA + " -approve-local"
	return "[Unit]\nDescription=BoxWeave native peer root single-original approval\nAfter=incus.service\nRequires=incus.service\n\n" +
		"[Service]\nType=simple\nUser=root\nGroup=root\nUMask=0077\nExecStart=" + command +
		"\nRestart=on-failure\nRestartSec=1s\nNoNewPrivileges=true\nPrivateTmp=true\nProtectHome=true\nProtectSystem=full" +
		"\nRestrictAddressFamilies=AF_UNIX AF_NETLINK\n\n[Install]\nWantedBy=multi-user.target\n", nil
}

func nativePeerRootInstalledBinarySafe() error {
	st, err := os.Lstat(nativePeerRootInstalledBinary)
	if err != nil {
		return err
	}
	owner, ok := st.Sys().(*syscall.Stat_t)
	if !ok || !st.Mode().IsRegular() || st.Mode().Perm()&0022 != 0 ||
		owner.Uid != 0 {
		return errors.New("installed root helper executable is not trusted root-owned regular file")
	}
	return nil
}

// NativePeerRootUnitStage persists the approved helper's systemd *unit*,
// but deliberately does NOT enable or start it. The operator must inspect
// the file and explicitly systemctl daemon-reload + enable --now. This is a
// host-only root maintenance action, not a publicly exposed network Tool.
// An existing unit, even stale/corrupt, is never overwritten or adopted.
func NativePeerRootUnitStage(ctx context.Context, cfg NativePeerRootScope) error {
	if os.Geteuid() != 0 {
		return errors.New("host root required for root service stage")
	}
	u, err := user.Lookup("boxweave")
	if err != nil {
		return err
	}
	if cfg.Project != "user-"+u.Uid {
		return errors.New("root service Project differs from installed restricted UID")
	}
	if n, err := strconv.Atoi(u.Uid); err != nil || n == 0 {
		return errors.New("unsafe root helper UID")
	}
	if err := nativePeerRootInstalledBinarySafe(); err != nil {
		return err
	}
	body, err := NativePeerRootUnitText(cfg)
	if err != nil {
		return err
	}
	db, err := state.OpenNetworkAuditReadOnly(cfg.DBPath)
	if err != nil {
		return err
	}
	defer db.Close()
	plan, err := NativePeerRootIntegratedReviewOperation(ctx, db, cfg.Project, cfg.Bridge, cfg.ACL, cfg.OriginalID)
	if err != nil {
		return fmt.Errorf("native root service preflight refused: %w", err)
	}
	if plan.PlanSHA256 != cfg.ApprovedPlanSHA {
		return errors.New("root service operator-approved pre-mutation SHA drifted")
	}
	// First stage is exclusive and durable; neither an existing unit nor
	// the root intent journal is reset by staging/restarting/upgrading.
	fd, err := os.OpenFile(nativePeerRootUnitPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("root unit exists or unsafe: %w", err)
	}
	_, err = fd.WriteString(body)
	if err == nil {
		err = fd.Sync()
	}
	closeErr := fd.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	d, err := os.Open(filepath.Dir(nativePeerRootUnitPath))
	if err != nil {
		return err
	}
	defer d.Close()
	if err := d.Sync(); err != nil {
		return err
	}
	written, err := os.ReadFile(nativePeerRootUnitPath)
	if err != nil || strings.TrimSpace(string(written)) != strings.TrimSpace(body) {
		return errors.New("root service persisted systemd unit readback failed")
	}
	return nil
}

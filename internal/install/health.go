package install

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// Distinguish an active systemd state from a *stable and actually responding*
// instance of the exact reviewed executable. Do not accept a transient PID or
// a foreign process already bound to the loopback MCP port.
func observeInstalled(p Plan, m Manifest) error {
	if err := serviceActive(); err != nil {
		return fmt.Errorf("systemd unit inactive: %w", err)
	}
	out, err := exec.Command("systemctl", "show", "--property=MainPID", "--value", serviceName).Output()
	if err != nil {
		return err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil || pid < 2 {
		return errors.New("no stable service MainPID")
	}
	stat, err := os.Stat(filepath.Join("/proc", strconv.Itoa(pid)))
	if err != nil {
		return err
	}
	info, ok := stat.Sys().(*syscall.Stat_t)
	if !ok {
		return errors.New("process identity unavailable")
	}
	uid, _, err := ServiceIdentity(p)
	if err != nil {
		return err
	}
	if int(info.Uid) != uid {
		return errors.New("service MainPID belongs to wrong OS user")
	}
	h, err := sha(filepath.Join("/proc", strconv.Itoa(pid), "exe"))
	if err != nil {
		return err
	}
	if h != m.BinarySHA256 {
		return errors.New("running MainPID executable digest mismatch")
	}
	client := &http.Client{Timeout: time.Second, Transport: &http.Transport{Proxy: nil}}
	req, err := http.NewRequest(http.MethodPost, "http://127.0.0.1:18650/mcp", strings.NewReader("{}"))
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1024))
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		return fmt.Errorf("actual bound MCP unauthorized probe expected 401, got %d", resp.StatusCode)
	}
	if err := observeScopedInstalled(m); err != nil {
		return err
	}
	return observeInstalledRetirement(p, m)
}
func waitInstalledHealthy(p Plan, m Manifest) error {
	var last error
	streak := 0
	for i := 0; i < 20; i++ {
		last = observeInstalled(p, m)
		if last == nil {
			streak++
			if streak >= 3 {
				return nil
			}
		} else {
			streak = 0
		}
		time.Sleep(350 * time.Millisecond)
	}
	if last == nil {
		last = errors.New("no stable runtime confirmation")
	}
	return fmt.Errorf("installed runtime not continuously healthy: %w", last)
}

func observeInstalledRetirement(p Plan, m Manifest) error {
	if m.RetirementUnitSHA256 == "" {
		return nil
	}
	if e := verifyInstalledRetirementUnit(m); e != nil {
		return e
	}
	if e := runLocal("systemctl", "is-active", "--quiet", nativeRetirementUnitName); e != nil {
		return errors.New("installed retirement verifier not active")
	}
	props, e := retirementUnitProperties()
	if e != nil {
		return e
	}
	pid, e := strconv.Atoi(props["MainPID"])
	if e != nil || pid < 2 {
		return errors.New("retirement verifier PID unavailable")
	}
	st, e := os.Stat(filepath.Join("/proc", strconv.Itoa(pid)))
	if e != nil {
		return e
	}
	owner, ok := st.Sys().(*syscall.Stat_t)
	if !ok || owner.Uid != 0 {
		return errors.New("retirement verifier not root-owned process")
	}
	h, e := sha(filepath.Join("/proc", strconv.Itoa(pid), "exe"))
	if e != nil || h != m.BinarySHA256 {
		return errors.New("retirement verifier executable differs")
	}
	_, gid, e := ServiceIdentity(p)
	if e != nil {
		return e
	}
	sock, e := os.Lstat(nativeRetirementSocketPath)
	if e != nil {
		return e
	}
	so, ok := sock.Sys().(*syscall.Stat_t)
	if !ok || so.Uid != 0 || so.Gid != uint32(gid) || sock.Mode()&os.ModeSocket == 0 || sock.Mode().Perm() != 0660 {
		return errors.New("retirement verifier socket ownership differs")
	}
	conn, e := net.DialTimeout("unix", nativeRetirementSocketPath, time.Second)
	if e != nil {
		return e
	}
	defer conn.Close()
	uc, ok := conn.(*net.UnixConn)
	if !ok {
		return errors.New("unexpected retirement socket transport")
	}
	return nativePeerRootPeerIdentity(uc, 0, pid)
}

func retirementUnitProperties() (map[string]string, error) {
	raw, e := exec.Command("systemctl", "show", "-p", "FragmentPath", "-p", "DropInPaths", "-p", "MainPID", nativeRetirementUnitName).Output()
	if e != nil {
		return nil, e
	}
	props := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		k, v, ok := strings.Cut(line, "=")
		if ok {
			if _, dup := props[k]; dup {
				return nil, errors.New("ambiguous retirement service properties")
			}
			props[k] = v
		}
	}
	dp, ok := props["DropInPaths"]
	if !ok || dp != "" || props["FragmentPath"] != nativeRetirementUnitPath {
		return nil, errors.New("retirement verifier effective unit differs")
	}
	return props, nil
}

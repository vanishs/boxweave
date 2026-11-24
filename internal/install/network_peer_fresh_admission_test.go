package install

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vanishs/boxweave/internal/state"
)

func pr178FreshEvidence(t *testing.T) nativePeerFreshEvidence {
	t.Helper()
	target := pr178GuardTarget()
	guard, err := nativePeerFreshInitialGuard(target, s4b2UnitBridge())
	if err != nil {
		t.Fatal(err)
	}
	return nativePeerFreshEvidence{Project: target.Project, Bridge: target.Network, NativeSHA256: strings.Repeat("a", 64), Guard: guard, IPv4CIDR: "10.55.8.1/24", IPv6CIDR: "fd55:8::1/64"}
}

func TestPR178FreshInitialGuardBeforeAnyBox(t *testing.T) {
	p := pr178FreshEvidence(t).Guard
	var rules [][]any
	if err := json.Unmarshal([]byte(p.ExpectedKernelJSON), &rules); err != nil || len(rules) != 4 {
		t.Fatalf("missing exact default guard: %s %v", p.ExpectedKernelJSON, err)
	}
	if strings.Count(p.Install, "counter drop") != 3 || strings.Count(p.Replace, " accept") != 1 || strings.Contains(p.Install, "ct state") {
		t.Fatal("empty guard contains peer allow or missing denies")
	}
	target := pr178GuardTarget()
	if _, err := NativePeerGuardFromSnapshot(target, s4b2UnitBridge(), nil, nil, nil); err == nil {
		t.Fatal("normal guard accepted empty inventory")
	}
	pair := pr178Grant(pr178A, pr178B, "10.55.8.11", "10.55.8.12", "fd55:8::11", "fd55:8::12")
	if _, err := nativePeerGuardCompile(target, s4b2UnitBridge(), nil, []state.PeerGrantBinding{pair}, nil, true); err == nil {
		t.Fatal("fresh guard accepted historic grant")
	}
	if _, err := nativePeerGuardCompile(target, s4b2UnitBridge(), nil, nil, []state.PeerGrantBinding{pair}, true); err == nil {
		t.Fatal("fresh guard accepted historic tombstone")
	}
	if _, err := nativePeerGuardCompile(target, s4b2UnitBridge(), pr178GuardMembers(), nil, nil, true); err == nil {
		t.Fatal("fresh guard accepted existing owners")
	}
	if p != pr178Guard(t, pr178GuardMembers(), nil, nil) {
		t.Fatal("initial guard differs from no-grant guard")
	}
}

func TestPR178FreshAdmissionRefusesBeforeInspection(t *testing.T) {
	for _, tc := range []struct {
		name    string
		upgrade bool
		path    string
		err     error
	}{
		{"upgrade", true, "", nil}, {"state", false, "/var/lib/boxweave", nil},
		{"binary", false, "/usr/local/bin/boxweave", nil}, {"unit", false, "/etc/systemd/system/boxweave.service", nil},
		{"drop-in", false, "/etc/systemd/system/boxweave.service.d", nil}, {"journal", false, nativePrepareJournalPath, nil},
		{"root-journal", false, nativePeerGuardRootJournalDir, nil}, {"unreadable", false, "/var/lib/boxweave", fs.ErrPermission},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			got, err := nativePeerFreshAdmission(context.Background(), tc.upgrade, func(path string) (fs.FileInfo, error) {
				if path == tc.path {
					return nil, tc.err
				}
				return nil, fs.ErrNotExist
			}, func(context.Context) (nativePeerFreshEvidence, error) { calls++; return pr178FreshEvidence(t), nil })
			if err == nil || calls != 0 || got != (nativePeerFreshEvidence{}) {
				t.Fatal("old installation admitted", err, calls, got)
			}
		})
	}
}
func TestPR178FreshAdmissionTwoReadAndPathDrift(t *testing.T) {
	want := pr178FreshEvidence(t)
	for _, tc := range []string{"clean", "drift", "late-state", "native-error", "bad-digest", "cancel"} {
		t.Run(tc, func(t *testing.T) {
			ctx, stop := context.WithCancel(context.Background())
			defer stop()
			n := 0
			got, err := nativePeerFreshAdmission(ctx, false, func(path string) (fs.FileInfo, error) {
				if tc == "late-state" && n >= 2 && path == "/var/lib/boxweave" {
					return nil, nil
				}
				return nil, fs.ErrNotExist
			}, func(context.Context) (nativePeerFreshEvidence, error) {
				n++
				v := want
				if tc == "drift" && n == 2 {
					v.NativeSHA256 = strings.Repeat("b", 64)
				}
				if tc == "native-error" && n == 2 {
					return v, errors.New("SDK failure")
				}
				if tc == "bad-digest" {
					v.Guard.DigestSHA256 = "bad"
				}
				if tc == "cancel" && n == 2 {
					stop()
				}
				return v, nil
			})
			if tc == "clean" {
				if err != nil || n != 2 || got != want {
					t.Fatal(got, n, err)
				}
			} else if err == nil || got != (nativePeerFreshEvidence{}) {
				t.Fatal("uncertain fresh admission accepted", tc, got, err)
			}
		})
	}
}
func TestPR178FreshAdmissionDanglingSymlinkAndACL(t *testing.T) {
	dir := t.TempDir()
	link := filepath.Join(dir, "legacy")
	if err := os.Symlink(filepath.Join(dir, "missing"), link); err != nil {
		t.Fatal(err)
	}
	if err := nativePeerFreshPathsAbsent(func(path string) (fs.FileInfo, error) {
		if path == "/var/lib/boxweave" {
			return os.Lstat(link)
		}
		return nil, fs.ErrNotExist
	}); err == nil {
		t.Fatal("dangling old installation accepted")
	}
	for _, names := range [][]string{{"bw-user-1001-acl"}, {"operator", "bw-detached"}, {""}} {
		if nativePeerFreshACLInventory(names) == nil {
			t.Fatal("legacy ACL accepted", names)
		}
	}
	if err := nativePeerFreshACLInventory([]string{"operator"}); err != nil {
		t.Fatal(err)
	}
}

package install

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPR178FreshBootstrapIntentBeforeEffectAndNoReplay(t *testing.T) {
	for _, tc := range []string{"success", "existing-table", "preimage-drift", "late-state", "second-absence-fault", "applied-ACK-lost", "witness-fault", "postimage-drift", "post-witness-fault", "late-post-state", "receipt-collision"} {
		t.Run(tc, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "fresh")
			uid := uint32(os.Geteuid())
			evidence := pr178FreshEvidence(t)
			host := strings.Repeat("c", 64)
			reads, absence, writes, witnesses := 0, 0, 0, 0
			ctx := context.Background()
			lstat := func(path string) (fs.FileInfo, error) {
				if path == nativePeerFreshJournalDir {
					return os.Lstat(dir)
				}
				if path == "/var/lib/boxweave" && (tc == "late-state" && reads >= 3 || tc == "late-post-state" && writes > 0) {
					return nil, nil
				}
				return nil, fs.ErrNotExist
			}
			inspect := func(context.Context) (nativePeerFreshEvidence, error) {
				reads++
				got := evidence
				if tc == "preimage-drift" && reads == 3 || tc == "postimage-drift" && reads == 4 {
					got.NativeSHA256 = strings.Repeat("d", 64)
				}
				return got, nil
			}
			absent := func(context.Context, NativePeerGuardPlan) error {
				absence++
				if tc == "existing-table" || tc == "second-absence-fault" && absence == 2 {
					return errors.New("guard absence unverified")
				}
				return nil
			}
			install := func(_ context.Context, p NativePeerGuardPlan) error {
				writes++
				d, err := nativePeerFreshJournalOpenAt(dir, uid)
				if err != nil {
					t.Fatal("effect without durable journal", err)
				}
				defer d.Close()
				j, err := nativePeerFreshJournalRead(d, "DISPATCH_UNKNOWN")
				if err != nil || j.HostSHA256 != host || j.NativeSHA256 != evidence.NativeSHA256 || j.GuardSHA256 != p.DigestSHA256 {
					t.Fatal("effect without bound intent", j, err)
				}
				if _, err = os.Lstat(filepath.Join(dir, "guard-verified.json")); !errors.Is(err, fs.ErrNotExist) {
					t.Fatal("receipt before effect", err)
				}
				if tc == "receipt-collision" {
					if err = os.WriteFile(filepath.Join(dir, "guard-verified.json"), []byte("invalid"), 0600); err != nil {
						t.Fatal(err)
					}
				}
				if tc == "applied-ACK-lost" {
					return errors.New("ACK lost after write")
				}
				return nil
			}
			witness := func(_ context.Context, p NativePeerGuardPlan) error {
				witnesses++
				if p != evidence.Guard || tc == "witness-fault" || tc == "post-witness-fault" && witnesses == 2 {
					return errors.New("kernel readback fault")
				}
				return nil
			}
			got, err := nativePeerFreshBootstrapOnce(ctx, dir, uid, host, false, lstat, inspect, absent, install, witness)
			wantWrites := 1
			if tc == "existing-table" || tc == "preimage-drift" || tc == "late-state" || tc == "second-absence-fault" {
				wantWrites = 0
			}
			if writes != wantWrites {
				t.Fatal("wrong dispatch count", writes, err)
			}
			if tc == "success" {
				if err != nil || got.Status != "INITIAL_GUARD_VERIFIED" || witnesses != 2 || nativePeerFreshJournalVerifiedAt(dir, uid, got) != nil {
					t.Fatal("unverified initial guard", got, err)
				}
			} else if err == nil || got.Status == "INITIAL_GUARD_VERIFIED" {
				t.Fatal("uncertain setup reported verified", got, err)
			}
			if tc == "existing-table" {
				if _, e := os.Lstat(dir); !errors.Is(e, fs.ErrNotExist) {
					t.Fatal("journal created before absence proved", e)
				}
				return
			}
			if tc != "success" {
				want := got
				want.Status = "INITIAL_GUARD_VERIFIED"
				if nativePeerFreshJournalVerifiedAt(dir, uid, want) == nil {
					t.Fatal("uncertain receipt trusted")
				}
			}
			if _, err = nativePeerFreshBootstrapOnce(ctx, dir, uid, host, false, lstat, inspect, absent, install, witness); err == nil || writes != wantWrites {
				t.Fatal("existing original replayed", err, writes)
			}
		})
	}
}

func TestPR178FreshJournalRefusesTamperedReceipt(t *testing.T) {
	for _, tc := range []string{"foreign-host", "foreign-bridge", "symlink", "hardlink", "world-writable", "truncated", "noncanonical", "directory-mode", "wrong-owner"} {
		t.Run(tc, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "fresh")
			uid := uint32(os.Geteuid())
			e := pr178FreshEvidence(t)
			want := nativePeerFreshJournalRecord{Schema: 1, Mode: nativePeerFreshMode, HostSHA256: strings.Repeat("c", 64), Project: e.Project, Bridge: e.Bridge, NativeSHA256: e.NativeSHA256, GuardSHA256: e.Guard.DigestSHA256, Status: "INITIAL_GUARD_VERIFIED"}
			d, err := nativePeerFreshJournalCreateAt(dir, uid)
			if err != nil {
				t.Fatal(err)
			}
			intent := want
			intent.Status = "DISPATCH_UNKNOWN"
			if err = nativePeerFreshJournalWrite(d, intent); err != nil {
				t.Fatal(err)
			}
			receipt := want
			if tc == "foreign-host" {
				receipt.HostSHA256 = strings.Repeat("d", 64)
			}
			if tc == "foreign-bridge" {
				receipt.Bridge = "foreign-bridge"
			}
			if err = nativePeerFreshJournalWrite(d, receipt); err != nil {
				t.Fatal(err)
			}
			d.Close()
			path := filepath.Join(dir, "guard-verified.json")
			switch tc {
			case "symlink":
				if err = os.Rename(path, path+".old"); err == nil {
					err = os.Symlink(path+".old", path)
				}
			case "hardlink":
				err = os.Link(path, path+".other")
			case "world-writable":
				err = os.Chmod(path, 0666)
			case "truncated":
				err = os.WriteFile(path, []byte("{"), 0600)
			case "noncanonical":
				var data []byte
				data, err = os.ReadFile(path)
				if err == nil {
					err = os.WriteFile(path, append(data, '\n'), 0600)
				}
			case "directory-mode":
				err = os.Chmod(dir, 0777)
			case "wrong-owner":
				uid++
			}
			if err != nil {
				t.Fatal(err)
			}
			if nativePeerFreshJournalVerifiedAt(dir, uid, want) == nil {
				t.Fatal("tampered journal accepted", tc)
			}
		})
	}
}

func TestPR178FreshEvidenceRecompilesExactGuard(t *testing.T) {
	for _, tc := range []string{"wrong-prefix", "wrong-project", "forged-script", "forged-json", "wrong-table"} {
		t.Run(tc, func(t *testing.T) {
			e := pr178FreshEvidence(t)
			switch tc {
			case "wrong-prefix":
				e.IPv4CIDR = "10.77.0.1/24"
			case "wrong-project":
				e.Project = "user-77"
			case "forged-script":
				e.Guard.Install += "\nflush ruleset\n"
			case "forged-json":
				e.Guard.ExpectedKernelJSON = "[]"
			case "wrong-table":
				e.Guard.Table = "bw_pg_0123456789abcdef"
			}
			e.Guard.DigestSHA256 = nativePeerGuardDigest(e.Guard)
			if nativePeerFreshEvidenceValid(e) == nil {
				t.Fatal("forged fresh guard accepted", tc)
			}
		})
	}
}

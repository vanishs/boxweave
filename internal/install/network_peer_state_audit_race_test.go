package install

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	incus "github.com/lxc/incus/v7/client"
	"github.com/lxc/incus/v7/shared/api"
	"github.com/vanishs/boxweave/internal/state"
)

func s4b2AuditRaceStore(t *testing.T) (*state.Store, *sql.DB) {
	t.Helper()
	dir := t.TempDir()
	if e := os.Chmod(dir, 0700); e != nil {
		t.Fatal(e)
	}
	path := filepath.Join(dir, "audit.sqlite")
	s, e := state.Open(path)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = s.Close() })
	db, e := sql.Open("sqlite3", path)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = db.Close() })
	for i := 1; i <= 3; i++ {
		id := fmt.Sprintf("11111111-1111-4111-8111-%012d", i)
		name := fmt.Sprintf("b%d", i)
		v4 := fmt.Sprintf("10.243.18.%d", i+10)
		v6 := fmt.Sprintf("fd43:18::%d", i+10)
		mac := fmt.Sprintf("02:16:3e:11:12:%02x", i)
		now := time.Now().UnixNano()
		for _, q := range []struct {
			sql  string
			args []any
		}{
			{"INSERT INTO owned_boxes(uuid,project,name,incarnation,state,created_at) VALUES(?,?,?,?,'OWNED',?)", []any{id, "user-1001", name, id, now}},
			{"INSERT INTO network_bindings(project,box_uuid,incarnation,nic,bridge,acl,mac,ipv4,ipv6,state,observed_at) VALUES(?,?,?,?,?,?,?,?,?,'PINNED',?)", []any{"user-1001", id, id, "eth0", "bwbr0", "bw-s4b2", mac, v4, v6, now}},
			{"INSERT INTO network_address_fences(bridge,family,address,project,box_uuid,incarnation,state,observed_at) VALUES('bwbr0','ipv4',?,?,?,?,'RESERVED',?)", []any{v4, "user-1001", id, id, now}},
			{"INSERT INTO network_address_fences(bridge,family,address,project,box_uuid,incarnation,state,observed_at) VALUES('bwbr0','ipv6',?,?,?,?,'RESERVED',?)", []any{v6, "user-1001", id, id, now}},
		} {
			if _, e := db.Exec(q.sql, q.args...); e != nil {
				t.Fatal(e)
			}
		}
	}
	return s, db
}

// The embedded SDK interface exposes only the read methods used by the
// production scoped audit. Unexpected SDK calls fail instead of reaching Incus.
type s4b2ScopedAuditClient struct {
	incus.InstanceServer
	driver       *pr178RootNativeReadDriver
	project      string
	afterNetwork func(int) error
}

func (c *s4b2ScopedAuditClient) UseProject(project string) incus.InstanceServer {
	clone := *c
	clone.project = project
	return &clone
}

func (c *s4b2ScopedAuditClient) GetNetwork(name string) (*api.Network, string, error) {
	if c.project != "default" {
		return nil, "", errors.New("wrong network project")
	}
	n, tag, err := c.driver.network(name)
	if err != nil {
		return nil, "", err
	}
	if c.afterNetwork != nil {
		if err = c.afterNetwork(c.driver.networkReads); err != nil {
			return nil, "", err
		}
	}
	return n, tag, nil
}

func (c *s4b2ScopedAuditClient) GetInstanceNames(_ api.InstanceType) ([]string, error) {
	return c.driver.names(c.project)
}

func (c *s4b2ScopedAuditClient) GetInstance(name string) (*api.Instance, string, error) {
	return c.driver.instance(c.project, name)
}

func (c *s4b2ScopedAuditClient) GetNetworkACL(name string) (*api.NetworkACL, string, error) {
	if c.project != "default" {
		return nil, "", errors.New("wrong ACL project")
	}
	return c.driver.acl(name)
}

func TestS4B2RootAuditRefusesSQLiteAndNativeInterleaving(t *testing.T) {
	for _, tc := range []struct {
		name     string
		expected int
		fail     bool
	}{
		{"stable", 2, false},
		{"mac-drift", 2, true},
		{"lease-drift", 2, true},
		{"native-drift", 2, true},
		{"root-failure", 1, true},
		{"wrong-root-scope", 1, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			fixture, d := pr178RootSDKFixture(t)
			_, root, db, id := scopedInstallStore(t, fixture)
			proof, err := root.ReadRootScopedPeerOriginal(ctx, id, fixture.Intent.Project, fixture.Intent.Bridge)
			if err != nil || len(proof.Inventory.Boxes) != 4 {
				t.Fatal("complete scoped SQLite original unavailable", proof, err)
			}

			mutations := 0
			client := &s4b2ScopedAuditClient{
				driver: d,
				afterNetwork: func(calls int) error {
					var query, uuid string
					switch tc.name {
					case "mac-drift":
						if calls == 1 {
							query, uuid = "UPDATE network_bindings SET mac='02:16:3e:aa:bb:cc' WHERE box_uuid=?", pr178C
						}
					case "lease-drift":
						if calls == 2 {
							query, uuid = "UPDATE network_address_fences SET state='QUARANTINED' WHERE box_uuid=? AND family='ipv6'", pr178B
						}
					case "root-failure":
						return errors.New("native error")
					}
					if query != "" {
						r, err := db.ExecContext(ctx, query, uuid)
						if err != nil {
							t.Fatal(err)
						}
						if n, err := r.RowsAffected(); err != nil || n != 1 {
							t.Fatal("missing SQLite drift fixture", n, err)
						}
						mutations++
					}
					return nil
				},
			}
			switch tc.name {
			case "native-drift":
				// Change a sibling ACL on its second native read without changing its ETag.
				d.driftACL = true
			case "wrong-root-scope":
				d.bridge.Project = fixture.Intent.Project
			}

			rep, err := nativePeerScopedRootCollectReadOnly(ctx, root, client, id, fixture.Intent.Project, fixture.Intent.Bridge)
			if d.networkReads != tc.expected || (err != nil) != tc.fail {
				t.Fatalf("unsafe audit acceptance calls=%d report=%+v err=%v", d.networkReads, rep, err)
			}
			if tc.name == "mac-drift" || tc.name == "lease-drift" {
				if mutations != 1 {
					t.Fatal("SQLite drift was not injected", mutations)
				}
			}
			if tc.fail {
				if !reflect.DeepEqual(rep, nativePeerScopedRootReadResult{}) {
					t.Fatalf("failed audit published a positive scoped receipt: %+v", rep)
				}
				return
			}
			if !reflect.DeepEqual(rep.Original, proof) || rep.NativeACLBefore.ACLCount != 4 ||
				len(rep.NativeACLBefore.SHA256) != 64 || len(rep.InstanceETags) != 4 ||
				rep.Plan.UnrelatedACLsVerifiedUnchanged != 2 {
				t.Fatalf("incomplete scoped root audit: %+v", rep)
			}
			for _, b := range proof.Inventory.Boxes {
				if d.instReads[b.Name] != 2 || d.aclReads[b.ACL] != 2 {
					t.Fatal("owner or per-Box ACL was not independently reread", b.Name)
				}
			}
		})
	}
}

func TestS4B2RootAuditPreventsUnpinnedAndCancelledNativeRead(t *testing.T) {
	s, db := s4b2AuditRaceStore(t)
	calls := 0
	read := func(context.Context, NativePeerBridgeAuditTarget) (NativePeerBridgeAuditResult, error) {
		calls++
		return NativePeerBridgeAuditResult{}, errors.New("should not execute")
	}
	if _, e := db.Exec("UPDATE network_bindings SET state='QUARANTINED'"); e != nil {
		t.Fatal(e)
	}
	if _, e := nativePeerBridgeAuditFromStoreWithVerifier(t.Context(), s, "user-1001", "bwbr0", "bw-s4b2", read); e == nil || calls != 0 {
		t.Fatal("invalid owner sent to root", e)
	}
	if _, e := nativePeerBridgeAuditFromStoreWithVerifier(t.Context(), s, "user-1001", "bwbr0", "bw-s4b2", nil); e == nil {
		t.Fatal("nil native witness")
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, e := nativePeerBridgeAuditFromStoreWithVerifier(ctx, s, "user-1001", "bwbr0", "bw-s4b2", read); e == nil || calls != 0 {
		t.Fatal("canceled audit dispatched", e)
	}
}

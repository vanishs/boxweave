package control

import (
	"context"
	"encoding/json"
	"github.com/lxc/incus/v7/shared/api"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/vanishs/boxweave/internal/native"
	"testing"
	"time"
)

func TestS4CDetailOptInWireShape(t *testing.T) {
	basic := getBoxResult{TrackingID: "tracking", Box: native.BoxState{Name: "bw-test"}}
	encoded, err := json.Marshal(basic)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]json.RawMessage
	if err = json.Unmarshal(encoded, &wire); err != nil || len(wire) != 2 {
		t.Fatal("default shape changed")
	}
	now := time.Now().UTC()
	detail := basic
	detail.ObservedAt = &now
	full := &api.InstanceFull{Instance: api.Instance{InstancePut: api.InstancePut{Config: api.ConfigMap{"cloud-init.user-data": "synthetic-fixture"}}, Name: "bw-test", Status: "Stopped"}, Snapshots: []api.InstanceSnapshot{{}}, Backups: []api.InstanceBackup{{}}}
	detail.InstanceFull, err = nativeFullJSON(full)
	if err != nil {
		t.Fatal(err)
	}
	srv := mcp.NewServer(&mcp.Implementation{Name: "s4c-test", Version: "1"}, nil)
	addStrictTool(srv, &mcp.Tool{Name: "box_get", Annotations: readToolAnnotations()}, func(_ context.Context, _ *mcp.CallToolRequest, in getBoxInput) (*mcp.CallToolResult, getBoxResult, error) {
		if in.Detail {
			return &mcp.CallToolResult{}, detail, nil
		}
		return &mcp.CallToolResult{}, basic, nil
	})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ct, st := mcp.NewInMemoryTransports()
	ss, err := srv.Connect(ctx, st, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer ss.Close()
	cli := mcp.NewClient(&mcp.Implementation{Name: "client", Version: "1"}, nil)
	sess, err := cli.Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer sess.Close()
	for _, enabled := range []bool{false, true} {
		res, err := sess.CallTool(ctx, &mcp.CallToolParams{Name: "box_get", Arguments: map[string]any{"box_id": "test", "detail": enabled}})
		if err != nil || res == nil || res.IsError {
			t.Fatalf("SDK rejected detail=%t: %v", enabled, err)
		}
		data, err := json.Marshal(res.StructuredContent)
		if err != nil {
			t.Fatal(err)
		}
		var got struct {
			InstanceFull *api.InstanceFull `json:"instance_full"`
			ObservedAt   *time.Time        `json:"observed_at"`
		}
		if err = json.Unmarshal(data, &got); err != nil {
			t.Fatal(err)
		}
		if enabled {
			if got.InstanceFull == nil || got.ObservedAt == nil || got.InstanceFull.State != nil || len(got.InstanceFull.Snapshots) != 1 || len(got.InstanceFull.Backups) != 1 || got.InstanceFull.Config["cloud-init.user-data"] != "synthetic-fixture" {
				t.Fatal("native detail fields lost")
			}
		} else if got.InstanceFull != nil || got.ObservedAt != nil {
			t.Fatal("detail leaked by default")
		}
	}
}

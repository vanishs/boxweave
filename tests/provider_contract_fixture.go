// This BoxWeave-owned test helper is temporarily compiled inside the pinned
// Provider package. Infrastructure is synthetic; policy RPC/store code is real.
package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/footprintai/containarium/internal/auth"
	pb "github.com/footprintai/containarium/pkg/pb/containarium/v1"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc/metadata"
)

func TestBoxWeaveContractFixture(t *testing.T) {
	if os.Getenv("BOXWEAVE_CONTRACT_FIXTURE") != "1" {
		t.Skip("explicit source-contract fixture only")
	}
	ctx := auth.ContextWithTestSubjectScopes(context.Background(), "fixture-admin",
		[]string{auth.RoleAdmin}, []string{auth.ScopeContainersRead, auth.ScopeContainersWrite, auth.ScopeNetworkPolicyRead, auth.ScopeNetworkPolicyWrite})
	policy := NewNetworkPolicyServer(NewMemNetworkPolicyStore())
	policy.tenantStoreDurable = func() bool { return true } // fixture, not production durability
	policy.tenantGuardRead = func() *pb.GuardStatus {
		return &pb.GuardStatus{Mode: pb.GuardMode_GUARD_MODE_ENFORCE, FirewallDriver: "nftables", LastPass: time.Now().UTC().Format(time.RFC3339)}
	}
	_, err := policy.SetNetworkPolicy(ctx, &pb.SetNetworkPolicyRequest{Policy: &pb.NetworkPolicy{
		Tenant: "testing", Mode: pb.NetworkPolicyMode_NETWORK_POLICY_MODE_ENFORCE,
		EgressMode:  pb.NetworkPolicyMode_NETWORK_POLICY_MODE_LOG_ONLY,
		EgressCidrs: []string{"10.0.0.0/8"}, EgressDomains: []string{"example.com"},
		AllowMetadata: true, Source: "fixture-owned",
	}})
	if err != nil {
		t.Fatal(err)
	}
	mux := runtime.NewServeMux(runtime.WithMetadata(func(context.Context, *http.Request) metadata.MD {
		md, _ := metadata.FromIncomingContext(ctx)
		return md
	}))
	if err := pb.RegisterNetworkPolicyServiceHandlerServer(context.Background(), mux, policy); err != nil {
		t.Fatal(err)
	}
	network := &NetworkServer{nativeTenantLocal: true, nativeTenantReady: func() bool { return true }, atomicTenantAccessReady: policy.atomicTenantAccessAvailable}
	if err := pb.RegisterNetworkServiceHandlerServer(context.Background(), mux, network); err != nil {
		t.Fatal(err)
	}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer synthetic-contract-only" {
			http.Error(w, "unauthorized", 401)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/containers":
			var boxes []map[string]any
			for i, key := range []string{"allinone", "testing"} {
				boxes = append(boxes, map[string]any{"username": key, "name": key + "-container", "backendId": "fixture-local",
					"state": "CONTAINER_STATE_RUNNING", "network": map[string]any{"ipAddress": fmt.Sprintf("10.0.0.%d", i+2)},
					"networkTenant": key, "networkTenantSource": pb.NetworkTenantSource_NETWORK_TENANT_SOURCE_NATIVE.String()})
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"containers": boxes})
		case "/v1/backends":
			_ = json.NewEncoder(w).Encode(map[string]any{"backends": []any{map[string]any{"id": "fixture-local", "type": "local"}}})
		case "/v1/network/guard":
			var entries []map[string]any
			for i, key := range []string{"allinone", "testing"} {
				sum := sha256.Sum256([]byte(key))
				entries = append(entries, map[string]any{"container": key + "-container", "subject": key, "ip": fmt.Sprintf("10.0.0.%d", i+2), "aclName": "containarium-tenant-" + hex.EncodeToString(sum[:])[:12], "attached": true})
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"tenant": map[string]any{"mode": "GUARD_MODE_ENFORCE", "firewallDriver": "nftables", "lastPass": time.Now().UTC().Format(time.RFC3339), "entries": entries}})
		default:
			mux.ServeHTTP(w, r)
		}
	})
	server := httptest.NewServer(handler)
	defer server.Close()
	fmt.Println(server.URL)
	_, _ = io.Copy(io.Discard, os.Stdin)
}

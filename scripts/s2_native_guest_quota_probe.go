//go:build ignore

// Isolated test-only official Incus SDK guest-Exec quota witness.
// Never shipped as a BoxWeave CLI or run on a shared host.
package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/lxc/incus/v7/shared/api"
	"github.com/vanishs/boxweave/internal/native"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	if len(os.Args) != 2 || (os.Args[1] != "create" && os.Args[1] != "seed" && os.Args[1] != "flood") ||
		os.Getenv("GITHUB_ACTIONS") != "true" ||
		os.Getenv("RUNNER_ENVIRONMENT") != "github-hosted" || os.Geteuid() == 0 {
		panic("isolated nonroot Incus native quota lab required")
	}
	name := "bw-s2-quota-native-110h"
	project := os.Getenv("BW_S2_QUOTA_PROJECT")
	socket := os.Getenv("BW_S2_QUOTA_SOCKET")
	uuid := os.Getenv("BW_S2_QUOTA_UUID")
	if !strings.HasPrefix(project, "user-") || len(project) > 64 ||
		socket != "/var/lib/incus/unix.socket.user" {
		panic("invalid restricted quota lab")
	}
	r := native.Reader{Socket: socket, Project: project}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	if os.Args[1] == "create" {
		id, err := r.Create(ctx, native.InstanceAction{
			Name: name, Start: true, CPU: 1, MemoryMiB: 512,
			DiskGiB: 2, Internet: true,
		})
		must(err)
		op, err := r.ObserveUntilFinal(ctx, id)
		must(err)
		if op.StatusCode != api.Success {
			panic("native quota guest create did not complete successfully")
		}
		box, err := r.ObserveCreated(ctx, name)
		must(err)
		if box.UUID == "" || box.RootDiskSize != "2GiB" || box.NativeState != "Running" {
			panic("quota guest lacks the exact 2GiB running volume")
		}
		fmt.Println("S2_NATIVE_QUOTA_GUEST_UUID=" + box.UUID)
		return
	}
	if uuid == "" {
		panic("no exact native quota guest UUID")
	}
	spec := native.ExecSpec{Argv: []string{"/bin/sh", "-c", "printf initialized"}}
	if os.Args[1] == "flood" {
		// Only ~48MiB of the existing 2GiB quota remains when called.
		// The 128MiB stream must NOT fully fit in the native output spool.
		spec.Argv = []string{"/bin/sh", "-c", "head -c 134217728 /dev/zero"}
		spec.TimeoutMS = 30000
	}
	id, err := r.Exec(ctx, name, uuid, spec)
	must(err)
	fmt.Println("S2_NATIVE_QUOTA_OP_ID=" + id)
	op, err := r.ObserveUntilFinal(ctx, id)
	must(err)
	fmt.Printf("S2_NATIVE_QUOTA_OP_STATUS=%s\n", op.Status)
	if os.Args[1] == "seed" {
		ev, err := native.ExecEvidenceFromFinal(op)
		must(err)
		if ev.Exit == nil || *ev.Exit != 0 || ev.StdoutPath == "" {
			panic("native recorded spool seed not verified")
		}
		fmt.Println("S2_NATIVE_QUOTA_SPOOL_SEEDED=PASS")
	} else {
		// Do not infer guest exit 0 from native Operation success/failure.
		// The CI host reads the exact native operation's partial spool.
		fmt.Printf("S2_NATIVE_QUOTA_OP_STATUS_CODE=%d\n", op.StatusCode)
		if ev, err := native.ExecEvidenceFromFinal(op); err == nil && ev.Exit != nil {
			fmt.Printf("S2_NATIVE_QUOTA_GUEST_EXIT=%d\n", *ev.Exit)
		}
	}
}

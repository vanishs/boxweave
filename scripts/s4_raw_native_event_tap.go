//go:build ignore

// Test-only independent Incus 7.6 operation event observer.
// Subscribes before any fresh native effect; never dispatches or retries one.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	incus "github.com/lxc/incus/v7/client"
	"github.com/lxc/incus/v7/shared/api"
)

func fail(class string) {
	fmt.Fprintln(os.Stderr, "S4_EVENT_TAP_FAIL="+class)
	os.Exit(2)
}

func main() {
	socket := flag.String("socket", "", "restricted Incus user socket")
	project := flag.String("project", "", "single-use owned native project")
	output := flag.String("output", "", "0600 event receipt stream")
	ready := flag.String("ready", "", "0600 subscribe-ready marker")
	stop := flag.String("stop", "", "stop-file path")
	flag.Parse()
	if os.Getenv("GITHUB_ACTIONS") != "true" ||
		os.Getenv("RUNNER_ENVIRONMENT") != "github-hosted" ||
		os.Geteuid() == 0 ||
		*socket != "/var/lib/incus/unix.socket.user" ||
		!strings.HasPrefix(*project, "user-") ||
		!filepath.IsAbs(*output) || !filepath.IsAbs(*ready) ||
		!filepath.IsAbs(*stop) ||
		filepath.Dir(*output) != filepath.Dir(*ready) ||
		filepath.Dir(*ready) != filepath.Dir(*stop) {
		fail("UNSAFE_DISPOSABLE_SCOPE")
	}
	if _, err := os.Stat(*output); err == nil {
		fail("EXISTING_EVENT_RECEIPT")
	}
	if _, err := os.Stat(*ready); err == nil {
		fail("EXISTING_READY_MARKER")
	}
	client, err := incus.ConnectIncusUnix(*socket, nil)
	if err != nil {
		fail("INCUS_SOCKET_UNAVAILABLE")
	}
	events, err := client.UseProject(*project).GetEventsByType([]string{"operation"})
	if err != nil {
		fail("PROJECT_OPERATION_SUBSCRIPTION_DENIED")
	}
	defer events.Disconnect()
	ch := events.AddChannel([]string{"operation"}, 2048)
	file, err := os.OpenFile(*output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		fail("EVENT_RECEIPT_OPEN_FAILED")
	}
	defer file.Close()
	if err := file.Sync(); err != nil {
		fail("EVENT_RECEIPT_SYNC_FAILED")
	}
	if err := os.WriteFile(*ready, []byte("PROJECT_OPERATION_SUBSCRIBED\n"), 0600); err != nil {
		fail("READY_MARKER_FAILED")
	}
	disconnected := make(chan error, 1)
	go func() { disconnected <- events.Wait() }()
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	timer := time.NewTimer(12 * time.Minute)
	defer timer.Stop()
	seen := 0
	for {
		select {
		case event, ok := <-ch:
			if !ok {
				fail("OPERATION_STREAM_CLOSED")
			}
			if event.Type != "operation" ||
				(event.Project != "" && event.Project != *project) {
				continue
			}
			var op api.Operation
			if err := json.Unmarshal(event.Metadata, &op); err != nil {
				fail("INVALID_NATIVE_EVENT_METADATA")
			}
			code := int(op.StatusCode)
			if code != 200 && code != 400 && code != 401 {
				continue
			}
			if op.ID == "" {
				fail("MISSING_NATIVE_OPERATION_ID")
			}
			sum := sha256.Sum256(event.Metadata)
			receipt := map[string]any{
				"source": "INCUS_OPERATION_EVENT_WEBSOCKET",
				"project": *project,
				"event_project": event.Project,
				"native_operation_id": op.ID,
				"status_code": code,
				"status": op.Status,
				"error_present": op.Err != "",
				"raw_metadata_sha256": hex.EncodeToString(sum[:]),
				"timestamp_utc": event.Timestamp.UTC().Format(time.RFC3339Nano),
			}
			if err := json.NewEncoder(file).Encode(receipt); err != nil {
				fail("EVENT_RECEIPT_WRITE_FAILED")
			}
			if err := file.Sync(); err != nil {
				fail("EVENT_RECEIPT_SYNC_FAILED")
			}
			seen++
			if seen > 4096 {
				fail("EVENT_RECEIPT_BUDGET_EXCEEDED")
			}
		case <-tick.C:
			if _, err := os.Stat(*stop); err == nil {
				return
			} else if !os.IsNotExist(err) {
				fail("EVENT_STOP_MARKER_READ_FAILED")
			}
			if !events.IsActive() {
				fail("OPERATION_STREAM_DISCONNECTED")
			}
		case <-disconnected:
			fail("OPERATION_STREAM_DISCONNECTED")
		case <-timer.C:
			fail("EVENT_MONITOR_TIME_BUDGET_EXCEEDED")
		}
	}
}

//go:build ignore

// CI-only restricted Incus Unix proxy. A real native recorded-output DELETE
// succeeds once, then its response is deliberately dropped. Never installed.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
)

func main() {
	listen := flag.String("listen", "", "CI-only private Unix listener")
	upstream := flag.String("upstream", "", "restricted native Incus socket")
	marker := flag.String("marker", "", "original acceptance witness")
	project := flag.String("project", "", "owned disposable Project")
	nativeID := flag.String("native-id", "", "exact original native Exec operation ID")
	flag.Parse()
	if os.Geteuid() == 0 || os.Getenv("GITHUB_ACTIONS") != "true" ||
		os.Getenv("RUNNER_ENVIRONMENT") != "github-hosted" ||
		*upstream != "/var/lib/incus/unix.socket.user" ||
		!strings.HasPrefix(*project, "user-") || len(*nativeID) != 36 ||
		!strings.HasPrefix(*listen, "/tmp/bw-v1-s2-ghci.") ||
		!strings.HasSuffix(*listen, "/ledger/gc-delete-proxy.sock") ||
		!strings.HasPrefix(*marker, "/tmp/bw-v1-s2-ghci.") ||
		filepath.Dir(*marker) != filepath.Dir(*listen) {
		log.Fatal("S2 destructive ACK loss allowed only inside an exact nonroot disposable CI fixture")
	}
	for i, c := range *nativeID {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				log.Fatal("invalid native operation identifier")
			}
		} else if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			log.Fatal("invalid native operation identifier")
		}
	}
	for _, path := range []string{*listen, *marker} {
		if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
			log.Fatalf("refusing existing CI artifact: %s: %v", path, err)
		}
	}
	transport := &http.Transport{
		DisableKeepAlives: true,
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, "unix", *upstream)
		},
	}
	reverse := httputil.NewSingleHostReverseProxy(&url.URL{Scheme: "http", Host: "unix.socket"})
	reverse.Transport = transport
	reverse.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, _ error) {
		http.Error(w, "restricted CI native proxy request failed", http.StatusBadGateway)
	}
	path := fmt.Sprintf("/1.0/instances/bw-s2-mcp-quota-110i/logs/exec-output/exec_%s.stdout", *nativeID)
	var used atomic.Bool
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			reverse.ServeHTTP(w, r)
			return
		}
		if r.URL.Path != path || r.URL.Query().Get("project") != *project ||
			!used.CompareAndSwap(false, true) {
			_ = os.WriteFile(*marker+".unexpected", []byte("UNEXPECTED_SECOND_OR_FOREIGN_DELETE\n"), 0600)
			http.Error(w, "native DELETE outside exact one-use CI original refused", http.StatusConflict)
			return
		}
		req := r.Clone(r.Context())
		req.URL.Scheme, req.URL.Host, req.RequestURI = "http", "unix.socket", ""
		res, err := transport.RoundTrip(req)
		if err != nil {
			http.Error(w, "original native DELETE transport failed", http.StatusBadGateway)
			return
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusNoContent {
			http.Error(w, fmt.Sprintf("original native DELETE refused (%d)", res.StatusCode),
				http.StatusBadGateway)
			return
		}
		_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 1<<20))
		if err := os.WriteFile(*marker, []byte("GC_EXACT_NATIVE_STDOUT_DELETE_ACCEPTED_ACK_SUPPRESSED\n"), 0600); err != nil {
			http.Error(w, "native DELETE witness not durable", http.StatusBadGateway)
			return
		}
		hijacker, ok := w.(http.Hijacker)
		if !ok {
			http.Error(w, "native ACK drop unsupported", http.StatusBadGateway)
			return
		}
		conn, _, err := hijacker.Hijack()
		if err != nil {
			log.Fatal(err)
		}
		_ = conn.Close()
	})
	l, err := net.Listen("unix", *listen)
	if err != nil {
		log.Fatal(err)
	}
	if err := os.Chmod(*listen, 0600); err != nil {
		log.Fatal(err)
	}
	log.Println("S2_GC_DELETE_ACK_DROP_PROXY_LISTENING")
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second}
	if err := server.Serve(l); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

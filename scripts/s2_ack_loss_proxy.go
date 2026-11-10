//go:build ignore

// Single-use S2 Incus Unix-socket ACK-drop proxy. This is a test program,
// never a shipped BoxWeave service. The proxy runs as the same restricted OS
// actor as BoxWeave, against the same isolated Incus socket and Project.
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
	"strings"
	"sync/atomic"
	"time"
)

func main() {
	proxyPath := flag.String("listen", "", "private disposable Unix socket")
	upstreamPath := flag.String("upstream", "", "restricted Incus Unix socket")
	marker := flag.String("marker", "", "test-only observed upstream acceptance")
	flag.Parse()
	if os.Geteuid() == 0 || os.Getenv("GITHUB_ACTIONS") != "true" ||
		os.Getenv("RUNNER_ENVIRONMENT") != "github-hosted" ||
		!strings.HasPrefix(*proxyPath, "/tmp/bw-v1-s2-ghci.") ||
		!strings.Contains(*proxyPath, "/ledger/ack-proxy.sock") ||
		!strings.HasPrefix(*marker, "/tmp/bw-v1-s2-ghci.") ||
		*upstreamPath != "/var/lib/incus/unix.socket.user" {
		log.Fatal("S2 ACK injection only permitted for the one-use restricted GitHub VM")
	}
	if _, err := os.Lstat(*proxyPath); !errors.Is(err, os.ErrNotExist) {
		log.Fatalf("refusing preexisting listener: %v", err)
	}
	transport := &http.Transport{
		DisableKeepAlives: true,
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, "unix", *upstreamPath)
		},
	}
	upstreamURL := &url.URL{Scheme: "http", Host: "unix.socket"}
	reverse := httputil.NewSingleHostReverseProxy(upstreamURL)
	reverse.Transport = transport
	reverse.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, _ error) {
		http.Error(w, "native reverse proxy request unavailable", http.StatusBadGateway)
	}
	var trapUsed atomic.Bool
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/1.0/instances" {
			reverse.ServeHTTP(w, r)
			return
		}
		if !trapUsed.CompareAndSwap(false, true) {
			// A second original-create request after an uncertain ACK is
			// forbidden even if Incus would reject the duplicate name.
			file, err := os.OpenFile(*marker, os.O_WRONLY|os.O_APPEND, 0600)
			if err == nil {
				_, _ = file.WriteString("UNEXPECTED_SECOND_CREATE_POST\\n")
				_ = file.Close()
			}
			reverse.ServeHTTP(w, r)
			return
		}
		// The native POST is forwarded exactly ONCE. Only AFTER the
		// upstream daemon accepts the mutation do we suppress the response
		// to the BoxWeave SDK, creating a real uncertain-ACK boundary.
		request := r.Clone(r.Context())
		request.URL.Scheme = "http"
		request.URL.Host = "unix.socket"
		request.RequestURI = ""
		response, err := transport.RoundTrip(request)
		if err != nil {
			http.Error(w, "native unavailable", http.StatusBadGateway)
			return
		}
		defer response.Body.Close()
		body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
		if err != nil || response.StatusCode != http.StatusAccepted ||
			!strings.Contains(string(body), "\"operation\"") {
			http.Error(w, fmt.Sprintf("upstream native create not accepted (%d)", response.StatusCode),
				http.StatusBadGateway)
			return
		}
		if err := os.WriteFile(*marker, []byte("UPSTREAM_ACCEPTED_ONCE_AND_ACK_SUPPRESSED\n"), 0600); err != nil {
			log.Fatal(err)
		}
		hijacker, ok := w.(http.Hijacker)
		if !ok {
			http.Error(w, "cannot simulate dropped native response", http.StatusBadGateway)
			return
		}
		connection, _, err := hijacker.Hijack()
		if err != nil {
			log.Fatal(err)
		}
		_ = connection.Close()
	})
	listener, err := net.Listen("unix", *proxyPath)
	if err != nil {
		log.Fatal(err)
	}
	if err := os.Chmod(*proxyPath, 0600); err != nil {
		log.Fatal(err)
	}
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second}
	log.Println("S2_ACK_DROP_PROXY_LISTENING")
	if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

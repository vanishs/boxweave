//go:build ignore

// Single-use S2 Incus Unix-socket ACK-drop proxy. This is a test program,
// never a shipped BoxWeave service. The proxy runs as the same restricted OS
// actor as BoxWeave, against the same isolated Incus socket and Project.
package main

import (
	"bytes"
	"context"
	"encoding/json"
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

	incus "github.com/lxc/incus/v7/client"
)

func main() {
	proxyPath := flag.String("listen", "", "private disposable Unix socket")
	upstreamPath := flag.String("upstream", "", "restricted Incus Unix socket")
	marker := flag.String("marker", "", "test-only observed upstream acceptance")
	project := flag.String("project", "", "exact disposable restricted native Project")
	flag.Parse()
	if os.Geteuid() == 0 || os.Getenv("GITHUB_ACTIONS") != "true" ||
		os.Getenv("RUNNER_ENVIRONMENT") != "github-hosted" ||
		!strings.HasPrefix(*proxyPath, "/tmp/bw-v1-s2-ghci.") ||
		!strings.Contains(*proxyPath, "/ledger/ack-proxy.sock") ||
		!strings.HasPrefix(*marker, "/tmp/bw-v1-s2-ghci.") ||
		*upstreamPath != "/var/lib/incus/unix.socket.user" ||
		!strings.HasPrefix(*project, "user-") {
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
	var etagUsed atomic.Bool
	record := func(line string) error {
		f, err := os.OpenFile(*marker, os.O_WRONLY|os.O_APPEND, 0600)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = f.WriteString(line + "\n")
		return err
	}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Deterministic native ETag conflict: BoxWeave already selected its
		// If-Match. The one-use proxy advances the *actual same* Incus
		// instance ETag using a harmless user.* key before forwarding the
		// original stale request. This tests S2 fail-closed semantics.
		if r.Method == http.MethodPut &&
			r.URL.Path == "/1.0/instances/bw-s2-etag-native-110e" {
			if r.URL.Query().Get("project") != *project ||
				r.Header.Get("If-Match") == "" {
				http.Error(w, "unsafe or unguarded native ETag race", http.StatusBadRequest)
				return
			}
			if !etagUsed.CompareAndSwap(false, true) {
				_ = record("UNEXPECTED_SECOND_ETAG_PUT")
				reverse.ServeHTTP(w, r)
				return
			}
			d, err := incus.ConnectIncusUnix(*upstreamPath, nil)
			if err != nil {
				http.Error(w, "cannot connect to isolated Incus", 502)
				return
			}
			d = d.UseProject(*project)
			inst, originalTag, err := d.GetInstance("bw-s2-etag-native-110e")
			if err != nil {
				http.Error(w, "owned instance missing", 502)
				return
			}
			if originalTag == "" {
				http.Error(w, "native ETag missing", 502)
				return
			}
			if inst.Config == nil {
				inst.Config = map[string]string{}
			}
			inst.Config["user.s2-etag-race"] = "native-proxy-race-witness"
			op, err := d.UpdateInstance("bw-s2-etag-native-110e", inst.InstancePut, originalTag)
			if err != nil {
				http.Error(w, "native race update rejected", 502)
				return
			}
			if err := op.Wait(); err != nil {
				http.Error(w, "native race update did not settle", 502)
				return
			}
			if err := record("ETAG_RACE_NATIVE_CHANGED_BEFORE_STALE_PUT"); err != nil {
				http.Error(w, "test marker write failed", 502)
				return
			}
			req := r.Clone(r.Context())
			req.URL.Scheme = "http"
			req.URL.Host = "unix.socket"
			req.RequestURI = ""
			res, err := transport.RoundTrip(req)
			if err != nil {
				http.Error(w, "stale native request failed transport", 502)
				return
			}
			defer res.Body.Close()
			if res.StatusCode != http.StatusPreconditionFailed {
				http.Error(w, fmt.Sprintf("native stale ETag not rejected, status=%d", res.StatusCode), 502)
				return
			}
			if err := record("ETAG_RACE_ORIGINAL_STALE_PUT_REFUSED"); err != nil {
				http.Error(w, "test marker write failed", 502)
				return
			}
			for k, values := range res.Header {
				for _, v := range values {
					w.Header().Add(k, v)
				}
			}
			w.WriteHeader(res.StatusCode)
			_, _ = io.Copy(w, res.Body)
			return
		}
		if r.Method != http.MethodPost || r.URL.Path != "/1.0/instances" {
			reverse.ServeHTTP(w, r)
			return
		}
		if !trapUsed.CompareAndSwap(false, true) {
			// After the original Create ACK-drop, only the explicitly
			// named second ETag fixture is allowed; unexpected repeated
			// original names become an audit failure (never redelivered).
			raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
			if err != nil {
				http.Error(w, "bad create body", 400)
				return
			}
			_ = r.Body.Close()
			var intent map[string]any
			if json.Unmarshal(raw, &intent) != nil || intent["name"] != "bw-s2-etag-native-110e" ||
				r.URL.Query().Get("project") != *project {
				_ = record("UNEXPECTED_SECOND_CREATE_POST")
				http.Error(w, "unapproved second native create", 409)
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(raw))
			if err := record("ETAG_FIXTURE_CREATE_FORWARDED"); err != nil {
				http.Error(w, "test marker failure", 502)
				return
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

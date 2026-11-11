//go:build ignore

// Disposable-only Incus response hold. It forwards the actual upstream GET,
// then delays delivery to the installed BoxWeave handler while the authorized
// test actor revokes its OWN token. Never shipped or used on shared Incus.
package main

import (
	"context"
	"errors"
	"flag"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
func one(path, value string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	if _, err = io.WriteString(f, value); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
func main() {
	listen := flag.String("listen", "", "single-use restricted socket")
	upstream := flag.String("upstream", "", "native restricted Incus socket")
	project := flag.String("project", "", "exact restricted native project")
	name := flag.String("name", "", "sole owned guest identity")
	dir := flag.String("dir", "", "private per-run control directory")
	flag.Parse()
	if os.Geteuid() == 0 || os.Getenv("GITHUB_ACTIONS") != "true" ||
		os.Getenv("RUNNER_ENVIRONMENT") != "github-hosted" ||
		*upstream != "/var/lib/incus/unix.socket.user" ||
		!strings.HasPrefix(*dir, "/tmp/bw-v1-s2-ghci.") ||
		filepath.Base(*dir) != "p1-private-read" ||
		*listen != filepath.Join(*dir, "proxy.sock") ||
		*project != "user-"+os.Getenv("BW_S2_TEST_UID") ||
		*name != "bw-s2-read-revoke-110r" || exists(*listen) {
		log.Fatal("refusing non-disposable or unexpected private-read proxy")
	}
	transport := &http.Transport{DisableKeepAlives: true,
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, "unix", *upstream)
		},
	}
	reverse := httputil.NewSingleHostReverseProxy(&url.URL{Scheme: "http", Host: "unix.socket"})
	reverse.Transport = transport
	reverse.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, _ error) {
		http.Error(w, "disposable native read unavailable", http.StatusBadGateway)
	}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mode := ""
		if r.Method == http.MethodGet {
			// A separate P2 scenario holds the actual native recorded-output
			// HTTP 200 while the MCP caller disconnects. This is NOT a
			// synthetic stream or a canceled guest native Operation.
			if exists(filepath.Join(*dir, "arm-stalled-output")) &&
				strings.HasPrefix(r.URL.Path, "/1.0/instances/"+*name+"/logs/exec-output/") &&
				r.URL.Query().Get("project") == *project {
				mode = "stalled-output"
			}
			if mode == "" {
			for _, m := range []string{"host_status", "box_list", "box_get", "operation_output_get"} {
				if !exists(filepath.Join(*dir, "arm-"+m)) {
					continue
				}
				if m == "host_status" && r.URL.Path == "/1.0/projects/"+*project {
					mode = m
				}
				if m != "host_status" && r.URL.Path == "/1.0/instances/"+*name &&
					r.URL.Query().Get("project") == *project {
					mode = m
				}
				break
			}
			}
		}
		if mode == "" || exists(filepath.Join(*dir, "held-"+mode)) {
			reverse.ServeHTTP(w, r)
			return
		}
		req := r.Clone(r.Context())
		req.URL.Scheme, req.URL.Host, req.RequestURI = "http", "unix.socket", ""
		res, err := transport.RoundTrip(req)
		if err != nil {
			http.Error(w, "real upstream native read failed", http.StatusBadGateway)
			return
		}
		defer res.Body.Close()
		body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
		if err != nil || res.StatusCode != http.StatusOK {
			http.Error(w, "real upstream native read not successful", http.StatusBadGateway)
			return
		}
		if err := one(filepath.Join(*dir, "held-"+mode), "UPSTREAM_HTTP_200_HELD\n"); err != nil {
			http.Error(w, "read hold witness unavailable", http.StatusBadGateway)
			return
		}
		released := false
		for i := 0; i < 200; i++ {
			if exists(filepath.Join(*dir, "release-"+mode)) {
				released = true
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		if !released {
			http.Error(w, "test release was never authorized", http.StatusGatewayTimeout)
			return
		}
		for k, vals := range res.Header {
			for _, v := range vals {
				w.Header().Add(k, v)
			}
		}
		w.WriteHeader(res.StatusCode)
		_, _ = w.Write(body)
	})
	listener, err := net.Listen("unix", *listen)
	if err != nil {
		log.Fatal(err)
	}
	if err := os.Chmod(*listen, 0600); err != nil {
		log.Fatal(err)
	}
	log.Println("S2_P1_PRIVATE_NATIVE_READ_DELAY_PROXY_READY")
	if err := (&http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second}).Serve(listener); err != nil &&
		!errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

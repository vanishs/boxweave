//go:build ignore

// Restricted single-use real Incus 202 ACK-loss proxy for S2 Stop/Delete.
// No native operation is retried. No arbitrary path, Project or host socket
// can be changed; never ship this helper with the BoxWeave executable.
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
	"path/filepath"
	"regexp"
	"strings"
	"sync/atomic"
	"time"
)

var nativeOpID = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func singleWitness(path, value string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	_, err = io.WriteString(f, value)
	if err == nil {
		err = f.Sync()
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	return err
}

func main() {
	listen := flag.String("listen", "", "one-use Unix listener")
	upstream := flag.String("upstream", "", "real restricted Incus Unix socket")
	project := flag.String("project", "", "exact restricted native Project")
	phase := flag.String("phase", "", "stop or delete")
	marker := flag.String("marker", "", "durable real native accepted-202 proof")
	flag.Parse()
	if os.Geteuid() == 0 || os.Getenv("GITHUB_ACTIONS") != "true" ||
		os.Getenv("RUNNER_ENVIRONMENT") != "github-hosted" ||
		!strings.HasPrefix(*listen, "/tmp/bw-v1-s2-ghci.") ||
		(*phase != "stop" && *phase != "delete") ||
		*listen != filepath.Join(filepath.Dir(*listen), "proxy.sock") ||
		filepath.Base(filepath.Dir(*listen)) != "p1-"+*phase ||
		*marker != filepath.Join(filepath.Dir(*listen), "native-accepted.txt") ||
		*upstream != "/var/lib/incus/unix.socket.user" ||
		!strings.HasPrefix(*project, "user-") {
		log.Fatal("native Stop/Delete ACK-loss proxy confined to one-use hosted CI")
	}
	for _, f := range []string{*listen, *marker, *marker + ".unexpected"} {
		if _, err := os.Lstat(f); !errors.Is(err, os.ErrNotExist) {
			log.Fatal("refusing existing native/ACK witness or listener")
		}
	}
	name := "bw-s2-p1-"+*phase+"-110p"
	method, path := http.MethodPut, "/1.0/instances/"+name+"/state"
	if *phase == "delete" {
		method, path = http.MethodDelete, "/1.0/instances/"+name
	}
	transport := &http.Transport{DisableKeepAlives: true, DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, "unix", *upstream)
	}}
	reverse := httputil.NewSingleHostReverseProxy(&url.URL{Scheme: "http", Host: "unix.socket"})
	reverse.Transport = transport
	reverse.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, _ error) {
		http.Error(w, "test native socket unavailable", http.StatusBadGateway)
	}
	var fired atomic.Bool
	httpHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			reverse.ServeHTTP(w, r)
			return
		}
		if r.Method != method || r.URL.Path != path || r.URL.Query().Get("project") != *project ||
			!fired.CompareAndSwap(false, true) {
			_ = singleWitness(*marker+".unexpected", "SECOND_OR_FOREIGN_NATIVE_MUTATION_REJECTED\n")
			http.Error(w, "unapproved or duplicate native mutation denied", http.StatusConflict)
			return
		}
		data, err := io.ReadAll(io.LimitReader(r.Body, 1<<16))
		if err != nil {
			http.Error(w, "unreadable native mutation", http.StatusBadRequest)
			return
		}
		if *phase == "stop" {
			var state struct {
				Action string `json:"action"`
			}
			if json.Unmarshal(data, &state) != nil || state.Action != "stop" {
				http.Error(w, "native action was not exact Stop", http.StatusBadRequest)
				return
			}
		} else if len(bytes.TrimSpace(data)) != 0 {
			http.Error(w, "native Delete included unexpected mutation body", http.StatusBadRequest)
			return
		}
		req := r.Clone(r.Context())
		req.URL.Scheme, req.URL.Host, req.RequestURI = "http", "unix.socket", ""
		req.Body = io.NopCloser(bytes.NewReader(data))
		req.ContentLength = int64(len(data))
		resp, err := transport.RoundTrip(req)
		if err != nil {
			http.Error(w, "native transport did not return accepted status", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		var result struct {
			Operation string `json:"operation"`
		}
		jsonErr := json.Unmarshal(body, &result)
		u, urlErr := url.Parse(result.Operation)
		nativeID := ""
		if u != nil && strings.HasPrefix(u.Path, "/1.0/operations/") {
			nativeID = strings.TrimPrefix(u.Path, "/1.0/operations/")
		}
		if err != nil || resp.StatusCode != http.StatusAccepted || jsonErr != nil ||
			urlErr != nil || !nativeOpID.MatchString(nativeID) {
			http.Error(w, "expected real native 202 and exact Operation ID", http.StatusBadGateway)
			return
		}
		if err := singleWitness(*marker, fmt.Sprintf("S2_P1_NATIVE_%s_202_ACCEPTED_ID=%s\n",
			strings.ToUpper(*phase), nativeID)); err != nil {
			http.Error(w, "cannot persist unique native accepted receipt", http.StatusBadGateway)
			return
		}
		// The native operation really took place. Only its HTTP acknowledgment
		// to the BoxWeave SDK is discarded. Never issue a second request.
		hijack, ok := w.(http.Hijacker)
		if !ok {
			http.Error(w, "cannot drop exact native 202 ACK", http.StatusBadGateway)
			return
		}
		conn, _, err := hijack.Hijack()
		if err != nil {
			log.Fatal(err)
		}
		_ = conn.Close()
	})
	listener, err := net.Listen("unix", *listen)
	if err != nil {
		log.Fatal(err)
	}
	if err := os.Chmod(*listen, 0600); err != nil {
		log.Fatal(err)
	}
	log.Println("S2_P1_NATIVE_RESTRICTED_STOP_DELETE_ACK_PROXY_READY")
	server := &http.Server{Handler: httpHandler, ReadHeaderTimeout: 5 * time.Second}
	if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

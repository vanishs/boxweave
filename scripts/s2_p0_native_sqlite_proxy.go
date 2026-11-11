//go:build ignore

// Single-use GitHub-hosted Incus Unix proxy: one exact owned Update receives
// an ACTUAL native 202 before arming the SQLite WAL sync-cut in BoxWeave.
// No installed proxy, arbitrary native operation or host service is exposed.
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

var nativeUUID = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func writeDurable(path, value string) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	if _, err = f.WriteString(value); err == nil {
		err = f.Sync()
	}
	if e := f.Close(); err == nil {
		err = e
	}
	return err
}

func main() {
	listen := flag.String("listen", "", "disposable restricted Unix listener")
	upstream := flag.String("upstream", "", "restricted native Incus socket")
	project := flag.String("project", "", "exact one-use native Project")
	name := flag.String("name", "", "exact pre-created owned instance name")
	arm := flag.String("arm", "", "armed SQLite fault witness")
	marker := flag.String("marker", "", "real upstream accepted native operation")
	flag.Parse()
	dir := filepath.Dir(*listen)
	phase := strings.TrimPrefix(filepath.Base(dir), "p0-")
	wantName := "bw-s2-p0-before-110j"
	if phase == "after" {
		wantName = "bw-s2-p0-after-110k"
	}
	if os.Geteuid() == 0 || os.Getenv("GITHUB_ACTIONS") != "true" ||
		os.Getenv("RUNNER_ENVIRONMENT") != "github-hosted" ||
		(phase != "before" && phase != "after") || *name != wantName ||
		*upstream != "/var/lib/incus/unix.socket.user" ||
		!strings.HasPrefix(*project, "user-") ||
		!strings.HasPrefix(dir, "/tmp/bw-v1-s2-ghci.") ||
		*listen != filepath.Join(dir, "proxy.sock") ||
		*marker != filepath.Join(dir, "native-202.txt") ||
		*arm != filepath.Join(dir, "arm") {
		log.Fatal("restricted native+SQLite combined fault only allowed in exact one-use CI lab")
	}
	for _, path := range []string{*listen, *arm, *marker} {
		if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
			log.Fatal("refusing preexisting P0 native fault witness")
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
		http.Error(w, "restricted native test proxy failed", http.StatusBadGateway)
	}
	var used atomic.Bool
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			reverse.ServeHTTP(w, r)
			return
		}
		if r.Method != http.MethodPut || r.URL.Path != "/1.0/instances/"+*name ||
			r.URL.Query().Get("project") != *project || r.Header.Get("If-Match") == "" ||
			!used.CompareAndSwap(false, true) {
			_ = writeDurable(filepath.Join(dir, "unexpected-native-dispatch.txt"), "FORBIDDEN_NATIVE_MUTATION\n")
			http.Error(w, "unapproved or second native dispatch refused", http.StatusConflict)
			return
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			http.Error(w, "unreadable original update body", http.StatusBadRequest)
			return
		}
		var request struct {
			Config map[string]string `json:"config"`
		}
		if err := json.Unmarshal(body, &request); err != nil || request.Config["limits.cpu"] != "2" {
			http.Error(w, "unapproved Update intent", http.StatusBadRequest)
			return
		}
		req := r.Clone(r.Context())
		req.URL.Scheme, req.URL.Host, req.RequestURI = "http", "unix.socket", ""
		req.Body = io.NopCloser(bytes.NewReader(body))
		req.ContentLength = int64(len(body))
		response, err := transport.RoundTrip(req)
		if err != nil {
			http.Error(w, "original native Update transport failed", http.StatusBadGateway)
			return
		}
		defer response.Body.Close()
		ack, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
		var native struct {
			Operation string `json:"operation"`
		}
		jsonErr := json.Unmarshal(ack, &native)
		nativeURL, urlErr := url.Parse(native.Operation)
		nativeID := ""
		if nativeURL != nil && strings.HasPrefix(nativeURL.Path, "/1.0/operations/") {
			nativeID = strings.TrimPrefix(nativeURL.Path, "/1.0/operations/")
		}
		log.Printf("S2_P0_SCOPED_NATIVE_UPDATE_HTTP_STATUS=%d native_id_well_formed=%t",
			response.StatusCode, nativeUUID.MatchString(nativeID))
		if err != nil || response.StatusCode != http.StatusAccepted || jsonErr != nil ||
			urlErr != nil || !nativeUUID.MatchString(nativeID) ||
			nativeURL.Query().Get("project") != "" && nativeURL.Query().Get("project") != *project {
			http.Error(w, "exact original native operation was not accepted", http.StatusBadGateway)
			return
		}
		if err := writeDurable(*marker, "NATIVE_REAL_UPDATE_202_ACCEPTED="+nativeID+"\n"); err != nil {
			http.Error(w, "cannot record original native acceptance", http.StatusBadGateway)
			return
		}
		// This file is created ONLY after real Incus accepted the original
		// Update, but before the original 202 reaches BoxWeave's SDK.
		if err := writeDurable(*arm, "arm-on-real-native-202\n"); err != nil {
			http.Error(w, "cannot arm exact SQLite WAL COMMIT cut", http.StatusBadGateway)
			return
		}
		for k, values := range response.Header {
			for _, v := range values {
				w.Header().Add(k, v)
			}
		}
		w.WriteHeader(response.StatusCode)
		_, _ = w.Write(ack)
	})
	listener, err := net.Listen("unix", *listen)
	if err != nil {
		log.Fatal(err)
	}
	if err := os.Chmod(*listen, 0600); err != nil {
		log.Fatal(err)
	}
	fmt.Println("S2_P0_RESTRICTED_COMBINED_NATIVE_SQLITE_PROXY_LISTENING")
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second}
	if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

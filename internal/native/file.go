package native

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path"
	"sort"
	"strings"
	"sync/atomic"

	"github.com/pkg/sftp"
)

// File access stays inside the official Incus instance SFTP endpoint.
// v1 deliberately does not provide host filesystem paths or a custom agent.
const (
	MaxGuestFileBytes   = 64 << 10
	MaxGuestFileEntries = 4096
	// pkg/sftp v1 ReadDirContext buffers the entire Guest directory. Bound
	// received bytes before the library can decode unbounded directory data.
	MaxGuestFileListWireBytes = 512 << 10
)

var ErrGuestFileExists = errors.New("exclusive guest file already exists; no create performed")

// ErrGuestFilePreDispatchDenied proves SFTP OpenFile was never called.
// It is safe to finalize a local FAILED receipt without creating UNKNOWN.
var ErrGuestFilePreDispatchDenied = errors.New("unsafe guest file rejected before SFTP mutation")

type GuestFileEntry struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Size int64  `json:"size"`
}

type GuestFileListing struct {
	Entries            []GuestFileEntry `json:"entries"`
	NextCursor         string           `json:"next_cursor,omitempty"`
	ConsistentSnapshot bool             `json:"consistent_snapshot"`
}

type GuestFilePage struct {
	Data       []byte `json:"-"`
	Size       int64  `json:"size"`
	Offset     int64  `json:"offset"`
	Ended      bool   `json:"ended"`
	WholeFile  bool   `json:"whole_file"`
	PageSHA256 string `json:"page_sha256"`
}

// Guest file paths are absolute, byte-bounded and canonical inside the Guest.
// Kernel pseudo-filesystems and file-descriptor magic paths are never exposed.
// A second component-by-component Lstat rejects symlink traversal.
func SafeGuestFilePath(p string) bool {
	if len(p) < 2 || len(p) > 512 || !strings.HasPrefix(p, "/") ||
		path.Clean(p) != p || strings.IndexByte(p, 0) >= 0 ||
		strings.IndexAny(p, string([]byte{92, 10, 13})) >= 0 {
		return false
	}
	parts := strings.Split(strings.TrimPrefix(p, "/"), "/")
	for _, part := range parts {
		if part == "" || part == "." || part == ".." || len(part) > 255 {
			return false
		}
	}
	switch parts[0] {
	case "proc", "sys", "dev", "run":
		return false
	}
	if len(parts) > 1 && parts[0] == "var" && parts[1] == "run" {
		return false
	}
	return true
}

func guestFileType(info os.FileInfo) string {
	switch {
	case info.Mode().IsRegular():
		return "file"
	case info.IsDir():
		return "directory"
	case info.Mode()&os.ModeSymlink != 0:
		return "symlink"
	default:
		return "special"
	}
}

// Each path component is explicitly Lstat'd. The final component may be
// absent only when the subsequent request uses O_CREAT|O_EXCL. This is scoped
// to trusted supervised guests; SFTP v3 does not offer openat2/RESOLVE_BENEATH
// and cannot promise adversarial concurrent parent-directory rename safety.
func verifyGuestComponents(c *sftp.Client, p string, finalType string, allowMissing bool) error {
	if !SafeGuestFilePath(p) {
		return ErrNativeUnsupported
	}
	parts := strings.Split(strings.TrimPrefix(p, "/"), "/")
	segment := ""
	for i, part := range parts {
		segment += "/" + part
		info, err := c.Lstat(segment)
		last := i == len(parts)-1
		if err != nil {
			if last && allowMissing && errors.Is(err, os.ErrNotExist) {
				return nil
			}
			return err
		}
		if !last {
			if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
				return ErrNativeUnsupported
			}
			continue
		}
		if allowMissing {
			return ErrGuestFileExists
		}
		if guestFileType(info) != finalType {
			return ErrNativeUnsupported
		}
	}
	return nil
}

// sftpReadBudget limits transport bytes fed to the eager v1 SFTP decoder,
// not just the number of records returned to the caller. SFTP v3 has no
// native server-side snapshot or stable pagination cursor.
type sftpReadBudget struct {
	reader    io.Reader
	remaining int64
	exhausted atomic.Bool
}

func (b *sftpReadBudget) Read(p []byte) (int, error) {
	if b.remaining <= 0 {
		b.exhausted.Store(true)
		return 0, ErrNativeUnsupported
	}
	if int64(len(p)) > b.remaining {
		p = p[:int(b.remaining)]
	}
	n, err := b.reader.Read(p)
	b.remaining -= int64(n)
	return n, err
}

// Uses the existing official Incus native SFTP connection and pinned client.
// The transport guard limits memory consumed by SFTP directory enumeration;
// it is NOT a guarantee about total Go process RSS or concurrent Guest races.
func newBoundedSFTPClient(conn io.ReadWriteCloser, maxReceived int64) (*sftp.Client, *sftpReadBudget, error) {
	if conn == nil || maxReceived < 1024 || maxReceived > MaxGuestFileListWireBytes {
		return nil, nil, ErrNativeUnsupported
	}
	reader := &sftpReadBudget{reader: conn, remaining: maxReceived}
	client, err := sftp.NewClientPipe(reader, conn)
	if err != nil {
		_ = conn.Close()
		return nil, nil, err
	}
	return client, reader, nil
}

func (r Reader) guestSFTP(ctx context.Context, name, uuid string, listWireLimit int64) (*sftp.Client, func(), *sftpReadBudget, error) {
	box, _, err := r.GetExactBox(ctx, name, uuid)
	if err != nil {
		return nil, nil, nil, err
	}
	if box.NativeState != "Running" {
		return nil, nil, nil, ErrNativeUnsupported
	}
	c, err := r.confined(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	var client *sftp.Client
	var budget *sftpReadBudget
	if listWireLimit > 0 {
		conn, err := c.GetInstanceFileSFTPConn(name)
		if err != nil {
			return nil, nil, nil, err
		}
		client, budget, err = newBoundedSFTPClient(conn, listWireLimit)
		if err != nil {
			return nil, nil, nil, err
		}
	} else {
		client, err = c.GetInstanceFileSFTP(name)
		if err != nil {
			return nil, nil, nil, err
		}
	}
	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = client.Close()
		case <-done:
		}
	}()
	return client, func() {
		close(done)
		_ = client.Close()
	}, budget, nil
}

// ListGuestFiles is a bounded response with a bounded incoming SFTP stream.
// The v1 client still buffers results up to the wire budget; the reply is
// *not* a directory snapshot and lexical cursors may miss concurrent changes.
func (r Reader) ListGuestFiles(ctx context.Context, name, uuid, p, after string, limit int) (GuestFileListing, error) {
	var out GuestFileListing
	if !SafeGuestFilePath(p) || limit < 0 || limit > 50 || len(after) > 255 ||
		strings.IndexAny(after, string([]byte{47, 92, 0, 10, 13})) >= 0 || after == "." || after == ".." {
		return out, ErrNativeUnsupported
	}
	if limit == 0 {
		limit = 20
	}
	c, closeClient, budget, err := r.guestSFTP(ctx, name, uuid, MaxGuestFileListWireBytes)
	if err != nil {
		return out, err
	}
	defer closeClient()
	if err := verifyGuestComponents(c, p, "directory", false); err != nil {
		return out, err
	}
	infos, err := c.ReadDirContext(ctx, p)
	if budget.exhausted.Load() {
		return out, ErrNativeUnsupported
	}
	if err != nil {
		return out, err
	}
	if len(infos) > MaxGuestFileEntries {
		return out, ErrNativeUnsupported
	}
	sort.Slice(infos, func(i, j int) bool { return infos[i].Name() < infos[j].Name() })
	out.Entries = make([]GuestFileEntry, 0, limit)
	for _, info := range infos {
		if info.Name() <= after || len(info.Name()) > 255 {
			continue
		}
		if len(out.Entries) == limit {
			out.NextCursor = out.Entries[len(out.Entries)-1].Name
			break
		}
		out.Entries = append(out.Entries, GuestFileEntry{
			Name: info.Name(), Type: guestFileType(info), Size: info.Size(),
		})
	}
	// A native name can be deleted/reused during a long SFTP enumeration.
	// Do not disclose a result if it no longer belongs to the admitted UUID.
	if _, _, err := r.GetExactBox(ctx, name, uuid); err != nil {
		return GuestFileListing{}, err
	}
	return out, nil
}

// ReadGuestFile gives a page SHA, never a fabricated full-file hash. Only
// an unchanged stat before/after an entire bounded read permits WholeFile.
func (r Reader) ReadGuestFile(ctx context.Context, name, uuid, p string, offset int64, length int) (GuestFilePage, error) {
	var out GuestFilePage
	if !SafeGuestFilePath(p) || offset < 0 || offset > 1<<30 ||
		length < 1 || length > MaxGuestFileBytes {
		return out, ErrNativeUnsupported
	}
	c, closeClient, _, err := r.guestSFTP(ctx, name, uuid, 0)
	if err != nil {
		return out, err
	}
	defer closeClient()
	if err := verifyGuestComponents(c, p, "file", false); err != nil {
		return out, err
	}
	before, err := c.Lstat(p)
	if err != nil || !before.Mode().IsRegular() {
		return out, ErrNativeUnsupported
	}
	f, err := c.Open(p)
	if err != nil {
		return out, err
	}
	defer f.Close()
	buf := make([]byte, length)
	n, readErr := f.ReadAt(buf, offset)
	if readErr != nil && !errors.Is(readErr, io.EOF) {
		return out, readErr
	}
	after, err := c.Lstat(p)
	if err != nil || !after.Mode().IsRegular() || after.Size() != before.Size() ||
		!after.ModTime().Equal(before.ModTime()) {
		return out, ErrUnverified
	}
	// A sparse/early EOF is not proof that a concurrent actor didn't change
	// bytes and restore metadata; this only observes a bounded read.
	if offset+int64(n) < before.Size() && readErr == io.EOF {
		return out, ErrUnverified
	}
	data := buf[:n]
	sum := sha256.Sum256(data)
	out = GuestFilePage{
		Data: data, Size: before.Size(), Offset: offset,
		Ended:      offset+int64(n) >= before.Size(),
		WholeFile:  offset == 0 && int64(n) == before.Size(),
		PageSHA256: hex.EncodeToString(sum[:]),
	}
	// Recheck the exact native Box incarnation at the response boundary;
	// source name/project checks before SFTP cannot alone prevent reuse.
	if _, _, err := r.GetExactBox(ctx, name, uuid); err != nil {
		return GuestFilePage{}, err
	}
	return out, nil
}

// CreateGuestFile performs exactly one O_EXCL SFTP mutation and independently
// reopens the created original path to verify bytes. A connection/write/close
// uncertainty is NEVER classified as "no native effect" and must leave the
// BoxWeave original UNKNOWN. SFTP does not return a native Operation ID.
func (r Reader) CreateGuestFile(ctx context.Context, name, uuid, p string, data []byte) error {
	if !SafeGuestFilePath(p) || len(data) > MaxGuestFileBytes {
		return ErrGuestFilePreDispatchDenied
	}
	c, closeClient, _, err := r.guestSFTP(ctx, name, uuid, 0)
	if err != nil {
		// No mutating file Open was issued. A failed setup is not UNKNOWN.
		return ErrGuestFilePreDispatchDenied
	}
	defer closeClient()
	if err = verifyGuestComponents(c, p, "file", true); err != nil {
		if errors.Is(err, ErrGuestFileExists) {
			return err
		}
		// Component Lstat failures, a symlink ancestor, a special file and
		// an absent parent all occur before the single native O_EXCL Open.
		return ErrGuestFilePreDispatchDenied
	}
	f, err := c.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return ErrGuestFileExists
		}
		return err
	}
	n, writeErr := f.Write(data)
	closeErr := f.Close()
	if writeErr != nil || closeErr != nil || n != len(data) {
		return ErrUnverified
	}
	if _, _, err = r.GetExactBox(ctx, name, uuid); err != nil {
		return err
	}
	if err = verifyGuestComponents(c, p, "file", false); err != nil {
		return ErrUnverified
	}
	read, err := c.Open(p)
	if err != nil {
		return err
	}
	defer read.Close()
	observed, err := io.ReadAll(io.LimitReader(read, MaxGuestFileBytes+1))
	if err != nil || len(observed) != len(data) {
		return ErrUnverified
	}
	observedDigest := sha256.Sum256(observed)
	expectedDigest := sha256.Sum256(data)
	if observedDigest != expectedDigest {
		return ErrUnverified
	}
	// Verify UUID again after actual Guest byte readback; no successful
	// receipt is allowed if the instance was replaced during verification.
	if _, _, err := r.GetExactBox(ctx, name, uuid); err != nil {
		return err
	}
	return nil
}

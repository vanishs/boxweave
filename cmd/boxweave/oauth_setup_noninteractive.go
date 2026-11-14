package main

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/vanishs/boxweave/internal/state"
	"golang.org/x/sys/unix"
)

// setupOAuthNoninteractive keeps OAuth bootstrapping host-local and separate
// from the public MCP API. Secret values never enter argv, environment variables,
// stdout, stderr or the system journal. Operators retrieve them only from a
// protected file on the exact host where the new controller is installed.
func setupOAuthNoninteractive(s *state.Store, generate bool, outputFile, passwordFile string) error {
	if generate {
		if outputFile == "" || passwordFile != "" {
			return errors.New("oauth-setup -generate requires -output-file and forbids -password-file")
		}
		random := make([]byte, 32)
		if _, err := rand.Read(random); err != nil {
			return fmt.Errorf("generate OAuth approval password: %w", err)
		}
		password := base64.RawURLEncoding.EncodeToString(random)
		clear(random)
		if err := createOAuthSecretFile(outputFile, password); err != nil {
			return err
		}
		// On persistence failure, remove the as-yet-unused new secret. Do not
		// silently reuse a prior file or expose the password in the error.
		if err := setupOAuthPasswordAfterPrompt(s, password); err != nil {
			_ = os.Remove(outputFile)
			return err
		}
		fmt.Fprintf(os.Stdout, "OAuth approval password stored at %s (0600). Read it on the trusted host terminal; do not copy it into chat or logs.\n", outputFile)
		return nil
	}
	if outputFile != "" || passwordFile == "" {
		return errors.New("oauth-setup noninteractive mode requires exactly one of -generate -output-file or -password-file")
	}
	password, err := readOAuthSecretFile(passwordFile)
	if err != nil {
		return err
	}
	return setupOAuthPasswordAfterPrompt(s, password)
}

// The immediate parent must be a private directory owned by the executing
// nonroot service account. Do not accept relative paths, symlinks, or /tmp
// as the final secrets directory.
func validateOAuthSecretPath(path string) error {
	if path == "" || !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return errors.New("OAuth secret file requires a clean absolute path")
	}
	parent, err := os.Lstat(filepath.Dir(path))
	if err != nil {
		return fmt.Errorf("inspect OAuth secret directory: %w", err)
	}
	if parent.Mode()&os.ModeSymlink != 0 || !parent.IsDir() || parent.Mode().Perm()&0077 != 0 {
		return errors.New("OAuth secret directory must be a real owner-only directory (0700)")
	}
	st, ok := parent.Sys().(*syscall.Stat_t)
	if !ok || st.Uid != uint32(os.Geteuid()) {
		return errors.New("OAuth secret directory must belong to the service account")
	}
	return nil
}

// O_EXCL prevents clobbering an existing credential. O_NOFOLLOW rejects
// a symlink at the final component. No plaintext goes to terminal output.
func createOAuthSecretFile(path, password string) error {
	if err := validateOAuthSecretPath(path); err != nil {
		return err
	}
	fd, err := unix.Open(path, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0600)
	if err != nil {
		return fmt.Errorf("create new OAuth secret file exclusively: %w", err)
	}
	file := os.NewFile(uintptr(fd), path)
	_, writeErr := io.WriteString(file, password+"\n")
	if writeErr == nil {
		writeErr = file.Sync()
	}
	closeErr := file.Close()
	if writeErr == nil {
		writeErr = closeErr
	}
	if writeErr != nil {
		_ = os.Remove(path)
		return fmt.Errorf("persist OAuth secret file: %w", writeErr)
	}
	return nil
}

// The existing secret must be a single-link, regular, owner-controlled file.
// The password may end with one newline for ordinary provisioning tooling;
// embedded newlines and other control characters are not accepted.
func readOAuthSecretFile(path string) (string, error) {
	if err := validateOAuthSecretPath(path); err != nil {
		return "", err
	}
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return "", fmt.Errorf("open protected OAuth secret file: %w", err)
	}
	file := os.NewFile(uintptr(fd), path)
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("inspect OAuth secret file: %w", err)
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !info.Mode().IsRegular() || !ok || st.Uid != uint32(os.Geteuid()) || st.Nlink != 1 || info.Mode().Perm()&0077 != 0 {
		return "", errors.New("OAuth secret file must be regular, owner-only, single-link, and owned by the service account")
	}
	b, err := io.ReadAll(io.LimitReader(file, 258))
	if err != nil {
		return "", fmt.Errorf("read OAuth secret file: %w", err)
	}
	if len(b) > 257 {
		return "", errors.New("OAuth secret file exceeds the maximum password size")
	}
	password := strings.TrimSuffix(string(b), "\n")
	if len(password) < 16 || len(password) > 256 || strings.ContainsAny(password, "\n\r\x00") {
		return "", errors.New("OAuth secret file requires exactly one password of 16-256 characters")
	}
	return password, nil
}

package main

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

// A second serve must never invoke Store.Recover while the first process
// still owns in-flight native work. The file is intentionally never removed:
// flock follows the inode, and unlinking a locked file allows split locks.
func acquireServeLock(dbPath string) (*os.File, error) {
	path := dbPath + ".serve.lock"
	fd, err := syscall.Open(path, syscall.O_RDWR|syscall.O_CREAT|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0600)
	if err != nil {
		return nil, fmt.Errorf("open private serve lock: %w", err)
	}
	f := os.NewFile(uintptr(fd), path)
	var st syscall.Stat_t
	if err := syscall.Fstat(fd, &st); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("stat serve lock: %w", err)
	}
	if st.Mode&syscall.S_IFMT != syscall.S_IFREG ||
		st.Mode&0777 != 0600 || st.Nlink != 1 || st.Uid != uint32(os.Geteuid()) {
		_ = f.Close()
		return nil, errors.New("unsafe private serve lock file")
	}
	if err := syscall.Flock(fd, syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
			return nil, errors.New("serve already active for this database")
		}
		return nil, fmt.Errorf("acquire serve lock: %w", err)
	}
	return f, nil
}

//go:build !s1lab

package install

import "os/exec"

func nativeMigrationLabAptCommand(cmd *exec.Cmd) *exec.Cmd { return cmd }

// No runtime environment can trigger a production native migration crash.
func nativeMigrationLabCrashRequested() bool { return false }

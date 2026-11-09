//go:build !s1lab

package install

// No runtime environment can trigger a production native migration crash.
func nativeMigrationLabCrashRequested() bool { return false }

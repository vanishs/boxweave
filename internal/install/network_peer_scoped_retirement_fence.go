package install

// CheckScopedRetirementFence is a fixed local mutation admission check.
func CheckScopedRetirementFence() error { return refuseScopedRetirementInProgress() }

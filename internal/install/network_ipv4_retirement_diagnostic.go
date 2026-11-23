package install

import (
	"context"
	"errors"
	"os"
)

type retirementProofError struct {
	reason string
	cause  error
}

func (e retirementProofError) Error() string { return e.reason }
func (e retirementProofError) Unwrap() error { return e.cause }
func retirementWireClass(s string) string {
	switch s {
	case "RETIREMENT_DEADLINE", "RETIREMENT_LEDGER_NOT_QUIESCENT_OR_INELIGIBLE", "RETIREMENT_NATIVE_PROOF_UNAVAILABLE", "RETIREMENT_BRIDGE_NOT_QUIESCENT", "RETIREMENT_NATIVE_IDENTITY_UNVERIFIED", "RETIREMENT_PROXY_NOT_CLEAR", "RETIREMENT_LEASE_NOT_CLEAR", "RETIREMENT_KERNEL_REFERENCE_UNVERIFIED", "RETIREMENT_CONNTRACK_NOT_CLEAR", "RETIREMENT_POLICY_UNVERIFIED", "RETIREMENT_COMPACTION_PROTECTED", "RETIREMENT_HELPER_NOT_CONFIGURED":
		return s
	}
	return "RETIREMENT_NATIVE_PROOF_UNAVAILABLE"
}
func retirementFailure(reason string, cause error) error {
	return retirementProofError{retirementWireClass(reason), cause}
}
func NativeRetirementFailureClass(e error) string {
	if errors.Is(e, context.Canceled) || errors.Is(e, context.DeadlineExceeded) {
		return "RETIREMENT_DEADLINE"
	}
	var r retirementProofError
	if errors.As(e, &r) {
		return retirementWireClass(r.reason)
	}
	return "RETIREMENT_NATIVE_PROOF_UNAVAILABLE"
}
func NativeRetirementSocketPresent() bool {
	st, e := os.Lstat(retirementClientSocket())
	return e == nil && st.Mode()&os.ModeSocket != 0
}

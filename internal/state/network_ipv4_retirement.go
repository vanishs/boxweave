package state

// Historical diagnostic wire shape only; schema 7 has no reclamation queue.
type IPv4RetirementStatus struct {
	Pending              int    `json:"pending"`
	ProtectedCompactions int    `json:"protected_compactions"`
	LastDeferredReason   string `json:"last_deferred_reason,omitempty"`
}

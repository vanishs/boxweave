package state

import "context"

// Removed schema-6 execution. Historical caller signatures refuse before any SQL or native witness.

type IPv4RetirementCandidate struct {
	Project, Bridge, ACL                 string
	Binding                              NetworkAuditBinding
	CreateOperationID, DeleteOperationID string
	OperationEpoch, EventEpoch           int64
	RequiresPeerCompaction               bool
}

type IPv4RetirementWitness func(context.Context, IPv4RetirementCandidate) (IPv4RetirementCandidate, string, error)

type IPv4RetirementPolicy struct {
	Candidate       IPv4RetirementCandidate
	Ledger          NetworkAuditLedger
	Active, Revoked []PeerGrantBinding
	AfterRevoked    []PeerGrantBinding
	Forwards        []ForwardClaim
}

type IPv4RetirementStatus struct {
	Pending              int    `json:"pending"`
	ProtectedCompactions int    `json:"protected_compactions"`
	LastDeferredReason   string `json:"last_deferred_reason,omitempty"`
}

func (s *Store) ReadIPv4RetirementCandidate(ctx context.Context, project, original string) (IPv4RetirementCandidate, error) {
	return IPv4RetirementCandidate{}, ErrResourceLocked
}

func (s *Store) RetireVerifiedIPv4(ctx context.Context, project, original string, witness IPv4RetirementWitness) error {
	return ErrResourceLocked
}

func (s *Store) ReadIPv4RetirementPolicy(ctx context.Context, want IPv4RetirementCandidate) (IPv4RetirementPolicy, error) {
	return IPv4RetirementPolicy{}, ErrResourceLocked
}

func (s *Store) ClaimPendingIPv4Retirements(ctx context.Context, project string, limit int) ([]string, error) {
	return nil, ErrResourceLocked
}

func (s *Store) RecordIPv4RetirementDeferred(ctx context.Context, project, original, reason string) error {
	return ErrResourceLocked
}

func (s *Store) ReadIPv4RetirementStatus(ctx context.Context, project string) (IPv4RetirementStatus, error) {
	return IPv4RetirementStatus{}, ErrResourceLocked
}

func (s *Store) ReadIPv4RetirementRollbackWitness(ctx context.Context, project string) (string, error) {
	return "", ErrResourceLocked
}

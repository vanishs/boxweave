package state

import "context"

// Removed schema-6 execution. Historical caller signatures refuse before any SQL or native witness.

type IPv4CompactionPreview func(context.Context, IPv4RetirementCandidate) (string, error)

func (s *Store) PrepareIPv4Compaction(ctx context.Context, project, original string, preview IPv4CompactionPreview) (bool, error) {
	return false, ErrResourceLocked
}

func (s *Store) ReadIPv4CompactionPlan(ctx context.Context, project, original string) (string, error) {
	return "", ErrResourceLocked
}

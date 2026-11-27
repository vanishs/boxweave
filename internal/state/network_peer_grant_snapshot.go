package state

import "context"

// Removed schema-6 execution. Historical caller signatures refuse before any SQL or native witness.

type PeerGrantBinding struct {
	SourceUUID, SourceIncarnation, TargetUUID, TargetIncarnation string
	SourceIPv4, TargetIPv4, SourceIPv6, TargetIPv6               string
}

func (s *Store) ReadPeerGrantSnapshot(ctx context.Context, project, bridge, acl, stagedOriginal string) ([]PeerGrantBinding, error) {
	return nil, ErrResourceLocked
}

func (s *Store) readPeerGrantSnapshot(ctx context.Context, project, bridge, acl, stagedOriginal string, allowUnknownRecovery bool) ([]PeerGrantBinding, error) {
	return nil, ErrResourceLocked
}

func (s *Store) readPeerGrantSnapshotMode(ctx context.Context, project, bridge, acl, stagedOriginal string, allowUnknownRecovery, allowEmpty bool) ([]PeerGrantBinding, error) {
	return nil, ErrResourceLocked
}

func (s *Store) ReadPeerRevokedSnapshot(ctx context.Context, project, bridge, acl string) ([]PeerGrantBinding, error) {
	return nil, ErrResourceLocked
}

func (s *Store) readPeerRevokedSnapshot(ctx context.Context, project, bridge, acl string, allowEmpty bool) ([]PeerGrantBinding, error) {
	return nil, ErrResourceLocked
}

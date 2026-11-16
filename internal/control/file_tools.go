package control

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
)

type fileListInput struct {
	BoxID  string `json:"box_id"`
	Path   string `json:"path"`
	Cursor string `json:"cursor,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

type fileListResult struct {
	TrackingID         string                  `json:"tracking_id"`
	BoxID              string                  `json:"box_id"`
	Path               string                  `json:"path"`
	Entries            []native.GuestFileEntry `json:"entries"`
	NextCursor         string                  `json:"next_cursor,omitempty"`
	ConsistentSnapshot bool                    `json:"consistent_snapshot"`
}

type fileReadInput struct {
	BoxID  string `json:"box_id"`
	Path   string `json:"path"`
	Offset int64  `json:"offset,omitempty"`
	Length int    `json:"length,omitempty"`
}

type fileReadResult struct {
	TrackingID string `json:"tracking_id"`
	BoxID      string `json:"box_id"`
	Path       string `json:"path"`
	DataBase64 string `json:"data_base64"`
	Size       int64  `json:"size"`
	Offset     int64  `json:"offset"`
	Ended      bool   `json:"ended"`
	WholeFile  bool   `json:"whole_file"`
	PageSHA256 string `json:"page_sha256"`
}

type fileApplyInput struct {
	BoxID          string `json:"box_id"`
	Path           string `json:"path"`
	Mode           string `json:"mode"`
	ContentBase64  string `json:"content_base64,omitempty"`
	IdempotencyKey string `json:"idempotency_key"`
	WorkRef        string `json:"work_ref,omitempty"`
	AgentRef       string `json:"agent_ref,omitempty"`
}

func (s *Service) registerFileTools(server *mcp.Server) {
	addStrictTool(server, &mcp.Tool{
		Name:        "box_file_list",
		Description: "List an owned Guest directory using bounded Incus SFTP metadata, lexical cursor and honest non-snapshot pagination. No host paths or symlink traversal.",
		Annotations: readToolAnnotations(),
	}, func(ctx context.Context, req *mcp.CallToolRequest, in fileListInput) (*mcp.CallToolResult, fileListResult, error) {
		actor, _, _, err := s.operationPrincipal(req)
		if err != nil {
			return nil, fileListResult{}, err
		}
		if !native.SafeGuestFilePath(in.Path) || in.BoxID == "" {
			return nil, fileListResult{}, state.ErrInvalid
		}
		box, err := s.readOwned(ctx, req, in.BoxID)
		if err != nil {
			return nil, fileListResult{}, err
		}
		page, err := s.Native.ListGuestFiles(ctx, box.Name, box.UUID, in.Path, in.Cursor, in.Limit)
		if err != nil {
			return nil, fileListResult{}, err
		}
		tracking, err := newTrackingID()
		if err != nil {
			return nil, fileListResult{}, err
		}
		out := fileListResult{
			TrackingID: tracking, BoxID: box.UUID, Path: in.Path,
			Entries: page.Entries, NextCursor: page.NextCursor,
			ConsistentSnapshot: page.ConsistentSnapshot,
		}
		return privateReadReply(ctx, s, actor, out)
	})

	addStrictTool(server, &mcp.Tool{
		Name:        "box_file_read",
		Description: "Read at most 64KiB per page from an exact owned Guest regular file; base64 bytes and page SHA256, never an invented full-file hash for partial data.",
		Annotations: readToolAnnotations(),
	}, func(ctx context.Context, req *mcp.CallToolRequest, in fileReadInput) (*mcp.CallToolResult, fileReadResult, error) {
		actor, _, _, err := s.operationPrincipal(req)
		if err != nil {
			return nil, fileReadResult{}, err
		}
		if !native.SafeGuestFilePath(in.Path) || in.BoxID == "" {
			return nil, fileReadResult{}, state.ErrInvalid
		}
		if in.Length == 0 {
			in.Length = 16384
		}
		box, err := s.readOwned(ctx, req, in.BoxID)
		if err != nil {
			return nil, fileReadResult{}, err
		}
		page, err := s.Native.ReadGuestFile(ctx, box.Name, box.UUID, in.Path, in.Offset, in.Length)
		if err != nil {
			return nil, fileReadResult{}, err
		}
		tracking, err := newTrackingID()
		if err != nil {
			return nil, fileReadResult{}, err
		}
		out := fileReadResult{
			TrackingID: tracking, BoxID: box.UUID, Path: in.Path,
			DataBase64: base64.StdEncoding.EncodeToString(page.Data),
			Size:       page.Size, Offset: page.Offset, Ended: page.Ended,
			WholeFile: page.WholeFile, PageSHA256: page.PageSHA256,
		}
		return privateReadReply(ctx, s, actor, out)
	})

	addStrictTool(server, &mcp.Tool{
		Name:        "box_file_apply",
		Description: "Durable owned-Guest file create with exclusive SFTP O_EXCL and verified byte readback. Modes create|replace|delete; unsafe conditional replace/delete explicitly UNSUPPORTED. No host paths, guest patch agent, raw file data in SQLite or fictitious native Operation ID.",
		Annotations: mutationAnnotations(true),
	}, func(ctx context.Context, req *mcp.CallToolRequest, in fileApplyInput) (*mcp.CallToolResult, state.Receipt, error) {
		if _, _, err := requireAdmin(req, s); err != nil {
			return nil, state.Receipt{}, err
		}
		if in.BoxID == "" || !native.SafeGuestFilePath(in.Path) ||
			(in.Mode != "create" && in.Mode != "replace" && in.Mode != "delete") {
			return nil, state.Receipt{}, state.ErrInvalid
		}
		if in.Mode != "create" {
			return nil, state.Receipt{}, fmt.Errorf("UNSUPPORTED: native SFTP cannot atomically condition replace/delete against a concurrent Guest writer: %w", native.ErrNativeUnsupported)
		}
		if len(in.ContentBase64) > base64.StdEncoding.EncodedLen(native.MaxGuestFileBytes) {
			return nil, state.Receipt{}, state.ErrInvalid
		}
		payload, err := base64.StdEncoding.Strict().DecodeString(in.ContentBase64)
		if err != nil || len(payload) > native.MaxGuestFileBytes {
			return nil, state.Receipt{}, state.ErrInvalid
		}
		sum := sha256.Sum256(payload)
		intent := struct {
			BoxID  string `json:"box_id"`
			Path   string `json:"path"`
			Mode   string `json:"mode"`
			Bytes  int    `json:"bytes"`
			SHA256 string `json:"sha256"`
		}{in.BoxID, in.Path, "create", len(payload), hex.EncodeToString(sum[:])}
		// Check the durable original BEFORE native Box preflight: a vanished
		// Box never changes the authority or meaning of an original receipt.
		original, duplicate, err := s.replayNative(ctx, req, "box_file_apply",
			in.IdempotencyKey, in.BoxID, in.WorkRef, in.AgentRef, intent)
		if err != nil {
			return nil, state.Receipt{}, err
		}
		if duplicate {
			return &mcp.CallToolResult{}, original, nil
		}
		box, err := s.readOwned(ctx, req, in.BoxID)
		if err != nil {
			return nil, state.Receipt{}, err
		}
		rec, err := s.admitNative(ctx, req, "box_file_apply", in.IdempotencyKey,
			box.UUID, box.Incarnation, "box:"+box.Name,
			in.WorkRef, in.AgentRef, intent)
		if err != nil {
			return nil, state.Receipt{}, err
		}
		// SFTP is synchronous and has no Incus Operation ID. Claim and
		// finalization are durable; any lost reply/write is kept UNKNOWN.
		effect := func(eCtx context.Context) error {
			return s.Native.CreateGuestFile(eCtx, box.Name, box.UUID, in.Path, payload)
		}
		if err := s.worker().SubmitGuestFileCreate(rec.ID, box, effect); err != nil {
			return &mcp.CallToolResult{}, rec, errors.Join(err, state.ErrState)
		}
		return &mcp.CallToolResult{}, rec, nil
	})
}

package native

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// DeleteExactRecordedOutput uses only the pinned official SDK HTTP transport,
// exact original native operation and Project, never filesystem paths on the
// controller host. On any ambiguous network, identity or postread error the
// caller MUST journal REVIEW_REQUIRED without automatically reissuing DELETE.
func (r Reader) DeleteExactRecordedOutput(ctx context.Context,
	name, uuid, nativeID, stream, path string) error {
	if !safeName.MatchString(name) || uuid == "" ||
		(stream != "stdout" && stream != "stderr") ||
		len(nativeID) != 36 ||
		path != fmt.Sprintf("/1.0/instances/%s/logs/exec-output/exec_%s.%s", name, nativeID, stream) ||
		strings.Contains(path, "..") {
		return ErrNativeUnsupported
	}
	if _, _, err := r.GetExactBox(ctx, name, uuid); err != nil {
		return err
	}
	c, err := r.confined(ctx)
	if err != nil {
		return err
	}
	info, err := c.GetConnectionInfo()
	if err != nil || info == nil {
		return ErrUnverified
	}
	target, err := verifiedRecordedOutputURL(info.URL, path, r.Project)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, target, nil)
	if err != nil {
		return err
	}
	resp, err := c.DoHTTP(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("native recorded-output DELETE HTTP %d", resp.StatusCode)
	}
	// A 200 alone does not prove the original bytes were removed. Recheck
	// the same original native UUID and verify the exact file is absent.
	if _, _, err = r.GetExactBox(ctx, name, uuid); err != nil {
		return err
	}
	_, _, err = r.ReadRecordedOutput(ctx, path, 0, 1)
	if errors.Is(err, ErrRecordedOutputUnavailable) {
		return nil
	}
	if err != nil {
		return err
	}
	return ErrUnverified
}

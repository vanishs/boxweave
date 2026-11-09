package native

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"

	incus "github.com/lxc/incus/v7/client"
	"github.com/lxc/incus/v7/shared/api"
)

// BoxObservation is a native read-only observation, not a resolution of the
// original UNKNOWN operation. ABSENT is never proof of no previous side effect.
type BoxObservation struct {
	State        string `json:"state"`
	Project      string `json:"project"`
	Name         string `json:"name"`
	ExpectedUUID string `json:"expected_uuid"`
	Fingerprint  string `json:"observation_digest"`
}

func (r Reader) ObserveOwnedBox(ctx context.Context, name, uuid string) (BoxObservation, error) {
	out := BoxObservation{Project: r.Project, Name: name, ExpectedUUID: uuid}
	if name == "" || uuid == "" {
		return out, errors.New("exact native name and UUID required")
	}
	pre := r.Read(ctx)
	if !pre.Ready {
		return out, fmt.Errorf("native restricted preflight: %s", pre.ErrorClass)
	}
	client, err := incus.ConnectIncusUnixWithContext(ctx, r.Socket, nil)
	if err != nil {
		return out, err
	}
	scoped := client.UseProject(r.Project)
	instance, etag, err := scoped.GetInstance(name)
	switch {
	case api.StatusErrorCheck(err, http.StatusNotFound):
		out.State = "ABSENT_UNCERTAIN"
	case err != nil:
		return out, err
	default:
		if instance.Config["volatile.uuid"] != uuid {
			return out, errors.New("native identity changed: refuse recovery challenge or release")
		}
		out.State = "PRESENT_UUID_CONFIRMED"
	}
	h := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%s|%s|%s",
		out.Project, out.Name, out.ExpectedUUID, out.State, etag)))
	out.Fingerprint = hex.EncodeToString(h[:])
	return out, nil
}

package native

import (
	"context"
	"time"

	"github.com/lxc/incus/v7/shared/api"
)

// BoxFullObservation is one native observation, not an atomic snapshot.
type BoxFullObservation struct {
	Box          BoxState
	InstanceFull *api.InstanceFull
	ObservedAt   time.Time
}

type fullInstanceReader interface {
	HasExtension(string) bool
	GetInstanceFull(string) (*api.InstanceFull, string, error)
}

// The SDK fallback reads state, snapshots and backups separately by name;
// it cannot prove that all responses describe the exact same incarnation.
func (r Reader) GetExactBoxFull(ctx context.Context, name, uuid string) (BoxFullObservation, error) {
	if !safeName.MatchString(name) || uuid == "" {
		return BoxFullObservation{}, ErrNotOwned
	}
	c, err := r.confined(ctx)
	if err != nil {
		return BoxFullObservation{}, err
	}
	return readExactBoxFull(c, r.Project, name, uuid)
}

func readExactBoxFull(c fullInstanceReader, project, name, uuid string) (BoxFullObservation, error) {
	if !safeName.MatchString(name) || project == "" || uuid == "" {
		return BoxFullObservation{}, ErrNotOwned
	}
	if !c.HasExtension("instance_get_full") {
		return BoxFullObservation{}, ErrNativeUnsupported
	}
	full, _, err := c.GetInstanceFull(name)
	if err != nil {
		return BoxFullObservation{}, err
	}
	if full == nil || full.Name != name || full.Config["volatile.uuid"] != uuid ||
		(full.Project != "" && full.Project != project) {
		return BoxFullObservation{}, ErrNotOwned
	}
	return BoxFullObservation{Box: observedBoxState(project, name, uuid, &full.Instance),
		InstanceFull: full, ObservedAt: time.Now().UTC()}, nil
}

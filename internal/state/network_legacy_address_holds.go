package state

import (
	"context"
	"database/sql"

	"github.com/vanishs/boxweave/internal/networklimits"
)

// Only the Legacy reader may exclude independently verified archive rows.
func readLegacyCreateAddressHoldsTx(ctx context.Context, tx *sql.Tx, project, bridge string) (networklimits.AddressHolds, error) {
	if e := verifyIPv4RetirementHistory(ctx, tx, project, bridge, ""); e != nil {
		return networklimits.AddressHolds{}, e
	}
	prefix := "create-pin:" + bridge + ":"
	return collectCreateAddressHoldsTx(ctx, tx, project, bridge,
		addressHoldQuery{addressHoldBindingsQuery + legacyHeldBindingsSuffix, []any{bridge, project}},
		addressHoldQuery{addressHoldClaimsQuery + legacyHeldClaimsSuffix, []any{len(prefix), prefix, project, bridge}})
}

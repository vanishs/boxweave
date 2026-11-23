package native

import "context"

// ObserveCreateRecoveryIdle is a conservative, restricted Project-only
// witness used around an exact original Create marker when its native ACK
// was lost. An active or unrecognized operation blocks recovery. Empty
// inventory alone is never proof of an absent Create effect.
func (r Reader) ObserveCreateRecoveryIdle(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c, err := r.confined(ctx)
	if err != nil {
		return err
	}
	ops, err := c.GetOperations()
	if err != nil {
		return err
	}
	for _, op := range ops {
		if op.ID == "" || !op.StatusCode.IsFinal() {
			return ErrUnverified
		}
	}
	return nil
}

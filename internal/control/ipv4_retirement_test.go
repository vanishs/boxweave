package control

import (
	"context"
	"testing"
	"time"
)

func TestIPv4RetirementLoopRetriesWithoutRestart(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	attempts := 0
	runIPv4RetirementLoop(ctx, time.Millisecond, func(work context.Context) {
		if work.Err() != nil {
			t.Fatal(work.Err())
		}
		attempts++
		if attempts == 3 {
			cancel()
		}
	})
	if attempts != 3 {
		t.Fatal(attempts)
	}
}

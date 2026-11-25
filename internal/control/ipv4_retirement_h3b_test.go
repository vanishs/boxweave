package control

import (
	"fmt"
	"testing"
	"time"
)

func TestH3BRetirementDeferralLogLimiter(t *testing.T) {
	var l retirementLogLimiter
	now := time.Now()
	expect := func(id, reason string, at time.Time, want bool, count int) {
		t.Helper()
		a, b := l.observe(id, reason, at)
		if a != want || b != count {
			t.Fatal(a, b, want, count)
		}
	}
	expect("a", "first", now, true, 0)
	expect("a", "first", now.Add(time.Second), false, 0)
	expect("a", "first", now.Add(2*time.Second), false, 0)
	expect("a", "changed", now.Add(3*time.Second), true, 2)
	expect("a", "changed", now.Add(4*time.Second), false, 0)
	expect("a", "changed", now.Add(6*time.Minute), true, 1)
	expect("a", "changed", now.Add(6*time.Minute+time.Second), false, 0)
	if l.clear("a") != 1 {
		t.Fatal("lost repeat count")
	}
	for i := 0; i < retirementLogLimit+5; i++ {
		expect(fmt.Sprint(i), "first", now.Add(time.Duration(i)*time.Second), true, 0)
	}
	if len(l.entries) != retirementLogLimit {
		t.Fatal("unbounded limiter")
	}
}

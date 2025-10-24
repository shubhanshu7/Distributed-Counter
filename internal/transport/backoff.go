package transport

import (
	"math"
	"time"
)

type backoffState struct {
	base     time.Duration
	factor   float64
	max      time.Duration
	maxTries int
	attempt  int
}

func newBackoff(base time.Duration, factor float64, max time.Duration, maxTries int) *backoffState {
	return &backoffState{base: base, factor: factor, max: max, maxTries: maxTries}
}

func (b *backoffState) Next() (time.Duration, bool) {
	if b.maxTries > 0 && b.attempt >= b.maxTries {
		return 0, false
	}
	d := float64(b.base) * math.Pow(b.factor, float64(b.attempt))
	b.attempt++
	if time.Duration(d) > b.max {
		return b.max, true
	}
	return time.Duration(d), true
}

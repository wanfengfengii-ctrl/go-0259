package domain

import "sync"

// Clock is a controllable, monotonic logical clock. The API surface exposes a
// way to advance it so that deterministic tests can position events at precise
// logical times; production only ever calls Now and Set.
type Clock struct {
	mu sync.Mutex
	t  LogicalTime
}

// NewClock returns a Clock starting at zero.
func NewClock() *Clock {
	return &Clock{}
}

// Now returns the current logical time without advancing it.
func (c *Clock) Now() LogicalTime {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

// Tick advances the clock by one and returns the new time.
func (c *Clock) Tick() LogicalTime {
	c.t++
	return c.t
}

// Set forces the clock to at least t, preserving monotonicity.
func (c *Clock) Set(t LogicalTime) LogicalTime {
	c.mu.Lock()
	defer c.mu.Unlock()
	if t > c.t {
		c.t = t
	}
	return c.t
}

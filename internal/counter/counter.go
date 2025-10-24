package counter

import (
	"counter/internal/types"
	"sync"
)

type Counter struct {
	mu         sync.RWMutex
	components map[string]int64
	lastSeq    map[string]int64
	selfID     string
	selfSeq    int64
}

func New(selfID string) *Counter {
	c := &Counter{
		components: make(map[string]int64),
		lastSeq:    make(map[string]int64),
		selfID:     selfID,
	}
	c.components[selfID] = 0
	c.lastSeq[selfID] = 0
	return c
}

func (c *Counter) LocalIncrement() (local, global int64, msg types.ApplyMsg) {
	c.mu.Lock()
	c.selfSeq++
	c.components[c.selfID] = c.selfSeq
	local = c.components[c.selfID]
	for _, v := range c.components {
		global += v
	}
	msg = types.ApplyMsg{NodeID: c.selfID, Seq: c.selfSeq, Value: c.components[c.selfID]}
	c.mu.Unlock()
	return
}

func (c *Counter) Apply(msg types.ApplyMsg) {
	c.mu.Lock()
	if last, ok := c.lastSeq[msg.NodeID]; ok && msg.Seq <= last { // dedupe
		c.mu.Unlock()
		return
	}
	if cur, ok := c.components[msg.NodeID]; !ok || msg.Value > cur { // G-Counter max
		c.components[msg.NodeID] = msg.Value
	}
	if curSeq, ok := c.lastSeq[msg.NodeID]; !ok || msg.Seq > curSeq {
		c.lastSeq[msg.NodeID] = msg.Seq
	}
	c.mu.Unlock()
}

func (c *Counter) MergeFullState(state types.FullState) {
	c.mu.Lock()
	for id, val := range state.Components {
		if cur, ok := c.components[id]; !ok || val > cur {
			c.components[id] = val
		}
		if curSeq, ok := c.lastSeq[id]; !ok || val > curSeq {
			c.lastSeq[id] = val
		}
	}
	c.mu.Unlock()
}

func (c *Counter) Snapshot() types.FullState {
	c.mu.RLock()
	cp := make(map[string]int64, len(c.components))
	for k, v := range c.components {
		cp[k] = v
	}
	c.mu.RUnlock()
	return types.FullState{Components: cp}
}

func (c *Counter) Counts() (local, global int64) {
	c.mu.RLock()
	local = c.components[c.selfID]
	for _, v := range c.components {
		global += v
	}
	c.mu.RUnlock()
	return
}

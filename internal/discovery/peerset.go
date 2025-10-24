package discovery

import (
	"counter/internal/types"
	"sort"
	"sync"
	"time"
)

type peerEntry struct {
	ID       string
	Addr     string
	LastSeen time.Time
}

type PeerSet struct {
	mu    sync.RWMutex
	self  types.PeerInfo
	peers map[string]peerEntry
}

func NewPeerSet(self types.PeerInfo) *PeerSet {
	return &PeerSet{self: self, peers: make(map[string]peerEntry)}
}

func (ps *PeerSet) List() []types.PeerInfo {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	res := make([]types.PeerInfo, 0, len(ps.peers)+1)
	for _, p := range ps.peers {
		res = append(res, types.PeerInfo{ID: p.ID, Addr: p.Addr})
	}
	res = append(res, ps.self)
	sort.Slice(res, func(i, j int) bool { return res[i].ID < res[j].ID })
	return res
}

func (ps *PeerSet) Upsert(id, addr string) {
	if id == ps.self.ID {
		return
	}
	ps.mu.Lock()
	ps.peers[id] = peerEntry{ID: id, Addr: addr, LastSeen: time.Now()}
	ps.mu.Unlock()
}

func (ps *PeerSet) Heartbeat(id string) {
	if id == ps.self.ID {
		return
	}
	ps.mu.Lock()
	if p, ok := ps.peers[id]; ok {
		p.LastSeen = time.Now()
		ps.peers[id] = p
	}
	ps.mu.Unlock()
}

func (ps *PeerSet) RemoveStale(olderThan time.Duration) []string {
	deadline := time.Now().Add(-olderThan)
	ps.mu.Lock()
	removed := make([]string, 0)
	for id, p := range ps.peers {
		if p.LastSeen.IsZero() || p.LastSeen.Before(deadline) {
			delete(ps.peers, id)
			removed = append(removed, id)
		}
	}
	ps.mu.Unlock()
	return removed
}

func (ps *PeerSet) Self() types.PeerInfo { return ps.self }

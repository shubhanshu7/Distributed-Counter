package discovery_test

import (
	"counter/internal/discovery"
	"counter/internal/types"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPeerSet_ListAndUpsert(t *testing.T) {
	self := types.PeerInfo{ID: "self", Addr: "http://localhost:8081"}
	ps := discovery.NewPeerSet(self)

	ps.Upsert("p1", "http://localhost:8082")
	ps.Upsert("p2", "http://localhost:8083")
	list := ps.List()

	require.Len(t, list, 3)
	seen := map[string]bool{}
	for _, p := range list {
		seen[p.ID] = true
	}
	require.True(t, seen["self"])
	require.True(t, seen["p1"])
	require.True(t, seen["p2"])
}

func TestPeerSet_HeartbeatAndRemoveStale(t *testing.T) {
	self := types.PeerInfo{ID: "self", Addr: "http://localhost:8081"}
	ps := discovery.NewPeerSet(self)

	ps.Upsert("p1", "http://localhost:8082")
	ps.Heartbeat("p1")

	removed := ps.RemoveStale(50 * time.Millisecond)
	require.Empty(t, removed)

	time.Sleep(60 * time.Millisecond)

	removed = ps.RemoveStale(50 * time.Millisecond)
	require.Contains(t, removed, "p1")
}

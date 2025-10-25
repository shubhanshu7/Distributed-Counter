package counter_test

import (
	"counter/internal/counter"
	"counter/internal/types"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLocalIncrement_Concurrent(t *testing.T) {
	c := counter.New("self")
	const N = 200

	var wg sync.WaitGroup
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func() {
			defer wg.Done()
			c.LocalIncrement()
		}()
	}
	wg.Wait()

	local, global := c.Counts()
	require.Equal(t, int64(N), local, "local must equal number of increments")
	require.Equal(t, local, global, "global equals local when only one node")
}

func TestApply_DedupAndMaxMerge(t *testing.T) {
	c := counter.New("self")

	c.LocalIncrement()
	_, baseGlobal := c.Counts()

	msg := types.ApplyMsg{NodeID: "peerA", Seq: 1, Value: 1}
	c.Apply(msg)

	c.Apply(msg)

	_, g1 := c.Counts()
	require.Equal(t, baseGlobal+1, g1, "duplicate apply must be ignored")

	c.Apply(types.ApplyMsg{NodeID: "peerA", Seq: 2, Value: 5})
	_, g2 := c.Counts()
	require.Equal(t, baseGlobal+5, g2, "merge must take max value for sender component")
}

func TestMergeFullState(t *testing.T) {
	c := counter.New("self")

	for i := 0; i < 3; i++ {
		c.LocalIncrement()
	}
	local, global := c.Counts()
	require.Equal(t, int64(3), local)
	require.Equal(t, int64(3), global)

	state := types.FullState{
		Components: map[string]int64{
			"self":  3, // same
			"peer1": 7, // new peer component
			"peer2": 0, // zero is fine
		},
	}
	c.MergeFullState(state)

	l2, g2 := c.Counts()
	require.Equal(t, int64(3), l2)
	require.Equal(t, int64(3+7+0), g2)
}

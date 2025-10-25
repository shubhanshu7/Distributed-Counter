package transport_test

import (
	"counter/internal/discovery"
	"counter/internal/transport"
	"counter/internal/types"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSender_RetryAndFlush(t *testing.T) {
	var hits int32
	failTimes := int32(2)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/heartbeat" {
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.URL.Path == "/apply" {
			atomic.AddInt32(&hits, 1)
			if atomic.LoadInt32(&hits) <= failTimes {
				http.Error(w, "boom", http.StatusInternalServerError)
				return
			}
			var m types.ApplyMsg
			_ = json.NewDecoder(r.Body).Decode(&m)
			w.WriteHeader(http.StatusOK)
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	self := types.PeerInfo{ID: "self", Addr: "http://localhost:0"}
	ps := discovery.NewPeerSet(self)
	ps.Upsert("peer1", ts.URL)

	s := transport.NewSender(self, ps)
	s.Run()
	defer s.Stop()

	s.EnqueueBroadcast(types.ApplyMsg{NodeID: "self", Seq: 1, Value: 1})

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if atomic.LoadInt32(&hits) > failTimes {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	require.Greater(t, atomic.LoadInt32(&hits), failTimes, "should have retried and eventually succeeded")

	time.Sleep(100 * time.Millisecond)
}

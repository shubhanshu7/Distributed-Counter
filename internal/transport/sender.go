package transport

import (
	"bytes"
	"counter/internal/discovery"
	"counter/internal/types"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

type Sender struct {
	client   *http.Client
	peers    *discovery.PeerSet
	self     types.PeerInfo
	queueMu  sync.Mutex
	pending  map[string][]types.ApplyMsg
	stopping chan struct{}
}

func NewSender(self types.PeerInfo, peers *discovery.PeerSet) *Sender {
	return &Sender{
		client:   &http.Client{Timeout: 2 * time.Second},
		self:     self,
		peers:    peers,
		pending:  make(map[string][]types.ApplyMsg),
		stopping: make(chan struct{}),
	}
}

func (s *Sender) EnqueueBroadcast(msg types.ApplyMsg) {
	list := s.peers.List()
	for _, p := range list {
		if p.ID == s.self.ID {
			continue
		}
		s.queueMu.Lock()
		s.pending[p.ID] = append(s.pending[p.ID], msg)
		s.queueMu.Unlock()
	}
}

func (s *Sender) Run() {
	go func() {
		t := time.NewTicker(300 * time.Millisecond)
		defer t.Stop()
		for {
			select {
			case <-s.stopping:
				return
			case <-t.C:
				s.flushOnce()
			}
		}
	}()

	go func() { // heartbeat loop
		t := time.NewTicker(2 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-s.stopping:
				return
			case <-t.C:
				s.heartbeatOnce()
			}
		}
	}()

	go func() { // prune loop
		t := time.NewTicker(3 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-s.stopping:
				return
			case <-t.C:
				removed := s.peers.RemoveStale(6 * time.Second)
				if len(removed) > 0 {
					log.Printf("pruned stale peers: %v", removed)
				}
			}
		}
	}()
}

func (s *Sender) Stop() { close(s.stopping) }

func (s *Sender) heartbeatOnce() {
	list := s.peers.List()
	for _, p := range list {
		if p.ID == s.self.ID {
			continue
		}
		go func(p types.PeerInfo) {
			req := types.HeartbeatRequest{NodeID: s.self.ID}
			_ = s.postJSON(p.Addr+"/heartbeat", req, nil)
		}(p)
	}
}

func (s *Sender) flushOnce() {
	list := s.peers.List()
	for _, p := range list {
		if p.ID == s.self.ID {
			continue
		}
		s.queueMu.Lock()
		msgs := s.pending[p.ID]
		s.pending[p.ID] = nil
		s.queueMu.Unlock()
		if len(msgs) == 0 {
			continue
		}
		last := msgs[len(msgs)-1]
		bo := newBackoff(100*time.Millisecond, 2.0, 2*time.Second, 5)
		for {
			if err := s.postJSON(p.Addr+"/apply", last, nil); err != nil {
				d, retry := bo.Next()
				if !retry {
					log.Printf("apply to %s failed permanently: %v", p.ID, err)
					break
				}
				time.Sleep(d)
				continue
			}
			break
		}
	}
}

func (s *Sender) postJSON(url string, body any, out any) error {
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, url, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

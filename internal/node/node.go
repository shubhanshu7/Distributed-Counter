package node

import (
	"bytes"
	"counter/internal/counter"
	"counter/internal/discovery"
	"counter/internal/transport"
	"counter/internal/types"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"
)

type Node struct {
	self    types.PeerInfo
	peers   *discovery.PeerSet
	counter *counter.Counter
	sender  *transport.Sender
}

func NewNode(self types.PeerInfo) *Node {
	ps := discovery.NewPeerSet(self)
	c := counter.New(self.ID)
	s := transport.NewSender(self, ps)
	return &Node{self: self, peers: ps, counter: c, sender: s}
}

func (n *Node) StartHTTPServer(addr string) {
	http.HandleFunc("/join", n.handleJoin)
	http.HandleFunc("/heartbeat", n.handleHeartbeat)
	http.HandleFunc("/peers", n.handlePeers)

	http.HandleFunc("/increment", n.handleIncrement)
	http.HandleFunc("/count", n.handleCount)
	http.HandleFunc("/apply", n.handleApply)
	http.HandleFunc("/state", n.handleState)

	go n.sender.Run()
	log.Printf("node %s listening on %s (public %s)", n.self.ID, addr, n.self.Addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

// --- Discovery handlers ---

func (n *Node) handleJoin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req types.JoinRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	n.peers.Upsert(req.NodeID, req.Addr)
	resp := types.JoinResponse{Peers: n.peers.List()}
	writeJSON(w, resp)
}

func (n *Node) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req types.HeartbeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	n.peers.Heartbeat(req.NodeID)
	w.WriteHeader(http.StatusOK)
}

func (n *Node) handlePeers(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, types.PeersResponse{Peers: n.peers.List()})
}

// --- Counter handlers ---

func (n *Node) handleIncrement(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	local, global, msg := n.counter.LocalIncrement()
	n.sender.EnqueueBroadcast(msg)
	writeJSON(w, types.CountResponse{Local: local, Global: global})
}

func (n *Node) handleCount(w http.ResponseWriter, r *http.Request) {
	local, global := n.counter.Counts()
	writeJSON(w, types.CountResponse{Local: local, Global: global})
}

func (n *Node) handleApply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var msg types.ApplyMsg
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	n.counter.Apply(msg)
	w.WriteHeader(http.StatusOK)
}

func (n *Node) handleState(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, n.counter.Snapshot())
}

// Bootstrap (join & initial sync)
func (n *Node) Bootstrap(peersCSV, httpAddr string) {
	client := &http.Client{Timeout: 2 * time.Second}
	peers := parsePeers(peersCSV)
	for _, p := range peers {
		if p == "" {
			continue
		}
		jr := types.JoinRequest{NodeID: n.self.ID, Addr: n.self.Addr}
		if !strings.HasPrefix(p, "http") {
			p = "http://" + p
		}
		if err := post(p+"/join", jr, client, nil); err == nil {
			var pr types.PeersResponse
			_ = get(p+"/peers", client, &pr)
			for _, pi := range pr.Peers {
				n.peers.Upsert(pi.ID, pi.Addr)
			}
			var st types.FullState
			if err := get(p+"/state", client, &st); err == nil {
				n.counter.MergeFullState(st)
			}
		} else {
			log.Printf("bootstrap join to %s failed: %v", p, err)
		}
	}
}

func parsePeers(csv string) []string {
	csv = strings.TrimSpace(csv)
	if csv == "" {
		return nil
	}
	parts := strings.Split(csv, ",")
	sort.Strings(parts)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		out = append(out, strings.TrimSpace(p))
	}
	return out
}

func post(url string, body any, client *http.Client, out any) error {
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, url, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
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

func get(url string, client *http.Client, out any) error {
	resp, err := client.Get(url)
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

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

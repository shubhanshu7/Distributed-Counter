package types

type JoinRequest struct {
	NodeID string `json:"node_id"`
	Addr   string `json:"addr"`
}

type JoinResponse struct {
	Peers []PeerInfo `json:"peers"`
}

type PeerInfo struct {
	ID   string `json:"id"`
	Addr string `json:"addr"`
}

type HeartbeatRequest struct {
	NodeID string `json:"node_id"`
}

type PeersResponse struct {
	Peers []PeerInfo `json:"peers"`
}

type CountResponse struct {
	Local  int64 `json:"local"`
	Global int64 `json:"global"`
}

type ApplyMsg struct {
	NodeID string `json:"node_id"`
	Seq    int64  `json:"seq"`
	Value  int64  `json:"value"`
}

type FullState struct {
	Components map[string]int64 `json:"components"`
}

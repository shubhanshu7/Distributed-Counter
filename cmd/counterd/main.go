package main

import (
	"counter/internal/node"
	"counter/internal/types"
	"flag"
	"fmt"
	"os"
)

func main() {
	addr := flag.String("addr", ":8081", "listen to address :8081")
	id := flag.String("id", "", "node ID (defaults to host:port)")
	peers := flag.String("peers", "", "comma-separated peer addrs (host:port or http://host:port)")
	public := flag.String("public-addr", "", "publicly reachable base URL (default http://localhost:PORT)")
	flag.Parse()

	if *id == "" {
		host, _ := os.Hostname()
		*id = fmt.Sprintf("%s%s", host, *addr)
	}

	pub := *public
	if pub == "" {
		pub = fmt.Sprintf("http://localhost%s", *addr)
	}

	self := types.PeerInfo{ID: *id, Addr: pub}

	n := node.NewNode(self)
	n.Bootstrap(*peers, *addr)
	n.StartHTTPServer(*addr)
}

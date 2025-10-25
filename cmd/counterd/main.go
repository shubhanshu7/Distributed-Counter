package main

import (
	"counter/internal/node"
	"counter/internal/types"
	"flag"
	"fmt"
	"os"
)

func main() {
	addr := flag.String("addr", ":8081", "default to address :8081")
	peers := flag.String("peers", "", "example of using the flag-  http://host:port)")

	flag.Parse()

	host, err := os.Hostname()
	if err != nil {
		fmt.Println("Error getting hostname ", err)
		return
	}
	id := fmt.Sprintf("%s%s", host, *addr)

	pub := fmt.Sprintf("http://localhost%s", *addr)

	self := types.PeerInfo{ID: id, Addr: pub}

	n := node.NewNode(self)
	n.Bootstrap(*peers, *addr)
	n.StartHTTPServer(*addr)
}

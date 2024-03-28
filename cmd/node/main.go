package main

import (
	"fmt"

	"github.com/webbben/p2p-file-share/internal/peer"
)

func main() {

	peers := peer.DiscoverPeers()
	fmt.Println("Peers discovered:")
	for _, ip := range peers {
		fmt.Println(ip)
	}

	peer.ListenForHandshakes()
}

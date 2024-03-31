package main

import (
	"fmt"

	"github.com/webbben/p2p-file-share/internal/network"
	"github.com/webbben/p2p-file-share/internal/peer"
)

func main() {
	ip := network.GetLocalIP()
	if ip != "" {
		fmt.Println("Local Machine IP:", ip)
	}
	peers := peer.DiscoverPeers()
	fmt.Println("Peers discovered:")
	for _, ip := range peers {
		fmt.Println(ip)
	}

	peer.ListenForHandshakes()
}

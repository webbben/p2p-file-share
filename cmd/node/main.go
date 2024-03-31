package main

import (
	"fmt"
	"time"

	"github.com/webbben/p2p-file-share/internal/network"
	"github.com/webbben/p2p-file-share/internal/peer"
)

func main() {
	go peer.ListenForHandshakes()
	ip := network.GetLocalIP()
	if ip != "" {
		fmt.Println("Local Machine IP:", ip)
	}

	// search for peers every minute
	for {
		peers := peer.DiscoverPeers()
		fmt.Println("Peers discovered:")
		for _, ip := range peers {
			fmt.Println(ip)
		}
		fmt.Println("(sleeping for 1 minute)")
		time.Sleep(1 * time.Minute)
	}
}

package main

import (
	"fmt"
	"os"
	"time"

	c "github.com/webbben/p2p-file-share/internal/config"
	"github.com/webbben/p2p-file-share/internal/network"
	"github.com/webbben/p2p-file-share/internal/peer"
	"github.com/webbben/p2p-file-share/internal/ui"
)

func main() {
	// run this in the background indefinitely, so this node is always discoverable
	go peer.ListenForHandshakes()
	time.Sleep(1 * time.Second)

	ip := network.GetLocalIP()
	if ip != "" {
		fmt.Println("Node IP:", ip)
	}
	// handle config
	config := c.LoadConfig()
	if config == nil {
		fmt.Println("No config found")
		if ui.YorN("Initialize new config?") {
			config = c.NewConfigWorkflow()
			c.InitializeConfigFile(*config)
		} else {
			fmt.Println("No config created; you will be unable to use this application node until a valid config has been created.")
			os.Exit(0)
			return
		}
	}
	fmt.Println("Node nickname:", config.Nickname)
	fmt.Println("Fileshare directory:", config.SharedDirectoryPath)

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

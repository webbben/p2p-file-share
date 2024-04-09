package main

import (
	"fmt"
	"os"
	"time"

	c "github.com/webbben/p2p-file-share/internal/config"
	"github.com/webbben/p2p-file-share/internal/network"
	"github.com/webbben/p2p-file-share/internal/peer"
	"github.com/webbben/p2p-file-share/internal/server"
	"github.com/webbben/p2p-file-share/internal/state"
	"github.com/webbben/p2p-file-share/internal/syncdir"
	"github.com/webbben/p2p-file-share/internal/ui"
)

func main() {
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

	// get an initial set of peers
	peers := peer.DiscoverPeers()
	state.SetPeers(peers)

	// start the message server to handle incoming connections from peers
	go server.MessageServer(*config)
	// watch for changes to the shared file directory
	go syncdir.WatchForFileChanges(config.SharedDirectoryPath)

	for {
		time.Sleep(1 * time.Minute)
		peers = peer.DiscoverPeers()
		state.SetPeers(peers)
	}
}

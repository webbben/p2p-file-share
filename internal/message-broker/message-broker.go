// functions for general communication between nodes (excluding file transfers)
package messagebroker

import (
	"encoding/json"
	"fmt"
	"net"
	"time"

	c "github.com/webbben/p2p-file-share/internal/config"
	m "github.com/webbben/p2p-file-share/internal/model"
	"github.com/webbben/p2p-file-share/internal/peer"
	"github.com/webbben/p2p-file-share/internal/state"
)

// broadcasts a message to all known peers
func BroadcastMessage(msg interface{}) {
	// make sure there are peers to broadcast to
	peers := state.GetPeers()
	if len(peers) == 0 {
		peers = peer.DiscoverPeers()
		state.SetPeers(peers)
	} else if state.PeerDataIsStale() {
		peers = peer.DiscoverPeers()
		state.SetPeers(peers)
	}
	for _, p := range peers {
		if err := sendSimplexMessage(p, msg); err != nil {
			fmt.Println("Failed to send message to peer;", err, "; peer info:", p)
			continue
		}
	}
}

// sends a message to a peer without expecting a response
func sendSimplexMessage(p m.Peer, msg interface{}) error {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%v", p.IP, c.PORT), time.Millisecond*time.Duration(c.MESSAGE_TIMEOUT_MS))
	if err != nil {
		return err
	}
	defer conn.Close()
	jsonData, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = conn.Write(jsonData)
	return err
}

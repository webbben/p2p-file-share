// functions for general communication between nodes (excluding file transfers)
package messagebroker

import (
	"encoding/json"
	"fmt"
	"net"
	"time"

	c "github.com/webbben/p2p-file-share/internal/config"
	m "github.com/webbben/p2p-file-share/internal/model"
	"github.com/webbben/p2p-file-share/internal/network"
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

// gets the file information from a node
func scanFiles(p m.Peer) ([]m.FileInfo, error) {
	buf, err := sendDuplexMessage(p, m.MiscMessage{Type: c.TYPE_SCAN_FILES})
	if err != nil {
		return nil, err
	}
	// we expect a summary of the file info in response
	var fileSummary m.NodeFileSummary
	if err = json.Unmarshal(buf, &fileSummary); err != nil {
		return nil, err
	}
	return fileSummary.Files, nil
}

// sends a message that expects a response from the peer
func sendDuplexMessage(p m.Peer, msg interface{}) ([]byte, error) {
	conn, err := net.DialTimeout("tcp", network.FormatSocketAddr(p.IP, c.PORT), time.Millisecond*time.Duration(c.MESSAGE_TIMEOUT_MS))
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	bytes, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}
	_, err = conn.Write(bytes)
	if err != nil {
		return nil, err
	}
	buf, err := network.ReadBuffer(conn, 1024)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

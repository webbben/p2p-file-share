// maintains the global state of the list of peers, since multiple packages will need to access this
package state

import (
	"fmt"
	"sync"
	"time"

	m "github.com/webbben/p2p-file-share/internal/model"
)

type State struct {
	HistoricPeersList map[string]time.Time
	CurrentPeersList  []m.Peer
	LastPeerSearch    time.Time
}

var (
	state      *State = &State{}
	stateMutex sync.Mutex
)

// sets the current peers list
func SetPeers(peers []m.Peer) {
	stateMutex.Lock()
	defer stateMutex.Unlock()

	now := time.Now().UTC()
	for _, peer := range peers {
		fmt.Println(peer)
		state.HistoricPeersList[peer.IP] = now
	}
	state.CurrentPeersList = peers
	state.LastPeerSearch = now
}

// adds a single peer to the peer state (not for batch updates)
func AddPeer(peer m.Peer) {
	stateMutex.Lock()
	defer stateMutex.Unlock()

	state.HistoricPeersList[peer.IP] = time.Now().UTC()
	for _, p := range state.CurrentPeersList {
		if p.IP == peer.IP {
			return // peer is already in the list
		}
	}
	state.CurrentPeersList = append(state.CurrentPeersList, peer)
}

// getst he current list of peers
func GetPeers() []m.Peer {
	return state.CurrentPeersList
}

// gets a map of all peers this node has discovered
func GetHistoricalPeers() map[string]time.Time {
	return state.HistoricPeersList
}

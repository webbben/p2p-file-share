package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"time"

	c "github.com/webbben/p2p-file-share/internal/config"
	filetransfer "github.com/webbben/p2p-file-share/internal/file-transfer"
	m "github.com/webbben/p2p-file-share/internal/model"
	"github.com/webbben/p2p-file-share/internal/peer"
	"github.com/webbben/p2p-file-share/internal/state"
)

// starts a server for TCP-based messages, and routes incoming messages to their correct functionality.
func MessageServer(config c.Config) {
	port := c.PORT
	server, err := net.Listen("tcp", fmt.Sprintf(":%v", port))
	if err != nil {
		fmt.Println("Error starting message server:", err)
		return
	}
	defer server.Close()

	log.Printf("TCP server listening on port %v\n", port)

	// accept and route incoming connections
	for {
		conn, err := server.Accept()
		if err != nil {
			log.Println("Error accepting connection:", err)
			continue
		}
		handleConnection(conn, config)
	}
}

// route the incoming connection based on its type and purpose
func handleConnection(conn net.Conn, config c.Config) {
	defer conn.Close()

	// read incoming data
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		log.Println("Error reading connection:", err)
		return
	}
	var msg map[string]interface{}
	if err := json.Unmarshal(buf[:n], &msg); err != nil {
		log.Println("Error parsing message:", err)
		return
	}

	// route by message type
	messageType, ok := msg["type"].(string)
	if !ok {
		log.Println("Error handling connection: Message missing type property")
		return
	}
	switch messageType {
	case c.TYPE_DISCOVER_PEER:
		var structMsg m.Handshake
		if err := mapToStruct(msg, &structMsg); err != nil {
			fmt.Println("error decoding handshake data:", err)
			return
		}
		peer.RespondToHandshake(conn, structMsg, config)
	case c.TYPE_FILE_REQUEST:
		var structMsg m.FileRequest
		if err := mapToStruct(msg, &structMsg); err != nil {
			fmt.Println("error decoding file request data:", err)
			return
		}
		filetransfer.SendFile(conn, structMsg.File)
	}
}

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

// converts the raw json data we read from the TCP connection to the actual data type we want
func mapToStruct(rawData map[string]interface{}, outStruct interface{}) error {
	jsonData, err := json.Marshal(rawData)
	if err != nil {
		return err
	}
	return json.Unmarshal(jsonData, outStruct)
}

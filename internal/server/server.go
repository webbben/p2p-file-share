package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net"

	"github.com/webbben/p2p-file-share/internal/config"
	filetransfer "github.com/webbben/p2p-file-share/internal/file-transfer"
	"github.com/webbben/p2p-file-share/internal/peer"
)

// starts a server for TCP-based messages, and routes incoming messages to their correct functionality.
func MessageServer() {
	port := config.PORT
	server, err := net.Listen("tcp", fmt.Sprintf(":%v", port))
	if err != nil {
		fmt.Println("Error starting message server:", err)
		return
	}
	server.Close()

	log.Printf("TCP server listening on port %v\n", port)

	// accept and route incoming connections
	for {
		conn, err := server.Accept()
		if err != nil {
			log.Println("Error accepting connection:", err)
			continue
		}
		handleConnection(conn)
	}
}

// route the incoming connection based on its type and purpose
func handleConnection(conn net.Conn) {
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
	case config.TYPE_DISCOVER_PEER:
		peer.RespondToHandshake(conn)
	case config.TYPE_FILE_REQUEST:
		filePath := msg["file"].(string)
		filetransfer.SendFile(conn, filePath)
	}
}

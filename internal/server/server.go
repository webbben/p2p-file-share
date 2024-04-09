package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net"

	c "github.com/webbben/p2p-file-share/internal/config"
	filetransfer "github.com/webbben/p2p-file-share/internal/file-transfer"
	m "github.com/webbben/p2p-file-share/internal/model"
	"github.com/webbben/p2p-file-share/internal/peer"
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

func BroadcastMessage(msg map[string]interface{}) {

}

// converts the raw json data we read from the TCP connection to the actual data type we want
func mapToStruct(rawData map[string]interface{}, outStruct interface{}) error {
	jsonData, err := json.Marshal(rawData)
	if err != nil {
		return err
	}
	return json.Unmarshal(jsonData, outStruct)
}

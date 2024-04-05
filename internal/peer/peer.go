package peer

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/webbben/p2p-file-share/internal/config"
	"github.com/webbben/p2p-file-share/internal/model"
	"github.com/webbben/p2p-file-share/internal/network"
)

const (
	startIP = 1
	endIP   = 255
	timeout = 500 * time.Millisecond
	port    = "8080"
)

var discoveredPeers []string
var mutex sync.Mutex

func DiscoverPeers() []string {
	localSubnet := network.GetLocalSubnetBase()
	myIP := network.GetLocalIP()
	fmt.Println("searching for peers on local subnet:", localSubnet)
	var wg sync.WaitGroup
	discoveredPeers = []string{}
	for i := startIP; i <= endIP; i++ {
		ip := fmt.Sprintf("%v.%v", localSubnet, i)
		if ip == myIP {
			continue
		}
		wg.Add(1)
		go func() {
			scanIP(ip)
			wg.Done()
		}()
	}

	wg.Wait()
	return discoveredPeers
}

func scanIP(ip string) {
	addr := net.JoinHostPort(ip, port)
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return // connection failed
	}
	defer conn.Close()

	// peer discovered
	fmt.Printf("Peer candidate: %s\n", ip)
	if crispHandshake(conn) {
		mutex.Lock()
		discoveredPeers = append(discoveredPeers, ip)
		mutex.Unlock()
	}
}

// exchange a crisp handshake with the IP to confirm that they are, in fact, your homie (peer)
func crispHandshake(conn net.Conn) bool {
	// send a handshake that just includes this nodes IP address
	localAddr := conn.LocalAddr().String()
	handshakeJson, err := json.Marshal(model.Handshake{
		Type: config.TYPE_DISCOVER_PEER,
		Data: localAddr,
	})
	if err != nil {
		fmt.Println(err)
		return false
	}
	conn.Write(handshakeJson)

	// wait for a response, or the timeout
	handshakeTimeout := time.After(5 * time.Second)
	select {
	case <-handshakeTimeout:
		fmt.Println("handshake timed out: peer didn't respond")
		return false
	default:
		// receive peer info
		scanner := bufio.NewScanner(conn)
		fmt.Println("waiting for response...")
		for scanner.Scan() {
			fmt.Println("ope")
			respBytes := scanner.Bytes()
			fmt.Println("got the bytes")
			var respJson model.Handshake
			err = json.Unmarshal(respBytes, &respJson)
			if err != nil {
				fmt.Println(err)
				return false
			}
			fmt.Printf("Peer info: %s\n", respJson.Data)
			return true
		}
	}
	return false
}

// listen for other peers who are looking to discover this node, and shake its hand
func ListenForHandshakes() {
	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		fmt.Println("Failed to start listener:", err)
		return
	}
	defer ln.Close()

	fmt.Printf("Listening for peer connections on port %s...\n", port)

	// dap up any homies (peers) that send handshakes
	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Failed to accept connection:", err)
			continue
		}
		go RespondToHandshake(conn)
	}
}

func RespondToHandshake(conn net.Conn) {
	defer conn.Close()
	fmt.Println("responding to handshake")

	// read peer info from handshake
	buf, err := network.ReadBuffer(conn, 1024)
	if err != nil {
		fmt.Println("error reading handshake buffer")
		return
	}
	var handshakeJson model.Handshake
	err = json.Unmarshal(buf, &handshakeJson)
	if err != nil {
		fmt.Println("error parsing handshake:", err)
		return
	}
	fmt.Println("Peer info:", handshakeJson.Data)

	// send back handshake response
	bytes, err := json.Marshal(model.Handshake{Type: config.TYPE_DISCOVER_PEER, Data: conn.RemoteAddr().String()})
	if err != nil {
		fmt.Println(err)
		return
	}
	conn.Write(bytes)
}

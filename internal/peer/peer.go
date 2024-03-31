package peer

import (
	"bufio"
	"fmt"
	"net"
	"sync"
	"time"

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
	fmt.Println("searching for peers on local subnet:", localSubnet)
	var wg sync.WaitGroup
	discoveredPeers = []string{}
	for i := startIP; i <= endIP; i++ {
		ip := fmt.Sprintf("%v.%v", localSubnet, i)
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
	localAddr := conn.LocalAddr().String()
	fmt.Fprintf(conn, "%s\n", localAddr)

	// set a timeout
	timeoutDur := 5 * time.Second
	handshakeTimeout := time.After(timeoutDur)

	// wait for a response, or the timeout
	select {
	case <-handshakeTimeout:
		fmt.Println("handshake timed out: peer didn't respond")
		return false
	default:
		// receive peer info
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			peerAddr := scanner.Text()
			fmt.Printf("Peer info: %s\n", peerAddr)
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
		go respondToHandshake(conn)
	}
}

func respondToHandshake(conn net.Conn) {
	defer conn.Close()

	// read peer info
	scanner := bufio.NewScanner(conn)
	scanner.Scan()
	peerAddr := scanner.Text()
	fmt.Println("Peer info:", peerAddr)

	// echo
	fmt.Fprintf(conn, "%s\n", conn.RemoteAddr())
}

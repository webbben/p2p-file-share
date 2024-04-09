package peer

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	c "github.com/webbben/p2p-file-share/internal/config"
	m "github.com/webbben/p2p-file-share/internal/model"
	"github.com/webbben/p2p-file-share/internal/network"
	"github.com/webbben/p2p-file-share/internal/state"
)

const (
	startIP = 1
	endIP   = 255
	timeout = 500 * time.Millisecond
	port    = "8080"
)

var discoveredPeers []m.Peer
var mutex sync.Mutex

func DiscoverPeers() []m.Peer {
	localSubnet := network.GetLocalSubnetBase()
	myIP := network.GetLocalIP()
	fmt.Println("searching for peers on local subnet:", localSubnet)
	var wg sync.WaitGroup
	discoveredPeers = []m.Peer{}
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
	fmt.Println("Number of peers found:", len(discoveredPeers))
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
	if success, p := crispHandshake(conn, ip); success {
		mutex.Lock()
		discoveredPeers = append(discoveredPeers, p)
		mutex.Unlock()
	}
}

// exchange a crisp handshake with the IP to confirm that they are, in fact, your homie (peer)
func crispHandshake(conn net.Conn, ip string) (bool, m.Peer) {
	// send a handshake that just includes this nodes IP address
	localAddr := conn.LocalAddr().String()
	handshakeJson, err := json.Marshal(m.Handshake{
		Type: c.TYPE_DISCOVER_PEER,
		Data: localAddr,
	})
	if err != nil {
		fmt.Println(err)
		return false, m.Peer{}
	}
	conn.Write(handshakeJson)

	// wait for a response, or timeout
	conn.SetDeadline(time.Now().Add(time.Second * 5))
	buf, err := network.ReadBuffer(conn, 1024)
	if err != nil {
		fmt.Println("error reading handshake response:", err)
		return false, m.Peer{}
	}
	var respJson m.Handshake
	err = json.Unmarshal(buf, &respJson)
	if err != nil {
		fmt.Println(err)
		return false, m.Peer{}
	}
	fmt.Printf("Peer response: %s\n", respJson.Data)
	// TODO: validate data in some way, to further authenticate peer node?
	return true, m.Peer{
		IP:       ip,
		Nickname: respJson.Nickname,
	}
}

func RespondToHandshake(conn net.Conn, handshakeData m.Handshake, config c.Config) {
	defer conn.Close()
	fmt.Println("responding to handshake")

	// send back handshake response
	remoteAddr := conn.RemoteAddr().String()
	remoteIP := strings.Split(remoteAddr, ":")[0] // remove the port number from the address
	bytes, err := json.Marshal(m.Handshake{
		Type:     c.TYPE_DISCOVER_PEER,
		Data:     remoteIP,        // echo their IP back, to confirm we are a legit node
		Nickname: config.Nickname, // send nickname of this node
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	conn.Write(bytes)

	// add this IP to this nodes peer list
	state.AddPeer(m.Peer{
		IP:       remoteIP,
		Nickname: handshakeData.Nickname,
	})
}

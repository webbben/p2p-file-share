package network

import (
	"fmt"
	"net"
	"strings"
)

// finds the IP address for this machine
func GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		fmt.Println("Failed to get local IP address:", err)
		return ""
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}

func GetLocalSubnetBase() string {
	localIP := GetLocalIP()
	if localIP == "" {
		return ""
	}
	lastIndex := strings.LastIndex(localIP, ".")
	if lastIndex == -1 {
		return "" // no periods found? so probably invalid IP found
	}
	return localIP[:lastIndex]
}

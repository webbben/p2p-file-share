package network

import (
	"errors"
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

// reads a buffer from a connection, detecting protocol-specified error messages at the same time
func ReadBuffer(conn net.Conn, bufferSize int) ([]byte, error) {
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		return []byte{}, errors.Join(errors.New("failed to read buffer"), err)
	}
	response := string(buf[:n])
	if strings.HasPrefix(response, "ERROR:") {
		errorMsg := strings.TrimPrefix(response, "ERROR:")
		return []byte{}, errors.New(errorMsg)
	}
	return buf[:n], nil
}
